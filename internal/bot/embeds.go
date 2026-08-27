package bot

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"JoblessYu/internal/job"

	"github.com/bwmarrin/discordgo"
)

func defaultCriteriaState() criteriaState {
	return criteriaState{
		Positions: []string{"all"},
		Levels:    []string{"all"},
		Locations: []string{"all"},
		JobTypes:  []string{"all"},
	}
}

func buildJobSweeperEmbed(state criteriaState, notice string) *discordgo.MessageEmbed {
	positionLabel := positionLabelFromSlice(state.Positions)
	levelLabel := levelLabelFromSlice(state.Levels)
	locationLabel := locationLabelFromSlice(state.Locations)
	jobTypeLabel := jobTypeLabelFromSlice(state.JobTypes)

	desc := "### Filter Configuration\n" +
		fmt.Sprintf("> %s\n", positionLabel) +
		fmt.Sprintf("> %s\n", levelLabel) +
		fmt.Sprintf("> %s\n", locationLabel) +
		fmt.Sprintf("> %s", jobTypeLabel)

	if notice != "" {
		desc += "\n\n*" + notice + "*"
	}

	return &discordgo.MessageEmbed{
		Title:       "Job Sweeper",
		Description: desc,
		Color:       0x5865F2,
	}
}

func isValueSelected(values []string, target string) bool {
	if len(values) == 0 {
		return target == "all"
	}
	clean := filterAllValues(values)
	if len(clean) == 0 {
		return target == "all"
	}
	for _, v := range clean {
		if v == target {
			return true
		}
	}
	return false
}

func filterAllValues(vals []string) []string {
	var clean []string
	for _, v := range vals {
		if v != "" && v != "all" {
			clean = append(clean, v)
		}
	}
	return clean
}

func buildJobSweeperV2Components(state criteriaState, notice string) []discordgo.MessageComponent {
	header := "## Job Sweeper\n" +
		"Select filters below to find job listings (check multiple choices per dropdown)."

	if notice != "" {
		header += "\n\n*" + notice + "*"
	}

	positionOptions := []discordgo.SelectMenuOption{
		{Label: "All Positions", Value: "all"},
		{Label: "IT Management", Value: "management", Description: "PM, Product Manager, CTO, CIO, Director, VP"},
		{Label: "Web Dev", Value: "web_dev", Description: "Backend, Frontend, Fullstack, Golang, Node, React, Vue, PHP"},
		{Label: "Mobile Dev", Value: "mobile_dev", Description: "iOS, Android, Flutter, React Native, Swift, Kotlin"},
		{Label: "Enterprise Systems", Value: "enterprise", Description: "ERP, CRM, SAP, Oracle, Banking, Salesforce"},
		{Label: "Low-Code / No-Code", Value: "lowcode_nocode", Description: "RPA, UiPath, Power Apps, Mendix, OutSystems"},
		{Label: "Architecture", Value: "architecture", Description: "Solutions Architect, Enterprise Architect, Tech Architect"},
		{Label: "Blockchain", Value: "blockchain", Description: "Web3, Smart Contracts, Solidity, Crypto, Ethereum, Rust"},
		{Label: "Game Dev", Value: "game_dev", Description: "Unity, Unreal, Godot, Game Developer, VR/AR"},
		{Label: "Testing & QA", Value: "testing_qa", Description: "QA, Tester, Test Automation, SDET, Cypress, Selenium"},
		{Label: "Data Analytics", Value: "data_analytics", Description: "Data Analyst, BI Analyst, Tableau, Power BI, Looker"},
		{Label: "Data Engineering", Value: "data_engineering", Description: "Big Data, DataOps, MLOps, ETL, Spark, Airflow"},
		{Label: "Data Science & AI", Value: "data_ai", Description: "Machine Learning, AI Engineer, Data Scientist, LLM, GenAI"},
		{Label: "Data Governance", Value: "data_governance", Description: "Data Architect, DBA, Database Administrator"},
		{Label: "Cloud Computing", Value: "cloud", Description: "AWS, Azure, GCP, Cloud Engineer, Cloud Architect"},
		{Label: "Systems & Network", Value: "systems_network", Description: "Sysadmin, Infrastructure, Linux, Network Engineer"},
		{Label: "DevOps & SRE", Value: "devops_sre", Description: "DevOps, Kubernetes, Terraform, SRE, CI/CD, Docker"},
		{Label: "IT Support", Value: "support_helpdesk", Description: "Helpdesk, Technical Support, IT Administrator"},
		{Label: "Cybersecurity", Value: "cybersecurity", Description: "Security Engineer, Penetration Testing, SOC Analyst"},
		{Label: "IT Compliance", Value: "compliance_risk", Description: "Compliance Officer, GRC, IT Auditor, Risk Manager"},
		{Label: "Embedded & IoT", Value: "embedded_iot", Description: "Embedded, Firmware, IoT, Robotics, RTOS, C/C++"},
		{Label: "Product Mgmt", Value: "product_mgmt", Description: "Product Manager, Product Owner, Product Analyst"},
		{Label: "Project Mgmt", Value: "project_mgmt", Description: "Scrum Master, Agile Coach, BrSE, BA, Technical Writer"},
		{Label: "Design & UX", Value: "design_ux", Description: "UI/UX Designer, Product Designer, Figma"},
		{Label: "IT Consulting", Value: "consulting_sales", Description: "IT Consultant, Pre-Sales, Technical Account Manager"},
	}
	for i := range positionOptions {
		if isValueSelected(state.Positions, positionOptions[i].Value) {
			positionOptions[i].Default = true
		}
	}

	levelOptions := []discordgo.SelectMenuOption{
		{Label: "All Levels", Value: "all"},
		{Label: "Intern", Value: "intern", Description: "Internship roles & thực tập sinh"},
		{Label: "Fresher", Value: "fresher", Description: "Fresh graduates & 0-1 years experience"},
		{Label: "Junior", Value: "junior", Description: "1-3 years of experience"},
		{Label: "Middle", Value: "middle", Description: "Mid-level ~3 years experience"},
		{Label: "Senior", Value: "senior", Description: "5+ years experience"},
		{Label: "Lead", Value: "lead", Description: "Lead, manager & director roles"},
	}
	for i := range levelOptions {
		if isValueSelected(state.Levels, levelOptions[i].Value) {
			levelOptions[i].Default = true
		}
	}

	locationOptions := []discordgo.SelectMenuOption{
		{Label: "All Locations", Value: "all"},
		{Label: "Ho Chi Minh", Value: "hcm", Description: "HCM / Saigon"},
		{Label: "Ha Noi", Value: "hanoi", Description: "Ha Noi / HN"},
		{Label: "Da Nang", Value: "danang", Description: "Da Nang"},
		{Label: "Remote", Value: "remote", Description: "Remote / WFH"},
	}
	for i := range locationOptions {
		if isValueSelected(state.Locations, locationOptions[i].Value) {
			locationOptions[i].Default = true
		}
	}

	typeOptions := []discordgo.SelectMenuOption{
		{Label: "All Job Types", Value: "all"},
		{Label: "Full-time", Value: "full_time", Description: "Full-time employment"},
		{Label: "Part-time", Value: "part_time", Description: "Part-time employment"},
		{Label: "Contract", Value: "contract", Description: "Contract / Freelance"},
	}
	for i := range typeOptions {
		if isValueSelected(state.JobTypes, typeOptions[i].Value) {
			typeOptions[i].Default = true
		}
	}

	zero := 0
	maxPositions := 5
	if maxPositions > len(positionOptions) {
		maxPositions = len(positionOptions)
	}

	maxLevels := 5
	if maxLevels > len(levelOptions) {
		maxLevels = len(levelOptions)
	}

	maxLocations := 4
	if maxLocations > len(locationOptions) {
		maxLocations = len(locationOptions)
	}

	maxTypes := 3
	if maxTypes > len(typeOptions) {
		maxTypes = len(typeOptions)
	}

	positionSelect := discordgo.SelectMenu{
		CustomID:    "select_position",
		Placeholder: positionLabelFromSlice(state.Positions),
		MinValues:   &zero,
		MaxValues:   maxPositions,
		Options:     positionOptions,
	}

	levelSelect := discordgo.SelectMenu{
		CustomID:    "select_level",
		Placeholder: levelLabelFromSlice(state.Levels),
		MinValues:   &zero,
		MaxValues:   maxLevels,
		Options:     levelOptions,
	}

	locationSelect := discordgo.SelectMenu{
		CustomID:    "select_location",
		Placeholder: locationLabelFromSlice(state.Locations),
		MinValues:   &zero,
		MaxValues:   maxLocations,
		Options:     locationOptions,
	}

	typeSelect := discordgo.SelectMenu{
		CustomID:    "select_type",
		Placeholder: jobTypeLabelFromSlice(state.JobTypes),
		MinValues:   &zero,
		MaxValues:   maxTypes,
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

func positionLabelFromSlice(vals []string) string {
	clean := filterAllValues(vals)
	if len(clean) == 0 {
		return "Position: All Positions"
	}
	if len(clean) == 1 {
		return "Position: " + positionLabelFromValue(clean[0])
	}
	if len(clean) == 2 {
		return fmt.Sprintf("Positions: %s, %s", positionLabelFromValue(clean[0]), positionLabelFromValue(clean[1]))
	}
	return fmt.Sprintf("Positions: %d selected", len(clean))
}

func levelLabelFromSlice(vals []string) string {
	clean := filterAllValues(vals)
	if len(clean) == 0 {
		return "Level: All Levels"
	}
	if len(clean) == 1 {
		return "Level: " + levelLabelFromValue(clean[0])
	}
	if len(clean) == 2 {
		return fmt.Sprintf("Levels: %s, %s", levelLabelFromValue(clean[0]), levelLabelFromValue(clean[1]))
	}
	return fmt.Sprintf("Levels: %d selected", len(clean))
}

func locationLabelFromSlice(vals []string) string {
	clean := filterAllValues(vals)
	if len(clean) == 0 {
		return "Location: All Locations"
	}
	if len(clean) == 1 {
		return "Location: " + locationLabelFromValue(clean[0])
	}
	if len(clean) == 2 {
		return fmt.Sprintf("Locations: %s, %s", locationLabelFromValue(clean[0]), locationLabelFromValue(clean[1]))
	}
	return fmt.Sprintf("Locations: %d selected", len(clean))
}

func jobTypeLabelFromSlice(vals []string) string {
	clean := filterAllValues(vals)
	if len(clean) == 0 {
		return "Job Type: All Job Types"
	}
	if len(clean) == 1 {
		return "Job Type: " + jobTypeLabelFromValue(clean[0])
	}
	if len(clean) == 2 {
		return fmt.Sprintf("Job Types: %s, %s", jobTypeLabelFromValue(clean[0]), jobTypeLabelFromValue(clean[1]))
	}
	return fmt.Sprintf("Job Types: %d selected", len(clean))
}

func positionLabelFromValue(v string) string {
	switch v {
	case "management":
		return "IT Management"
	case "web_dev":
		return "Web Dev"
	case "mobile_dev":
		return "Mobile Dev"
	case "enterprise":
		return "Enterprise Systems"
	case "lowcode_nocode":
		return "Low-Code/No-Code"
	case "architecture":
		return "Architecture"
	case "blockchain":
		return "Blockchain"
	case "game_dev":
		return "Game Dev"
	case "testing_qa":
		return "Testing & QA"
	case "data_analytics":
		return "Data Analytics"
	case "data_engineering":
		return "Data Engineering"
	case "data_ai":
		return "Data Science & AI"
	case "data_governance":
		return "Data Governance"
	case "cloud":
		return "Cloud Computing"
	case "systems_network":
		return "Systems & Network"
	case "devops_sre":
		return "DevOps & SRE"
	case "support_helpdesk":
		return "IT Support"
	case "cybersecurity":
		return "Cybersecurity"
	case "compliance_risk":
		return "IT Compliance"
	case "embedded_iot":
		return "Embedded & IoT"
	case "product_mgmt":
		return "Product Mgmt"
	case "project_mgmt":
		return "Project Mgmt"
	case "design_ux":
		return "Design & UX"
	case "consulting_sales":
		return "IT Consulting"
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
	case "middle":
		return "Middle"
	case "senior":
		return "Senior"
	case "lead":
		return "Lead / Manager"
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
	case "danang":
		return "Da Nang"
	case "remote":
		return "Remote"
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

func sanitizeInlineValue(v, fallback string) string {
	v = strings.ReplaceAll(v, "\r", " ")
	v = strings.ReplaceAll(v, "\n", " ")
	v = strings.ReplaceAll(v, "`", "'")
	v = strings.Join(strings.Fields(strings.TrimSpace(v)), " ")
	if v == "" {
		return fallback
	}
	return v
}

func buildJobEmbed(j job.JobEntry, page, total int) *discordgo.MessageEmbed {
	title := sanitizeInlineValue(j.Title, "Untitled role")
	company := sanitizeInlineValue(j.Company, "Unknown")
	location := sanitizeInlineValue(j.Location, "Unknown")

	desc := fmt.Sprintf("### %s\n", title) +
		fmt.Sprintf("> Company: `%s`\n", company) +
		fmt.Sprintf("> Location: `%s`", location)

	if j.Salary != "" {
		salary := sanitizeInlineValue(j.Salary, "")
		if salary != "" {
			desc += fmt.Sprintf("\n> Salary: `%s`", salary)
		}
	}
	if j.Remote {
		desc += "\n> Remote: `Yes`"
	}

	var tagPills []string
	if j.Level != "" {
		tagPills = append(tagPills, fmt.Sprintf("`%s`", sanitizeInlineValue(j.Level, "Unknown")))
	}
	if j.Type != "" {
		tagPills = append(tagPills, fmt.Sprintf("`%s`", sanitizeInlineValue(formatJobTypePill(j.Type), "Unknown")))
	}
	if len(j.Tags) > 0 {
		categories := make([]string, 0, len(j.Tags))
		for cat := range j.Tags {
			categories = append(categories, cat)
		}
		sort.Strings(categories)
		for _, cat := range categories {
			tagPills = append(tagPills, fmt.Sprintf("`%s: %s`", sanitizeInlineValue(cat, "tag"), sanitizeInlineValue(strings.Join(j.Tags[cat], " "), "")))
		}
	}

	if len(tagPills) > 0 {
		desc += "\n\n**Tags:**\n" + strings.Join(tagPills, " ")
	}

	if j.Summary != "" {
		desc += fmt.Sprintf("\n\n**Overview:**\n%s", j.Summary)
	}
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

// BuildStatusCardEmbed creates a rich embed for the static availability status card (Abyss Bot pattern).
func BuildStatusCardEmbed(online bool, details StatusDetails) *discordgo.MessageEmbed {
	color := 0x10B981 // Emerald green for Online
	statusHeader := "🟢 ONLINE"
	statusDesc := "JoblessYu is online and actively indexing IT job listings in Vietnam."

	if !online {
		color = 0xEF4444 // Crimson red for Offline
		statusHeader = "🔴 OFFLINE"
		statusDesc = "JoblessYu is currently offline for maintenance or system restart."
	}

	version := details.Version
	if version == "" {
		version = "v1.2.0"
	}

	retention := details.RetentionDays
	if retention <= 0 {
		retention = 30
	}

	return &discordgo.MessageEmbed{
		Title:       statusHeader,
		Description: statusDesc,
		Color:       color,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "📊 System Status",
				Value:  fmt.Sprintf("• **Status**: `%s`\n• **Version**: `%s`\n• **Retention**: `%d Days`", statusHeader, version, retention),
				Inline: true,
			},
			{
				Name:   "📦 Active Job Pool",
				Value:  fmt.Sprintf("• **Active Jobs**: `%d`", details.ActiveJobs),
				Inline: true,
			},
			{
				Name:   "💡 Quick Start",
				Value:  "Type `/jobs` in any channel to open the interactive job filter panel.",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("JoblessYu Service Monitor • %s", time.Now().Format("02/01/2006 15:04:05")),
		},
	}
}

// BuildDailyAnnouncementEmbed creates a rich summary embed for static 5:00 AM ICT daily job updates.
func BuildDailyAnnouncementEmbed(summary DailyScrapeSummary) *discordgo.MessageEmbed {
	locICT := time.FixedZone("ICT", 7*3600)
	lastScrapeStr := "N/A"
	if !summary.RunTime.IsZero() {
		lastScrapeStr = summary.RunTime.In(locICT).Format("02 Jan 2006 15:04 ICT")
	}
	nextScrapeStr := CalculateNextScrapeTime(time.Now()).In(locICT).Format("02 Jan 2006 15:04 ICT")

	addedStr := "+0 new listings"
	if summary.InsertedCount > 0 {
		addedStr = fmt.Sprintf("+%d new listings", summary.InsertedCount)
	} else if summary.Recent24hAdded > 0 {
		addedStr = fmt.Sprintf("+%d new listings today", summary.Recent24hAdded)
	}

	fields := []*discordgo.MessageEmbedField{
		{
			Name:   "📅 Timestamps",
			Value:  fmt.Sprintf("• **Last Scrape**: `%s`\n• **Next Scrape**: `%s`", lastScrapeStr, nextScrapeStr),
			Inline: true,
		},
		{
			Name:   "📦 Job Pool Insights",
			Value:  fmt.Sprintf("• **Fresh Roles Today**: `%s`\n• **Total Active Pool**: `%d listings`", addedStr, summary.TotalActiveJobs),
			Inline: true,
		},
		{
			Name: "🎯 Experience Level Breakdown",
			Value: fmt.Sprintf("• 🎓 **Intern / Fresher**: `%d roles`  • 🌱 **Junior / Mid**: `%d roles`\n• 🚀 **Senior**: `%d roles`  • ⚡ **Lead / Manager**: `%d roles`",
				summary.InternCount, summary.JuniorCount, summary.SeniorCount, summary.LeadCount),
			Inline: false,
		},
		{
			Name: "📍 Top Locations",
			Value: fmt.Sprintf("• 🏙️ **Ho Chi Minh**: `%d roles`  • 🏛️ **Ha Noi**: `%d roles`  • 🌊 **Da Nang**: `%d roles`  • 💻 **Remote**: `%d roles`",
				summary.HCMCount, summary.HanoiCount, summary.DaNangCount, summary.RemoteCount),
			Inline: false,
		},
		{
			Name:   "🚀 Explore Jobs",
			Value:  "Type `/jobs` anywhere on Discord to filter by position, level, location, and job type! (Results are ephemeral — only visible to you).",
			Inline: false,
		},
	}

	return &discordgo.MessageEmbed{
		Title:       "🌅 SCRAPE SUMMARY",
		Description: "Real-time IT job listings index for Vietnam software developers & engineers.",
		Color:       0xF59E0B, // Gold / Amber
		Fields:      fields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("JoblessYu Scrape Monitor • %s", time.Now().Format("02/01/2006 15:04:05")),
		},
	}
}
