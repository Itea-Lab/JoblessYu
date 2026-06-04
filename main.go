package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

type JobEntry struct {
	Title    string
	Company  string
	Location string
	URL      string
}

var (
	jobsCache = map[string][]JobEntry{}
	cacheLock sync.RWMutex
	commands  = []*discordgo.ApplicationCommand{
		{
			Name:        "jobs",
			Description: "Show latest jobs in a paginated embed",
			Type:        discordgo.ChatApplicationCommand,
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
			var total int
			_, err := fmt.Sscanf(i.Message.Embeds[0].Footer.Text, "Job %d of %d • %*s", &page, &total)
			if err != nil {
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

func buildJobEmbed(job JobEntry, page, total int) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       job.Company,
		Description: fmt.Sprintf("• **Role:** %s\n• **Location:** %s\n• **Apply:** [Open job](%s)", job.Title, job.Location, job.URL),
		Color:       0x5865F2,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Job %d of %d • Powered by JobSpy + Neon", page, total),
		},
	}
}

func fetchJobs(dbURL string) ([]JobEntry, error) {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		return nil, err
	}
	defer conn.Close(ctx)

	rows, err := conn.Query(
		ctx,
		`SELECT title, company, location, job_url
		 FROM jobs
		 ORDER BY fetched_at DESC
		 LIMIT 10`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []JobEntry
	for rows.Next() {
		var title, company, location, url string
		if err := rows.Scan(&title, &company, &location, &url); err != nil {
			continue
		}
		jobs = append(jobs, JobEntry{Title: title, Company: company, Location: location, URL: url})
	}

	return jobs, nil
}

func handleJobsSlash(s *discordgo.Session, i *discordgo.InteractionCreate, dbURL string) {
	jobs, err := fetchJobs(dbURL)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Failed to fetch jobs. Please try again later.",
				Flags:   1 << 6,
			},
		})
		log.Println("Slash jobs fetch error:", err)
		return
	}

	if len(jobs) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "⚠️ No jobs found in the database.",
				Flags:   1 << 6,
			},
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

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	})
	if err != nil {
		log.Println("Slash jobs send error:", err)
		return
	}

	msg, err := s.InteractionResponse(i.Interaction)
	if err != nil {
		log.Println("Failed to read slash response message:", err)
		return
	}

	cacheLock.Lock()
	jobsCache[msg.ID] = jobs
	cacheLock.Unlock()
}
