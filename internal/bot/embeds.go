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
		PositionValue: "all",
		LevelValue:    "all",
		LocationValue: "all",
		JobTypeValue:  "all",
	}
}

func buildJobSweeperEmbed(state criteriaState, notice string) *discordgo.MessageEmbed {
	positionLabel := positionLabelFromValue(state.PositionValue)
	levelLabel := levelLabelFromValue(state.LevelValue)
	locationLabel := locationLabelFromValue(state.LocationValue)
	jobTypeLabel := jobTypeLabelFromValue(state.JobTypeValue)

	desc := "### Filter Configuration\n" +
		fmt.Sprintf("> Position: `%s`\n", positionLabel) +
		fmt.Sprintf("> Experience: `%s`\n", levelLabel) +
		fmt.Sprintf("> Location: `%s`\n", locationLabel) +
		fmt.Sprintf("> Job Type: `%s`", jobTypeLabel)

	if notice != "" {
		desc += "\n\n*" + notice + "*"
	}

	return &discordgo.MessageEmbed{
		Title:       "Job Sweeper",
		Description: desc,
		Color:       0x5865F2,
	}
}

func buildJobSweeperV2Components(state criteriaState, notice string) []discordgo.MessageComponent {
	positionLabel := positionLabelFromValue(state.PositionValue)
	levelLabel := levelLabelFromValue(state.LevelValue)
	locationLabel := locationLabelFromValue(state.LocationValue)
	jobTypeLabel := jobTypeLabelFromValue(state.JobTypeValue)

	header := "## Job Sweeper\n" +
		"Select filters below to find job listings."

	if notice != "" {
		header += "\n\n*" + notice + "*"
	}

	positionOptions := []discordgo.SelectMenuOption{
		{Label: "All Positions", Value: "all"},
		{Label: "Web Development", Value: "web_dev"},
		{Label: "Cloud & DevOps", Value: "cloud_devops"},
		{Label: "Data & AI", Value: "data_ai"},
		{Label: "Mobile & Game", Value: "mobile_game"},
		{Label: "Testing & QA", Value: "testing_qa"},
		{Label: "IT Support & Security", Value: "support_security"},
		{Label: "Software Architecture", Value: "architecture"},
		{Label: "Embedded & IoT", Value: "embedded_iot"},
		{Label: "Enterprise Systems", Value: "enterprise"},
		{Label: "Systems & Network", Value: "systems_network"},
		{Label: "Management & Executive", Value: "management"},
		{Label: "Design & UX", Value: "design_ux"},
	}
	for i := range positionOptions {
		if positionOptions[i].Value == state.PositionValue {
			positionOptions[i].Default = true
		}
	}

	levelOptions := []discordgo.SelectMenuOption{
		{Label: "All Levels", Value: "all"},
		{Label: "Intern", Value: "intern", Description: "Internship roles"},
		{Label: "Fresher", Value: "fresher", Description: "Fresh graduates & 0-1 years experience"},
		{Label: "Junior", Value: "junior", Description: "1-3 years of experience"},
		{Label: "Senior", Value: "senior", Description: "5+ years & leadership roles"},
	}
	for i := range levelOptions {
		if levelOptions[i].Value == state.LevelValue {
			levelOptions[i].Default = true
		}
	}

	locationOptions := []discordgo.SelectMenuOption{
		{Label: "All Locations", Value: "all"},
		{Label: "Ho Chi Minh", Value: "hcm"},
		{Label: "Ha Noi", Value: "hanoi"},
	}
	for i := range locationOptions {
		if locationOptions[i].Value == state.LocationValue {
			locationOptions[i].Default = true
		}
	}

	typeOptions := []discordgo.SelectMenuOption{
		{Label: "All Job Types", Value: "all"},
		{Label: "Full-time", Value: "full_time"},
		{Label: "Part-time", Value: "part_time"},
		{Label: "Contract", Value: "contract"},
	}
	for i := range typeOptions {
		if typeOptions[i].Value == state.JobTypeValue {
			typeOptions[i].Default = true
		}
	}

	positionSelect := discordgo.SelectMenu{
		CustomID:    "select_position",
		Placeholder: "Position: " + positionLabel,
		Options:     positionOptions,
	}

	levelSelect := discordgo.SelectMenu{
		CustomID:    "select_level",
		Placeholder: "Level: " + levelLabel,
		Options:     levelOptions,
	}

	locationSelect := discordgo.SelectMenu{
		CustomID:    "select_location",
		Placeholder: "Location: " + locationLabel,
		Options:     locationOptions,
	}

	typeSelect := discordgo.SelectMenu{
		CustomID:    "select_type",
		Placeholder: "Job Type: " + jobTypeLabel,
		Options:     typeOptions,
	}

	searchBtn := discordgo.Button{
		CustomID: "trigger_job_search",
		Label:    "Search Jobs",
		Style:    discordgo.PrimaryButton,
	}

	accentColor := 0x5865F2
	container := discordgo.Container{
		AccentColor: &accentColor,
		Components: []discordgo.MessageComponent{
			discordgo.TextDisplay{Content: header},
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{positionSelect}},
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{levelSelect}},
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{locationSelect}},
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{typeSelect}},
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{searchBtn}},
		},
	}

	return []discordgo.MessageComponent{container}
}

func buildJobSweeperComponents() []discordgo.MessageComponent {
	return buildJobSweeperV2Components(defaultCriteriaState(), "")
}

func positionLabelFromValue(v string) string {
	switch v {
	case "web_dev":
		return "Web Development"
	case "cloud_devops":
		return "Cloud & DevOps"
	case "data_ai":
		return "Data & AI"
	case "mobile_game":
		return "Mobile & Game"
	case "testing_qa":
		return "Testing & QA"
	case "support_security":
		return "IT Support & Security"
	case "architecture":
		return "Software Architecture"
	case "embedded_iot":
		return "Embedded & IoT"
	case "enterprise":
		return "Enterprise Systems"
	case "systems_network":
		return "Systems & Network"
	case "management":
		return "Management & Executive"
	case "design_ux":
		return "Design & UX"
	default:
		return "All Positions"
	}
}

func levelLabelFromValue(v string) string {
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
		return "All Levels"
	}
}

func locationLabelFromValue(v string) string {
	switch v {
	case "hanoi":
		return "Ha Noi"
	case "hcm":
		return "Ho Chi Minh"
	default:
		return "All Locations"
	}
}

func jobTypeLabelFromValue(v string) string {
	switch v {
	case "full_time":
		return "Full-time"
	case "part_time":
		return "Part-time"
	case "contract":
		return "Contract"
	default:
		return "All Job Types"
	}
}

func buildJobEmbed(j job.JobEntry, page, total int) *discordgo.MessageEmbed {
	desc := fmt.Sprintf("### %s\n", j.Title) +
		fmt.Sprintf("> Company: `%s`\n", j.Company) +
		fmt.Sprintf("> Location: `%s`", j.Location)

	if j.Salary != "" {
		desc += fmt.Sprintf("\n> Salary: `%s`", j.Salary)
	}
	if j.Remote {
		desc += "\n> Remote: `Yes`"
	}

	var tagPills []string
	if j.Level != "" {
		tagPills = append(tagPills, fmt.Sprintf("`%s`", j.Level))
	}
	if j.Type != "" {
		tagPills = append(tagPills, fmt.Sprintf("`%s`", j.Type))
	}
	if len(j.Tags) > 0 {
		categories := make([]string, 0, len(j.Tags))
		for cat := range j.Tags {
			categories = append(categories, cat)
		}
		sort.Strings(categories)
		for _, cat := range categories {
			tagPills = append(tagPills, fmt.Sprintf("`%s: %s`", cat, strings.Join(j.Tags[cat], " ")))
		}
	}

	if len(tagPills) > 0 {
		desc += "\n\n**Tags:**\n" + strings.Join(tagPills, " ")
	}

	if j.Summary != "" {
		desc += fmt.Sprintf("\n\n**Overview:**\n%s", j.Summary)
	}

	applyLinks := []string{fmt.Sprintf("[Apply on %s](%s)", strings.Title(j.Site), j.URL)}
	for _, alt := range j.AlternateURLs {
		if alt.URL != "" && alt.Site != "" {
			applyLinks = append(applyLinks, fmt.Sprintf("[%s](%s)", strings.Title(alt.Site), alt.URL))
		}
	}
	desc += "\n\n" + strings.Join(applyLinks, " • ")

	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Job Listing (%d of %d)", page, total),
		Description: desc,
		Color:       0x5865F2,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("JoblessYu • Page %d of %d", page, total),
		},
	}
}
