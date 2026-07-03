package bot

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) handleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type == discordgo.InteractionApplicationCommand {
		if i.ApplicationCommandData().Name == "jobs" {
			b.handleJobsSlash(s, i)
		}
		return
	}

	if i.Type != discordgo.InteractionMessageComponent {
		return
	}

	if i.Message == nil {
		return
	}

	// The cache is keyed primarily on the original interaction ID that created
	// the job-list message. Discord populates i.Message.Interaction for
	// interaction-response messages, so a button click resolves back to the
	// slash command's interaction ID. We fall back to the message ID for
	// robustness against any discordgo version quirk.
	cacheKey := ""
	if i.Message.Interaction != nil {
		cacheKey = i.Message.Interaction.ID
	}
	if cacheKey == "" {
		cacheKey = i.Message.ID
	}
	if cacheKey == "" {
		return
	}

	b.cacheLock.RLock()
	cached, ok := b.jobsCache[cacheKey]
	b.cacheLock.RUnlock()
	jobs := cached.jobs
	if !ok || len(jobs) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "⚠️ This job list has expired. Please run `/jobs` again.",
				Flags:   discordgo.MessageFlagsEphemeral,
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
}

func (b *Bot) handleJobsSlash(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
	includeUnknown := false
	for _, opt := range i.ApplicationCommandData().Options {
		switch opt.Name {
		case "level":
			level = opt.StringValue()
		case "type":
			jobType = opt.StringValue()
		case "location":
			location = opt.StringValue()
		case "include_unknown":
			includeUnknown = opt.BoolValue()
		}
	}

	ctx := context.Background()
	jobs, err := b.jobService.FetchAndProcessJobs(ctx, level, jobType, location, includeUnknown)
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

	// Key on the original interaction ID (matches what the button-click handler
	// reads via i.Message.Interaction.ID). Also write under msg.ID when Discord
	// returns it, as a defensive fallback.
	b.cacheLock.Lock()
	entry := cachedJobs{jobs: jobs, insertedAt: time.Now()}
	b.jobsCache[i.Interaction.ID] = entry
	if msg != nil && msg.ID != "" {
		b.jobsCache[msg.ID] = entry
	}
	b.cacheLock.Unlock()
}
