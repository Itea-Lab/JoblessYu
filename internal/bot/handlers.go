package bot

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"

	"JoblessYu/internal/job"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) HandleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
	if strings.HasPrefix(customID, "job_page_") {
		b.handlePaginationComponent(s, i)
		return
	}

	switch customID {
	case "select_position", "select_level", "select_location", "select_type", "trigger_job_search":
		b.handleSweeperComponent(s, i)
	}
}

func (b *Bot) handleModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.ModalSubmitData().CustomID
	if strings.HasPrefix(customID, "job_page_goto_submit") {
		b.handlePageJumpSubmit(s, i)
	}
}

func (b *Bot) handleSweeperComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	state := b.getCriteriaState(i.Message.ID)
	customID := i.MessageComponentData().CustomID

	switch customID {
	case "select_position":
		state.Positions = normalizeSelectionValues(i.MessageComponentData().Values, state.Positions)
		b.saveCriteriaState(i.Message.ID, state)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Components: buildJobSweeperV2Components(state, ""),
				Flags:      discordgo.MessageFlagsIsComponentsV2,
			},
		})
	case "select_level":
		state.Levels = normalizeSelectionValues(i.MessageComponentData().Values, state.Levels)
		b.saveCriteriaState(i.Message.ID, state)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Components: buildJobSweeperV2Components(state, ""),
				Flags:      discordgo.MessageFlagsIsComponentsV2,
			},
		})
	case "select_location":
		state.Locations = normalizeSelectionValues(i.MessageComponentData().Values, state.Locations)
		b.saveCriteriaState(i.Message.ID, state)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Components: buildJobSweeperV2Components(state, ""),
				Flags:      discordgo.MessageFlagsIsComponentsV2,
			},
		})
	case "select_type":
		state.JobTypes = normalizeSelectionValues(i.MessageComponentData().Values, state.JobTypes)
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

func normalizeSelectionValues(vals []string, prevState []string) []string {
	if len(vals) == 0 {
		return []string{"all"}
	}

	prevHasAll := sliceContains(prevState, "all")
	currHasAll := sliceContains(vals, "all")

	if currHasAll && !prevHasAll {
		return []string{"all"}
	}

	var res []string
	for _, v := range vals {
		if v != "" && v != "all" {
			res = append(res, v)
		}
	}
	if len(res) == 0 {
		return []string{"all"}
	}
	return res
}

func sliceContains(slice []string, target string) bool {
	for _, v := range slice {
		if v == target {
			return true
		}
	}
	return false
}

func (b *Bot) handleSearchJobs(s *discordgo.Session, i *discordgo.InteractionCreate, state criteriaState) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
	if err != nil {
		log.Println("Search jobs defer update error:", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	q := job.JobQuery{
		Levels:     mapLevelsToQuery(state.Levels),
		Locations:  mapLocationsToQuery(state.Locations),
		Expertises: mapPositionsToQuery(state.Positions),
		JobTypes:   mapJobTypesToQuery(state.JobTypes),
		AIEnabled:  b.cfg.GroqAPIKey != "",
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
	case "job_page_goto":
		b.showPageJumpModal(s, i, sessionKey, page, len(jobs))
		return
	case "job_page_first":
		page = 1
	case "job_page_prev":
		if page > 1 {
			page--
		}
	case "job_page_next":
		if page < len(jobs) {
			page++
		}
	case "job_page_last":
		page = len(jobs)
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

func (b *Bot) showPageJumpModal(s *discordgo.Session, i *discordgo.InteractionCreate, sessionKey string, page, total int) {
	maxDigits := len(strconv.Itoa(total))
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("job_page_goto_submit:%s", sessionKey),
			Title:    fmt.Sprintf("Go to page (1-%d)", total),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "job_page_number",
						Label:       "Page number",
						Style:       discordgo.TextInputShort,
						Placeholder: fmt.Sprintf("Enter a number from 1 to %d", total),
						Value:       strconv.Itoa(page),
						Required:    true,
						MinLength:   1,
						MaxLength:   maxDigits,
					},
				}},
			},
		},
	})
	if err != nil {
		log.Println("Page jump modal error:", err)
	}
}

func (b *Bot) handlePageJumpSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	action, sessionKey, ok := strings.Cut(i.ModalSubmitData().CustomID, ":")
	if !ok || action != "job_page_goto_submit" || sessionKey == "" {
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

	pageRaw := extractPageNumberInput(i.ModalSubmitData().Components)
	page, err := strconv.Atoi(strings.TrimSpace(pageRaw))
	if err != nil || page < 1 || page > len(jobs) {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Please enter a valid page number between 1 and %d.", len(jobs)),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	b.cacheLock.Lock()
	b.jobsCache[sessionKey] = cachedJobs{jobs: jobs, currentPage: page, insertedAt: time.Now()}
	b.cacheLock.Unlock()

	components := buildJobResultV2Components(jobs[page-1], page, len(jobs), sessionKey)
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Components: components,
			Flags:      discordgo.MessageFlagsIsComponentsV2,
		},
	})
	if err != nil {
		log.Println("Page jump update error:", err)
	}
}

func extractPageNumberInput(components []discordgo.MessageComponent) string {
	for _, c := range components {
		switch row := c.(type) {
		case discordgo.ActionsRow:
			for _, inner := range row.Components {
				switch input := inner.(type) {
				case discordgo.TextInput:
					return input.Value
				case *discordgo.TextInput:
					return input.Value
				}
			}
		case *discordgo.ActionsRow:
			for _, inner := range row.Components {
				switch input := inner.(type) {
				case discordgo.TextInput:
					return input.Value
				case *discordgo.TextInput:
					return input.Value
				}
			}
		}
	}
	return ""
}

func buildJobPaginationComponents(page, total int, sessionKey string) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: buildJobNavigationButtons(page, total, sessionKey)},
	}
}

func buildJobNavigationButtons(page, total int, sessionKey string) []discordgo.MessageComponent {
	firstID := "job_page_first"
	prevID := "job_page_prev"
	gotoID := "job_page_goto"
	nextID := "job_page_next"
	lastID := "job_page_last"
	if sessionKey != "" {
		firstID = fmt.Sprintf("job_page_first:%s", sessionKey)
		prevID = fmt.Sprintf("job_page_prev:%s", sessionKey)
		gotoID = fmt.Sprintf("job_page_goto:%s", sessionKey)
		nextID = fmt.Sprintf("job_page_next:%s", sessionKey)
		lastID = fmt.Sprintf("job_page_last:%s", sessionKey)
	}

	return []discordgo.MessageComponent{
		discordgo.Button{Label: "<<", Style: discordgo.SecondaryButton, CustomID: firstID, Disabled: page == 1},
		discordgo.Button{Label: "<", Style: discordgo.SecondaryButton, CustomID: prevID, Disabled: page == 1},
		discordgo.Button{Label: fmt.Sprintf("%d/%d", page, total), Style: discordgo.SecondaryButton, CustomID: gotoID, Disabled: total <= 1},
		discordgo.Button{Label: ">", Style: discordgo.PrimaryButton, CustomID: nextID, Disabled: page == total},
		discordgo.Button{Label: ">>", Style: discordgo.PrimaryButton, CustomID: lastID, Disabled: page == total},
	}
}

func buildJobApplyButtonRow(jobURL, site string) []discordgo.MessageComponent {
	if strings.TrimSpace(jobURL) == "" {
		return nil
	}
	label := "Apply now"
	if siteLabel := deriveApplySiteLabel(site, jobURL); siteLabel != "" {
		label = fmt.Sprintf("Apply on %s", siteLabel)
	}
	return []discordgo.MessageComponent{
		discordgo.Button{Label: label, Style: discordgo.LinkButton, URL: jobURL},
	}
}

func deriveApplySiteLabel(site, jobURL string) string {
	if s := strings.ToLower(strings.TrimSpace(site)); s != "" {
		return s
	}

	u, err := url.Parse(strings.TrimSpace(jobURL))
	if err != nil {
		return ""
	}

	host := strings.ToLower(strings.TrimSpace(u.Hostname()))
	host = strings.TrimPrefix(host, "www.")
	if host == "" {
		return ""
	}

	parts := strings.Split(host, ".")
	if len(parts) >= 2 {
		return parts[len(parts)-2]
	}
	return host
}

func buildJobResultV2Components(j job.JobEntry, page, total int, sessionKey string) []discordgo.MessageComponent {
	embed := buildJobEmbed(j, page, total)
	content := fmt.Sprintf("## %s\n\n%s", embed.Title, embed.Description)
	content = truncateForDiscord(content, 3900)

	accentColor := 0x5865F2
	containerComponents := []discordgo.MessageComponent{
		discordgo.TextDisplay{Content: content},
	}
	if applyButtons := buildJobApplyButtonRow(j.URL, j.Site); len(applyButtons) > 0 {
		containerComponents = append(containerComponents, discordgo.ActionsRow{Components: applyButtons})
	}
	if embed.Footer != nil && strings.TrimSpace(embed.Footer.Text) != "" {
		containerComponents = append(containerComponents, discordgo.TextDisplay{Content: embed.Footer.Text})
	}

	container := discordgo.Container{
		AccentColor: &accentColor,
		Components:  containerComponents,
	}

	return []discordgo.MessageComponent{
		container,
		discordgo.ActionsRow{Components: buildJobNavigationButtons(page, total, sessionKey)},
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

func mapLevelsToQuery(vals []string) []string {
	var res []string
	for _, v := range vals {
		switch v {
		case "intern":
			res = append(res, "Intern")
		case "fresher":
			res = append(res, "Fresher")
		case "junior":
			res = append(res, "Junior")
		case "middle":
			res = append(res, "Middle")
		case "senior":
			res = append(res, "Senior")
		case "lead":
			res = append(res, "Lead")
		}
	}
	return res
}

func mapLocationsToQuery(vals []string) []string {
	var res []string
	for _, v := range vals {
		switch v {
		case "hanoi":
			res = append(res, "HN")
		case "hcm":
			res = append(res, "HCM")
		case "danang":
			res = append(res, "DaNang")
		case "remote":
			res = append(res, "Remote")
		}
	}
	return res
}

func mapPositionsToQuery(vals []string) []string {
	var res []string
	for _, v := range vals {
		if v != "" && v != "all" {
			res = append(res, v)
		}
	}
	return res
}

func mapJobTypesToQuery(vals []string) []string {
	var res []string
	for _, v := range vals {
		switch v {
		case "full_time":
			res = append(res, "Full-time")
		case "part_time":
			res = append(res, "Part-time")
		case "contract":
			res = append(res, "Contract")
		}
	}
	return res
}

func mapLevelToQuery(v string) string {
	res := mapLevelsToQuery([]string{v})
	if len(res) > 0 {
		return res[0]
	}
	return ""
}

func mapLocationToQuery(v string) string {
	res := mapLocationsToQuery([]string{v})
	if len(res) > 0 {
		return res[0]
	}
	return ""
}

func mapPositionToQuery(v string) string {
	if v == "all" || v == "" {
		return ""
	}
	return v
}

func mapJobTypeToQuery(v string) string {
	res := mapJobTypesToQuery([]string{v})
	if len(res) > 0 {
		return res[0]
	}
	return ""
}
