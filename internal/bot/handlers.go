package bot

import (
	"context"
	"fmt"
	"log"

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

	if i.Message == nil || i.Message.ID == "" {
		return
	}

	b.cacheLock.RLock()
	jobs, ok := b.jobsCache[i.Message.ID]
	b.cacheLock.RUnlock()
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

	ctx := context.Background()
	jobs, err := b.jobService.FetchAndProcessJobs(ctx, level, jobType, location)
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

	b.cacheLock.Lock()
	b.jobsCache[msg.ID] = jobs
	b.cacheLock.Unlock()
}
