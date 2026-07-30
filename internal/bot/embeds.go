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
		{Label: "IT Executive & Management", Value: "management"},
		{Label: "Web Application Development", Value: "web_dev"},
		{Label: "Mobile Application Development", Value: "mobile_dev"},
		{Label: "Core / Enterprise Systems", Value: "enterprise"},
		{Label: "Low-Code / No-Code Dev", Value: "lowcode_nocode"},
		{Label: "Technical Architecture", Value: "architecture"},
		{Label: "Blockchain Development", Value: "blockchain"},
		{Label: "Game Development", Value: "game_dev"},
		{Label: "Software Testing & QA", Value: "testing_qa"},
		{Label: "Data Analytics & BI", Value: "data_analytics"},
		{Label: "Data Engineering", Value: "data_engineering"},
		{Label: "Data Science & AI / ML", Value: "data_ai"},
		{Label: "Data Management & Governance", Value: "data_governance"},
		{Label: "Cloud Computing", Value: "cloud"},
		{Label: "Systems & Network Admin", Value: "systems_network"},
		{Label: "DevOps & Site Reliability (SRE)", Value: "devops_sre"},
		{Label: "IT Support & Helpdesk", Value: "support_helpdesk"},
		{Label: "Cybersecurity", Value: "cybersecurity"},
		{Label: "IT Compliance & Risk", Value: "compliance_risk"},
		{Label: "Embedded, IoT & Robotics", Value: "embedded_iot"},
		{Label: "Product Management", Value: "product_mgmt"},
		{Label: "Project Management & Tech Comm", Value: "project_mgmt"},
		{Label: "Design & User Experience", Value: "design_ux"},
		{Label: "IT Consulting & Sales", Value: "consulting_sales"},
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
	case "management":
		return "IT Executive & Management"
	case "web_dev":
		return "Web Application Development"
	case "mobile_dev":
		return "Mobile Application Development"
	case "enterprise":
		return "Core / Enterprise Systems"
	case "lowcode_nocode":
		return "Low-Code / No-Code Dev"
	case "architecture":
		return "Technical Architecture"
	case "blockchain":
		return "Blockchain Development"
	case "game_dev":
		return "Game Development"
	case "testing_qa":
		return "Software Testing & QA"
	case "data_analytics":
		return "Data Analytics & BI"
	case "data_engineering":
		return "Data Engineering"
	case "data_ai":
		return "Data Science & AI / ML"
	case "data_governance":
		return "Data Management & Governance"
	case "cloud":
		return "Cloud Computing"
	case "systems_network":
		return "Systems & Network Admin"
	case "devops_sre":
		return "DevOps & Site Reliability (SRE)"
	case "support_helpdesk":
		return "IT Support & Helpdesk"
	case "cybersecurity":
		return "Cybersecurity"
	case "compliance_risk":
		return "IT Compliance & Risk"
	case "embedded_iot":
		return "Embedded, IoT & Robotics"
	case "product_mgmt":
		return "Product Management"
	case "project_mgmt":
		return "Project Management & Tech Comm"
	case "design_ux":
		return "Design & User Experience"
	case "consulting_sales":
		return "IT Consulting & Sales"
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

func formatJobTypePill(t string) string {
	val := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(t, "-", ""), " ", ""))
	switch {
	case strings.Contains(val, "fulltime"):
		return "Full-time"
	case strings.Contains(val, "parttime"):
		return "Part-time"
	case strings.Contains(val, "contract"):
		return "Contract"
	default:
		return t
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
		tagPills = append(tagPills, fmt.Sprintf("`%s`", formatJobTypePill(j.Type)))
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
	desc = truncateForDiscord(desc, 3900)

	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Job Listing (%d of %d)", page, total),
		Description: desc,
		Color:       0x5865F2,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("JoblessYu • Page %d of %d", page, total),
		},
	}
}

func truncateForDiscord(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
