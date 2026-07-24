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
	case discordgo.InteractionModalSubmit:
		b.handleModalSubmit(s, i)
	}
}

func (b *Bot) handleJobsSlash(s *discordgo.Session, i *discordgo.InteractionCreate) {
	state := defaultCriteriaState()
	embed := buildJobSweeperEmbed(state, "")
	components := buildJobSweeperComponents()

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
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

	switch i.MessageComponentData().CustomID {
	case "job_page_prev", "job_page_next":
		b.handlePaginationComponent(s, i)
	case "select_level", "select_location", "open_position_modal", "trigger_job_search":
		b.handleSweeperComponent(s, i)
	}
}

func (b *Bot) handleSweeperComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	state := b.getCriteriaState(i.Message.ID)
	customID := i.MessageComponentData().CustomID

	switch customID {
	case "select_level":
		if vals := i.MessageComponentData().Values; len(vals) > 0 {
			state.LevelValue = vals[0]
		}
		b.saveCriteriaState(i.Message.ID, state)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Embeds:     []*discordgo.MessageEmbed{buildJobSweeperEmbed(state, "")},
				Components: buildJobSweeperComponents(),
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
				Embeds:     []*discordgo.MessageEmbed{buildJobSweeperEmbed(state, "")},
				Components: buildJobSweeperComponents(),
			},
		})
	case "open_position_modal":
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID: fmt.Sprintf("set_position_modal:%s", i.Message.ID),
				Title:    "Set Position Title",
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "position_title",
							Label:       "Position",
							Style:       discordgo.TextInputShort,
							Value:       state.PositionTitle,
							Placeholder: "Backend Engineer",
							Required:    true,
							MaxLength:   80,
						},
					}},
				},
			},
		})
	case "trigger_job_search":
		b.handleSearchJobs(s, i, state)
	}
}

func (b *Bot) handleModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.ModalSubmitData().CustomID
	if !strings.HasPrefix(customID, "set_position_modal:") {
		return
	}

	messageID := strings.TrimPrefix(customID, "set_position_modal:")
	if messageID == "" {
		return
	}

	state := b.getCriteriaState(messageID)
	for _, rowComp := range i.ModalSubmitData().Components {
		row, ok := rowComp.(discordgo.ActionsRow)
		if !ok {
			continue
		}
		for _, inputComp := range row.Components {
			input, ok := inputComp.(discordgo.TextInput)
			if !ok {
				continue
			}
			if input.CustomID == "position_title" {
				state.PositionTitle = strings.TrimSpace(input.Value)
			}
		}
	}
	if state.PositionTitle == "" {
		state.PositionTitle = defaultCriteriaState().PositionTitle
	}
	b.saveCriteriaState(messageID, state)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{buildJobSweeperEmbed(state, "")},
			Components: buildJobSweeperComponents(),
		},
	})
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
		AIEnabled: b.cfg.GroqAPIKey != "",
	}

	jobs, err := b.jobService.FetchAndProcessJobs(ctx, q)
	if err != nil {
		content := "Failed to fetch jobs. Please try again later."
		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content})
		log.Println("Search jobs fetch error:", err)
		return
	}

	jobs = filterJobsByPosition(jobs, state.PositionTitle)
	if len(jobs) == 0 {
		embed := buildJobSweeperEmbed(state, "No jobs found with the current filters.")
		components := buildJobSweeperComponents()
		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds:     &[]*discordgo.MessageEmbed{embed},
			Components: &components,
		})
		return
	}

	embed := buildJobEmbed(jobs[0], 1, len(jobs))
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{Label: "Prev", Style: discordgo.SecondaryButton, CustomID: "job_page_prev", Disabled: true},
			discordgo.Button{Label: "Next", Style: discordgo.SecondaryButton, CustomID: "job_page_next", Disabled: len(jobs) == 1},
		}},
	}

	msg, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})
	if err != nil {
		log.Println("Search jobs edit error:", err)
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
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "This job list has expired. Please run /jobs again.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	page := 1
	if len(i.Message.Embeds) > 0 && i.Message.Embeds[0].Footer != nil {
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
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{Label: "Prev", Style: discordgo.SecondaryButton, CustomID: "job_page_prev", Disabled: page == 1},
			discordgo.Button{Label: "Next", Style: discordgo.SecondaryButton, CustomID: "job_page_next", Disabled: page == len(jobs)},
		}},
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	})
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
	case "senior":
		return "Senior"
	default:
		return "Junior"
	}
}

func mapLocationToQuery(v string) string {
	switch v {
	case "hanoi":
		return "HN"
	case "all":
		return ""
	default:
		return "HCM"
	}
}

func filterJobsByPosition(jobs []job.JobEntry, position string) []job.JobEntry {
	position = strings.TrimSpace(strings.ToLower(position))
	if position == "" {
		return jobs
	}

	filtered := make([]job.JobEntry, 0, len(jobs))
	for _, j := range jobs {
		if strings.Contains(strings.ToLower(j.Title), position) {
			filtered = append(filtered, j)
		}
	}
	return filtered
}
