package bot

import (
	"fmt"
	"sort"
	"strings"

	"JoblessYu/internal/job"

	"github.com/bwmarrin/discordgo"
)

func defaultCriteriaState() criteriaState {
	return criteriaState{
		PositionTitle: "Backend Engineer",
		LevelValue:    "junior",
		LocationValue: "hcm",
	}
}

func buildJobSweeperEmbed(state criteriaState, notice string) *discordgo.MessageEmbed {
	levelLabel := levelLabelFromValue(state.LevelValue)
	locationLabel := locationLabelFromValue(state.LocationValue)

	desc := "**Current Active Filters:**\n\n" +
		fmt.Sprintf("• **Position:** `%s` *(Click button below to change)*\n", state.PositionTitle) +
		fmt.Sprintf("• **Level:** `%s`\n", levelLabel) +
		fmt.Sprintf("• **Location:** `%s`", locationLabel)
	if notice != "" {
		desc += "\n\n" + notice
	}

	return &discordgo.MessageEmbed{
		Title:       "🔍 Job Sweeper Criteria",
		Description: desc,
		Color:       0x5865F2,
	}
}

func buildJobSweeperV2Components(state criteriaState, notice string) []discordgo.MessageComponent {
	levelLabel := levelLabelFromValue(state.LevelValue)
	locationLabel := locationLabelFromValue(state.LocationValue)

	criteria := "**Current Active Filters:**\n\n" +
		fmt.Sprintf("• **Position:** `%s` *(Click button below to change)*\n", state.PositionTitle) +
		fmt.Sprintf("• **Level:** `%s`\n", levelLabel) +
		fmt.Sprintf("• **Location:** `%s`", locationLabel)
	if notice != "" {
		criteria += "\n\n" + notice
	}

	levelSelect := discordgo.SelectMenu{
		CustomID:    "select_level",
		Placeholder: "Choose Experience Level (Optional)",
		Options: []discordgo.SelectMenuOption{
			{Label: "Intern", Value: "intern", Description: "Entry level & internship roles", Emoji: &discordgo.ComponentEmoji{Name: "🌱"}},
			{Label: "Junior", Value: "junior", Description: "1-3 years of experience", Emoji: &discordgo.ComponentEmoji{Name: "💻"}},
			{Label: "Senior", Value: "senior", Description: "5+ years & leadership roles", Emoji: &discordgo.ComponentEmoji{Name: "⚡"}},
		},
	}

	locationSelect := discordgo.SelectMenu{
		CustomID:    "select_location",
		Placeholder: "Choose Location (Optional)",
		Options: []discordgo.SelectMenuOption{
			{Label: "Ho Chi Minh", Value: "hcm", Emoji: &discordgo.ComponentEmoji{Name: "🏙️"}},
			{Label: "Ha Noi", Value: "hanoi", Emoji: &discordgo.ComponentEmoji{Name: "🏛️"}},
			{Label: "Both / Remote", Value: "all", Emoji: &discordgo.ComponentEmoji{Name: "🌐"}},
		},
	}

	setPositionBtn := discordgo.Button{
		CustomID: "open_position_modal",
		Label:    "Set Position Title",
		Style:    discordgo.SecondaryButton,
		Emoji:    &discordgo.ComponentEmoji{Name: "📝"},
	}

	searchBtn := discordgo.Button{
		CustomID: "trigger_job_search",
		Label:    "Search Jobs",
		Style:    discordgo.SuccessButton,
		Emoji:    &discordgo.ComponentEmoji{Name: "🚀"},
	}

	accentColor := 0x5865F2
	container := discordgo.Container{
		AccentColor: &accentColor,
		Components: []discordgo.MessageComponent{
			discordgo.TextDisplay{Content: "## 🔍 Job Sweeper Criteria\n" + criteria},
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{levelSelect}},
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{locationSelect}},
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{setPositionBtn, searchBtn}},
		},
	}

	return []discordgo.MessageComponent{container}
}

func buildJobSweeperComponents() []discordgo.MessageComponent {
	levelSelect := discordgo.SelectMenu{
		CustomID:    "select_level",
		Placeholder: "Choose Experience Level (Optional)",
		Options: []discordgo.SelectMenuOption{
			{Label: "Intern", Value: "intern", Description: "Entry level & internship roles", Emoji: &discordgo.ComponentEmoji{Name: "🌱"}},
			{Label: "Junior", Value: "junior", Description: "1-3 years of experience", Emoji: &discordgo.ComponentEmoji{Name: "💻"}},
			{Label: "Senior", Value: "senior", Description: "5+ years & leadership roles", Emoji: &discordgo.ComponentEmoji{Name: "⚡"}},
		},
	}

	locationSelect := discordgo.SelectMenu{
		CustomID:    "select_location",
		Placeholder: "Choose Location (Optional)",
		Options: []discordgo.SelectMenuOption{
			{Label: "Ho Chi Minh", Value: "hcm", Emoji: &discordgo.ComponentEmoji{Name: "🏙️"}},
			{Label: "Ha Noi", Value: "hanoi", Emoji: &discordgo.ComponentEmoji{Name: "🏛️"}},
			{Label: "Both / Remote", Value: "all", Emoji: &discordgo.ComponentEmoji{Name: "🌐"}},
		},
	}

	setPositionBtn := discordgo.Button{
		CustomID: "open_position_modal",
		Label:    "Set Position Title",
		Style:    discordgo.SecondaryButton,
		Emoji:    &discordgo.ComponentEmoji{Name: "📝"},
	}

	searchBtn := discordgo.Button{
		CustomID: "trigger_job_search",
		Label:    "Search Jobs",
		Style:    discordgo.SuccessButton,
		Emoji:    &discordgo.ComponentEmoji{Name: "🚀"},
	}

	return []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{levelSelect}},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{locationSelect}},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{setPositionBtn, searchBtn}},
	}
}

func levelLabelFromValue(v string) string {
	switch v {
	case "intern":
		return "Intern"
	case "senior":
		return "Senior"
	default:
		return "Junior"
	}
}

func locationLabelFromValue(v string) string {
	switch v {
	case "hanoi":
		return "Ha Noi"
	case "all":
		return "Both / Remote"
	default:
		return "Ho Chi Minh"
	}
}

func buildJobEmbed(j job.JobEntry, page, total int) *discordgo.MessageEmbed {
	desc := fmt.Sprintf("• **Role:** %s\n• **Company:** %s\n• **Location:** %s", j.Title, j.Company, j.Location)

	tags := ""
	if j.Level != "" {
		tags += fmt.Sprintf("`%s` ", j.Level)
	}
	if j.Type != "" {
		tags += fmt.Sprintf("`%s` ", j.Type)
	}

	// Append detected skill/tool tags, grouped by category so the embed's
	// Tags line stays readable (e.g. `Cloud: AWS IAM` `IaC: CloudFormation`).
	if len(j.Tags) > 0 {
		categories := make([]string, 0, len(j.Tags))
		for cat := range j.Tags {
			categories = append(categories, cat)
		}
		sort.Strings(categories)
		for _, cat := range categories {
			tags += fmt.Sprintf("`%s: %s` ", cat, strings.Join(j.Tags[cat], " "))
		}
	}

	if tags = strings.TrimSpace(tags); tags != "" {
		desc += fmt.Sprintf("\n• **Tags:** %s", tags)
	}

	// AI-generated summary (only present when AI enriched the row).
	if j.Summary != "" {
		desc += fmt.Sprintf("\n• **Summary:** %s", j.Summary)
	}

	// Salary range if AI extracted it.
	if j.Salary != "" {
		desc += fmt.Sprintf("\n• **Salary:** %s", j.Salary)
	}

	// Remote-eligible badge.
	if j.Remote {
		desc += "\n• **Remote:** ✅ Yes"
	}

	desc += fmt.Sprintf("\n• **Apply:** [Open job](%s)", j.URL)

	return &discordgo.MessageEmbed{
		Title:       j.Company,
		Description: desc,
		Color:       0x5865F2,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Page %d of %d", page, total),
		},
	}
}
