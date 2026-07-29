package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"JoblessYu/internal/job"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) handleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		if i.ApplicationCommandData().Name == "jobs" {
			b.handleJobsSlash(s, i)
		}
	case discordgo.InteractionMessageComponent:
		b.handleMessageComponent(s, i)
	}
}

func (b *Bot) handleJobsSlash(s *discordgo.Session, i *discordgo.InteractionCreate) {
	state := defaultCriteriaState()
	components := buildJobSweeperV2Components(state, "")

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Components: components,
			Flags:      discordgo.MessageFlagsIsComponentsV2,
		},
	})
	if err != nil {
		log.Println("Slash jobs response error:", err)
		return
	}

	msg, err := s.InteractionResponse(i.Interaction)
	if err != nil {
		log.Println("Slash jobs fetch response error:", err)
		return
	}
	if msg == nil || msg.ID == "" {
		return
	}

	b.cacheLock.Lock()
	b.uiState[msg.ID] = cachedCriteria{state: state, insertedAt: time.Now()}
	b.cacheLock.Unlock()
}

func (b *Bot) handleMessageComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Message == nil {
		return
	}

	customID := i.MessageComponentData().CustomID
	if strings.HasPrefix(customID, "job_page_prev") || strings.HasPrefix(customID, "job_page_next") {
		b.handlePaginationComponent(s, i)
		return
	}

	switch customID {
	case "select_position", "select_level", "select_location", "select_type", "trigger_job_search":
		b.handleSweeperComponent(s, i)
	}
}

func (b *Bot) handleSweeperComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	state := b.getCriteriaState(i.Message.ID)
	customID := i.MessageComponentData().CustomID

	switch customID {
	case "select_position":
		if vals := i.MessageComponentData().Values; len(vals) > 0 {
			state.PositionValue = vals[0]
		}
		b.saveCriteriaState(i.Message.ID, state)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Components: buildJobSweeperV2Components(state, ""),
				Flags:      discordgo.MessageFlagsIsComponentsV2,
			},
		})
	case "select_level":
		if vals := i.MessageComponentData().Values; len(vals) > 0 {
			state.LevelValue = vals[0]
		}
		b.saveCriteriaState(i.Message.ID, state)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Components: buildJobSweeperV2Components(state, ""),
				Flags:      discordgo.MessageFlagsIsComponentsV2,
			},
		})
	case "select_location":
		if vals := i.MessageComponentData().Values; len(vals) > 0 {
			state.LocationValue = vals[0]
		}
		b.saveCriteriaState(i.Message.ID, state)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Components: buildJobSweeperV2Components(state, ""),
				Flags:      discordgo.MessageFlagsIsComponentsV2,
			},
		})
	case "select_type":
		if vals := i.MessageComponentData().Values; len(vals) > 0 {
			state.JobTypeValue = vals[0]
		}
		b.saveCriteriaState(i.Message.ID, state)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Components: buildJobSweeperV2Components(state, ""),
				Flags:      discordgo.MessageFlagsIsComponentsV2,
			},
		})
	case "trigger_job_search":
		b.handleSearchJobs(s, i, state)
	}
}

func (b *Bot) handleSearchJobs(s *discordgo.Session, i *discordgo.InteractionCreate, state criteriaState) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
	if err != nil {
		log.Println("Search jobs defer update error:", err)
		return
	}

	ctx := context.Background()
	q := job.JobQuery{
		Level:     mapLevelToQuery(state.LevelValue),
		Location:  mapLocationToQuery(state.LocationValue),
		Expertise: mapPositionToQuery(state.PositionValue),
		JobType:   mapJobTypeToQuery(state.JobTypeValue),
		AIEnabled: b.cfg.GroqAPIKey != "",
	}

	jobs, err := b.jobService.FetchAndProcessJobs(ctx, q)
	if err != nil {
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "Failed to fetch jobs. Please try again later.",
		})
		log.Println("Search jobs fetch error:", err)
		return
	}

	if len(jobs) == 0 {
		components := buildJobSweeperV2Components(state, "No jobs found with the current filters.")
		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Components: &components,
		})
		return
	}

	criteriaComponents := buildJobSweeperV2Components(state, "")
	_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Components: &criteriaComponents})

	embed := buildJobEmbed(jobs[0], 1, len(jobs))
	components := buildJobPaginationComponents(1, len(jobs))

	msg, err := s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: components,
	})
	if err != nil {
		log.Println("Search jobs followup error:", err)
		return
	}

	b.cacheLock.Lock()
	entry := cachedJobs{jobs: jobs, insertedAt: time.Now()}
	if i.Message != nil && i.Message.ID != "" {
		b.jobsCache[i.Message.ID] = entry
	}
	if i.Message != nil && i.Message.Interaction != nil && i.Message.Interaction.ID != "" {
		b.jobsCache[i.Message.Interaction.ID] = entry
	}
	if i.Interaction != nil && i.Interaction.ID != "" {
		b.jobsCache[i.Interaction.ID] = entry
	}
	if msg != nil && msg.ID != "" {
		b.jobsCache[msg.ID] = entry
	}
	b.cacheLock.Unlock()
}

func (b *Bot) handlePaginationComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
		// Self-healing fallback: If cache expired or bot was restarted, re-fetch active jobs from DB
		ctx := context.Background()
		var err error
		jobs, err = b.jobService.FetchAndProcessJobs(ctx, job.JobQuery{AIEnabled: b.cfg.GroqAPIKey != ""})
		if err != nil || len(jobs) == 0 {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "This job list has expired. Please run /jobs again.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
		// Save back to cache for subsequent pagination clicks
		b.cacheLock.Lock()
		b.jobsCache[cacheKey] = cachedJobs{jobs: jobs, insertedAt: time.Now()}
		b.cacheLock.Unlock()
	}

	page := 1
	customID := i.MessageComponentData().CustomID
	if _, suffix, ok := strings.Cut(customID, ":"); ok {
		if parsedPage, err := strconv.Atoi(suffix); err == nil && parsedPage >= 1 {
			page = parsedPage
		}
	} else if len(i.Message.Embeds) > 0 {
		var total int
		if _, err := fmt.Sscanf(i.Message.Embeds[0].Title, "Job Listing (%d of %d)", &page, &total); err != nil {
			page = 1
		}
		if page < 1 {
			page = 1
		}
	}

	switch {
	case strings.HasPrefix(customID, "job_page_prev"):
		if page > 1 {
			page--
		}
	case strings.HasPrefix(customID, "job_page_next"):
		if page < len(jobs) {
			page++
		}
	default:
		return
	}

	embed := buildJobEmbed(jobs[page-1], page, len(jobs))
	components := buildJobPaginationComponents(page, len(jobs))

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	})
	if err != nil {
		log.Println("Pagination update error:", err)
	}
}

func buildJobPaginationComponents(page, total int) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{Label: "Previous", Style: discordgo.SecondaryButton, CustomID: fmt.Sprintf("job_page_prev:%d", page), Disabled: page == 1},
			discordgo.Button{Label: "Next", Style: discordgo.PrimaryButton, CustomID: fmt.Sprintf("job_page_next:%d", page), Disabled: page == total},
		}},
	}
}

func (b *Bot) getCriteriaState(messageID string) criteriaState {
	b.cacheLock.RLock()
	defer b.cacheLock.RUnlock()

	if cached, ok := b.uiState[messageID]; ok {
		return cached.state
	}
	return defaultCriteriaState()
}

func (b *Bot) saveCriteriaState(messageID string, state criteriaState) {
	b.cacheLock.Lock()
	b.uiState[messageID] = cachedCriteria{state: state, insertedAt: time.Now()}
	b.cacheLock.Unlock()
}

func mapLevelToQuery(v string) string {
	switch v {
	case "intern":
		return "Intern"
	case "fresher":
		return "Fresher"
	case "junior":
		return "Junior"
	case "senior":
		return "Senior"
	default:
		return ""
	}
}

func mapLocationToQuery(v string) string {
	switch v {
	case "hanoi":
		return "HN"
	case "hcm":
		return "HCM"
	default:
		return ""
	}
}

func mapPositionToQuery(v string) string {
	if v == "all" || v == "" {
		return ""
	}
	return v
}

func mapJobTypeToQuery(v string) string {
	switch v {
	case "full_time":
		return "Full-time"
	case "part_time":
		return "Part-time"
	case "contract":
		return "Contract"
	default:
		return ""
	}
}
