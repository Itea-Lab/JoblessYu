package bot

import (
	"fmt"
	"strings"

	"JoblessYu/internal/domain"

	"github.com/bwmarrin/discordgo"
)

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "..."
}

func buildJobEmbed(job domain.JobEntry, page, total int) *discordgo.MessageEmbed {
	desc := fmt.Sprintf("• **Role:** %s\n• **Company:** %s\n• **Location:** %s", job.Title, job.Company, job.Location)

	tags := ""
	if job.Level != "" {
		tags += fmt.Sprintf("`%s` ", job.Level)
	}
	if job.Type != "" {
		tags += fmt.Sprintf("`%s`", job.Type)
	}
	if tags != "" {
		desc += fmt.Sprintf("\n• **Tags:** %s", strings.TrimSpace(tags))
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
