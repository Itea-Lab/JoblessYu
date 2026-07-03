package bot

import (
	"fmt"
	"sort"
	"strings"

	"JoblessYu/internal/domain"

	"github.com/bwmarrin/discordgo"
)

func buildJobEmbed(job domain.JobEntry, page, total int) *discordgo.MessageEmbed {
	desc := fmt.Sprintf("• **Role:** %s\n• **Company:** %s\n• **Location:** %s", job.Title, job.Company, job.Location)

	tags := ""
	if job.Level != "" {
		tags += fmt.Sprintf("`%s` ", job.Level)
	}
	if job.Type != "" {
		tags += fmt.Sprintf("`%s` ", job.Type)
	}

	// Append detected skill/tool tags, grouped by category so the embed's
	// Tags line stays readable (e.g. `Cloud: AWS IAM` `IaC: CloudFormation`).
	if len(job.Tags) > 0 {
		categories := make([]string, 0, len(job.Tags))
		for cat := range job.Tags {
			categories = append(categories, cat)
		}
		sort.Strings(categories)
		for _, cat := range categories {
			tags += fmt.Sprintf("`%s: %s` ", cat, strings.Join(job.Tags[cat], " "))
		}
	}

	if tags = strings.TrimSpace(tags); tags != "" {
		desc += fmt.Sprintf("\n• **Tags:** %s", tags)
	}

	desc += fmt.Sprintf("\n• **Apply:** [Open job](%s)", job.URL)

	return &discordgo.MessageEmbed{
		Title:       job.Company,
		Description: desc,
		Color:       0x5865F2,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Page %d of %d", page, total),
		},
	}
}
