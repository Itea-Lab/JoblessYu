package bot

import (
	"context"
	"fmt"
	"log"
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
			Flags:      discordgo.MessageFlagsEphemeral | discordgo.MessageFlagsIsComponentsV2,
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

	sessionKey := i.Interaction.ID
	components := buildJobResultV2Components(jobs[0], 1, len(jobs), sessionKey)

	msg, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Components: components,
		Flags:      discordgo.MessageFlagsEphemeral | discordgo.MessageFlagsIsComponentsV2,
	})
	if err != nil {
		log.Println("Search jobs followup error:", err)
		return
	}

	b.cacheLock.Lock()
	entry := cachedJobs{jobs: jobs, currentPage: 1, insertedAt: time.Now()}
	if sessionKey != "" {
		b.jobsCache[sessionKey] = entry
	}
	if msg != nil && msg.ID != "" {
		b.jobsCache[msg.ID] = entry
	}
	b.cacheLock.Unlock()
}

func (b *Bot) handlePaginationComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	action, sessionKey, ok := strings.Cut(customID, ":")
	if !ok || sessionKey == "" {
		return
	}

	b.cacheLock.RLock()
	cached, ok := b.jobsCache[sessionKey]
	b.cacheLock.RUnlock()
	jobs := cached.jobs
	if !ok || len(jobs) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "This job list has expired. Please run /jobs again.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	page := cached.currentPage
	if page < 1 {
		page = 1
	}

	switch action {
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

	b.cacheLock.Lock()
	b.jobsCache[sessionKey] = cachedJobs{jobs: jobs, currentPage: page, insertedAt: time.Now()}
	b.cacheLock.Unlock()

	components := buildJobResultV2Components(jobs[page-1], page, len(jobs), sessionKey)

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Components: components,
			Flags:      discordgo.MessageFlagsIsComponentsV2,
		},
	})
	if err != nil {
		log.Println("Pagination update error:", err)
	}
}

func buildJobPaginationComponents(page, total int, sessionKey string, jobURL string) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: buildJobPaginationButtons(page, total, sessionKey, jobURL)},
	}
}

func buildJobPaginationButtons(page, total int, sessionKey string, jobURL string) []discordgo.MessageComponent {
	prevID := "job_page_prev"
	nextID := "job_page_next"
	if sessionKey != "" {
		prevID = fmt.Sprintf("job_page_prev:%s", sessionKey)
		nextID = fmt.Sprintf("job_page_next:%s", sessionKey)
	}

	buttons := []discordgo.MessageComponent{
		discordgo.Button{Label: "Previous", Style: discordgo.SecondaryButton, CustomID: prevID, Disabled: page == 1},
		discordgo.Button{Label: "Next", Style: discordgo.PrimaryButton, CustomID: nextID, Disabled: page == total},
	}
	if strings.TrimSpace(jobURL) != "" {
		buttons = append(buttons, discordgo.Button{Label: "Apply on indeed", Style: discordgo.LinkButton, URL: jobURL})
	}
	return buttons
}


func buildJobResultV2Components(j job.JobEntry, page, total int, sessionKey string) []discordgo.MessageComponent {
	embed := buildJobEmbed(j, page, total)
	content := fmt.Sprintf("## %s\n\n%s", embed.Title, embed.Description)
	if embed.Footer != nil && strings.TrimSpace(embed.Footer.Text) != "" {
		content += "\n\n" + embed.Footer.Text
	}
	content = truncateForDiscord(content, 3900)

	accentColor := 0x5865F2
	container := discordgo.Container{
		AccentColor: &accentColor,
		Components: []discordgo.MessageComponent{
			discordgo.TextDisplay{Content: content},
			discordgo.ActionsRow{Components: buildJobPaginationButtons(page, total, sessionKey, j.URL)},
		},
	}

	return []discordgo.MessageComponent{
		container,
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
