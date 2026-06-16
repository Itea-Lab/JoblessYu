package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

type JobEntry struct {
	Title       string
	Company     string
	Location    string
	URL         string
	Description string
	Level       string // detected: Intern, Junior, Senior
	Type        string // detected: Fulltime, Parttime
}

var (
	jobsCache = map[string][]JobEntry{}
	cacheLock sync.RWMutex
	commands  = []*discordgo.ApplicationCommand{
		{
			Name:        "jobs",
			Description: "Browse IT Support jobs with pagination",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "level",
					Description: "Job experience level",
					Required:    true,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "Intern", Value: "Intern"},
						{Name: "Junior", Value: "Junior"},
						{Name: "Senior", Value: "Senior"},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "time",
					Description: "Job type (optional)",
					Required:    false,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "Fulltime", Value: "Fulltime"},
						{Name: "Parttime", Value: "Parttime"},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "location",
					Description: "Job location (optional)",
					Required:    false,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "HCM (Ho Chi Minh)", Value: "HCM"},
						{Name: "HN (Ha Noi)", Value: "HN"},
					},
				},
			},
		},
	}
)

func main() {
	// Load .env
	if err := godotenv.Load(); err != nil {
		log.Println("Note: .env file not found, using system env variables")
	}

	token := "Bot " + os.Getenv("DISCORD_BOT_TOKEN")
	dbURL := os.Getenv("DATABASE_URL")

	disbot, err := discordgo.New(token)
	if err != nil {
		log.Fatal("Error creating Discord session:", err)
	}

	disbot.Identify.Intents =
		discordgo.IntentsGuildMessages |
			discordgo.IntentsDirectMessages |
			discordgo.IntentsGuilds |
			discordgo.IntentsMessageContent

	disbot.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {

		// Ignore bot messages
		if m.Author.Bot {
			return
		}

		// Test embed command
		if m.Content == "!testembed" {
			embed := &discordgo.MessageEmbed{
				Title:       "✅ Embed Test",
				Description: "If you can see this, embeds are working correctly.",
				Color:       0x57F287,
			}

			_, err := s.ChannelMessageSendEmbed(m.ChannelID, embed)
			if err != nil {
				log.Println("Embed Error:", err)
			}
			return
		}

		// Removed text-based !jobs support; use the /jobs slash command instead.
	})

	disbot.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type == discordgo.InteractionApplicationCommand {
			if i.ApplicationCommandData().Name == "jobs" {
				handleJobsSlash(s, i, dbURL)
			}
			return
		}

		if i.Type != discordgo.InteractionMessageComponent {
			return
		}

		if i.Message == nil || i.Message.ID == "" {
			return
		}

		cacheLock.RLock()
		jobs, ok := jobsCache[i.Message.ID]
		cacheLock.RUnlock()
		if !ok || len(jobs) == 0 {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "⚠️ This job list has expired. Please run `/jobs` again.",
					Flags:   1 << 6,
				},
			})
			return
		}

		page := 1
		if i.Message.Embeds != nil && len(i.Message.Embeds) > 0 && i.Message.Embeds[0].Footer != nil {
			fmt.Sscanf(i.Message.Embeds[0].Footer.Text, "Page %d", &page)
			if page < 1 {
				page = 1
			}
		}

		switch i.MessageComponentData().CustomID {
		case "job_page_prev":
			if page > 1 {
				page--
			}
		case "job_page_next":
			if page < len(jobs) {
				page++
			}
		default:
			return
		}

		embed := buildJobEmbed(jobs[page-1], page, len(jobs))
		components := []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "⬅️ Prev",
						Style:    discordgo.SecondaryButton,
						CustomID: "job_page_prev",
						Disabled: page == 1,
					},
					discordgo.Button{
						Label:    "Next ➡️",
						Style:    discordgo.SecondaryButton,
						CustomID: "job_page_next",
						Disabled: page == len(jobs),
					},
				},
			},
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Embeds:     []*discordgo.MessageEmbed{embed},
				Components: components,
			},
		})
	})

	if err := disbot.Open(); err != nil {
		log.Fatal("Error opening connection:", err)
	}

	guildID := os.Getenv("DISCORD_GUILD_ID")
	for _, cmd := range commands {
		existing, err := disbot.ApplicationCommands(disbot.State.User.ID, guildID)
		if err == nil {
			skip := false
			for _, existingCmd := range existing {
				if existingCmd.Name == cmd.Name {
					skip = true
					break
				}
			}
			if skip {
				continue
			}
		}
		_, err = disbot.ApplicationCommandCreate(disbot.State.User.ID, guildID, cmd)
		if err != nil {
			log.Println("Failed to register slash command:", err)
		}
	}

	fmt.Println("JoblessYu Vessel is now running. Press CTRL+C to exit.")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	fmt.Println("Shutting down...")
	disbot.Close()
}

func detectJobMeta(description string) (level, jobType string) {
	lower := strings.ToLower(description)

	switch {
	case strings.Contains(lower, "intern"):
		level = "Intern"
	case strings.Contains(lower, "junior"):
		level = "Junior"
	case strings.Contains(lower, "senior"):
		level = "Senior"
	}

	switch {
	case strings.Contains(lower, "fulltime"), strings.Contains(lower, "full-time"):
		jobType = "Fulltime"
	case strings.Contains(lower, "parttime"), strings.Contains(lower, "part-time"):
		jobType = "Parttime"
	}

	return
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "..."
}

func buildJobEmbed(job JobEntry, page, total int) *discordgo.MessageEmbed {
	desc := fmt.Sprintf("• **Role:** %s\n• **Company:** %s\n• **Location:** %s", job.Title, job.Company, job.Location)

	tags := ""
	if job.Level != "" {
		tags += fmt.Sprintf("`%s` ", job.Level)
	}
	if job.Type != "" {
		tags += fmt.Sprintf("`%s`", job.Type)
	}
	if tags != "" {
		desc += fmt.Sprintf("\n• **Tags:** %s", strings.TrimSpace(tags))
	}

	desc += fmt.Sprintf("\n• **Apply:** [Open job](%s)", job.URL)

	if job.Description != "" {
		desc += fmt.Sprintf("\n\n%s", truncate(job.Description, 300))
	}
	return &discordgo.MessageEmbed{
		Title:       job.Company,
		Description: desc,
		Color:       0x5865F2,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Page %d of %d", page, total),
		},
	}
}

func fetchJobs(dbURL string, level, jobType, location string) ([]JobEntry, error) {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		return nil, err
	}
	defer conn.Close(ctx)

	args := []interface{}{}
	argIdx := 1
	query := `SELECT COALESCE(title, ''), COALESCE(company, ''), COALESCE(location, ''), job_url, COALESCE(job_type, ''), COALESCE(description, '') FROM jobs`

	var conditions []string
	if level != "" {
		conditions = append(conditions, fmt.Sprintf(`(title ILIKE $%d OR description ILIKE $%d)`, argIdx, argIdx+1))
		args = append(args, "%"+level+"%", "%"+level+"%")
		argIdx += 2
	}
	if jobType != "" {
		conditions = append(conditions, fmt.Sprintf(`(job_type ILIKE $%d OR description ILIKE $%d)`, argIdx, argIdx+1))
		args = append(args, "%"+jobType+"%", "%"+jobType+"%")
		argIdx += 2
	}
	switch location {
	case "HCM":
		conditions = append(conditions, fmt.Sprintf(`(location ILIKE $%d OR location ILIKE $%d)`, argIdx, argIdx+1))
		args = append(args, "%HCM%", "%Ho Chi Minh%")
		argIdx += 2
	case "HN":
		conditions = append(conditions, fmt.Sprintf(`(location ILIKE $%d OR location ILIKE $%d OR location ILIKE $%d)`, argIdx, argIdx+1, argIdx+2))
		args = append(args, "%HN%", "%Ha Noi%", "%Hanoi%")
		argIdx += 3
	}

	if len(conditions) > 0 {
		query += ` WHERE ` + strings.Join(conditions, " AND ")
	}

	query += ` ORDER BY fetched_at DESC LIMIT 20`

	rows, err := conn.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []JobEntry
	for rows.Next() {
		var title, company, loc, url, jobTypeDB, description string
		if err := rows.Scan(&title, &company, &loc, &url, &jobTypeDB, &description); err != nil {
			continue
		}
		md, err := htmltomarkdown.ConvertString(description)
		if err == nil {
			description = md
		} else {
			log.Printf("HTML-to-markdown conversion failed for job %s at %s: %v", title, company, err)
		}

		detectedLevel, detectedType := detectJobMeta(description)
		if detectedLevel == "" && level != "" {
			detectedLevel = level
		}

		jobs = append(jobs, JobEntry{
			Title:       title,
			Company:     company,
			Location:    loc,
			URL:         url,
			Description: description,
			Level:       detectedLevel,
			Type:        detectedType,
		})
	}

	var filtered []JobEntry
	for _, j := range jobs {
		if level != "" && j.Level != "" && !strings.EqualFold(j.Level, level) {
			continue
		}
		if jobType != "" && j.Type != "" && !strings.EqualFold(j.Type, jobType) {
			continue
		}
		filtered = append(filtered, j)
	}

	return filtered, nil
}

func handleJobsSlash(s *discordgo.Session, i *discordgo.InteractionCreate, dbURL string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		log.Println("Slash jobs defer error:", err)
		return
	}

	level := ""
	jobType := ""
	location := ""
	for _, opt := range i.ApplicationCommandData().Options {
		switch opt.Name {
		case "level":
			level = opt.StringValue()
		case "time":
			jobType = opt.StringValue()
		case "location":
			location = opt.StringValue()
		}
	}

	jobs, err := fetchJobs(dbURL, level, jobType, location)
	if err != nil {
		content := "❌ Failed to fetch jobs. Please try again later."
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &content,
		})
		log.Println("Slash jobs fetch error:", err)
		return
	}

	if len(jobs) == 0 {
		content := "⚠️ No jobs found in the database."
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &content,
		})
		return
	}

	embed := buildJobEmbed(jobs[0], 1, len(jobs))
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "⬅️ Prev",
					Style:    discordgo.SecondaryButton,
					CustomID: "job_page_prev",
					Disabled: true,
				},
				discordgo.Button{
					Label:    "Next ➡️",
					Style:    discordgo.SecondaryButton,
					CustomID: "job_page_next",
				},
			},
		},
	}

	msg, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})
	if err != nil {
		log.Println("Slash jobs edit error:", err)
		return
	}

	cacheLock.Lock()
	jobsCache[msg.ID] = jobs
	cacheLock.Unlock()
}
