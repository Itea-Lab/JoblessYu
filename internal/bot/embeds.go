package bot

import (
	"fmt"
	"sort"
	"strings"

	"JoblessYu/internal/job"

	"github.com/bwmarrin/discordgo"
)

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
