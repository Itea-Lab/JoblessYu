package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"JoblessYu/internal/config"

	"github.com/bwmarrin/discordgo"
)

// StatusDetails contains parameters for rendering the static availability status card.
type StatusDetails struct {
	ActiveJobs     int
	LastScrapeTime time.Time
	NextScrapeTime time.Time
	RetentionDays  int
	Version        string
}

var locICT = time.FixedZone("ICT", 7*3600)

// CalculateNextScrapeTime computes the next occurrence of daily 05:00 AM ICT from now.
func CalculateNextScrapeTime(now time.Time) time.Time {
	ictNow := now.In(locICT)
	next := time.Date(ictNow.Year(), ictNow.Month(), ictNow.Day(), 5, 0, 0, 0, locICT)
	if !ictNow.Before(next) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}

// DailyScrapeSummary contains parameters for the 5:00 AM daily job update announcement.
type DailyScrapeSummary struct {
	RunTime         time.Time
	TotalDuration   time.Duration
	JobspyCount     int
	CollyCount      int
	InsertedCount   int
	MergedCount     int
	EnrichedCount   int
	TotalActiveJobs int
	Recent24hAdded  int
	InternCount     int
	JuniorCount     int
	SeniorCount     int
	LeadCount       int
	HCMCount        int
	HanoiCount      int
	DaNangCount     int
	RemoteCount     int
}

// Notifier manages Discord lifecycle status cards, public announcements, and operational webhooks.
type Notifier struct {
	session           *discordgo.Session
	cfg               *config.Config
	statusMsgID       string
	dailySummaryMsgID string
	mu                sync.Mutex
}

func NewNotifier(session *discordgo.Session, cfg *config.Config) *Notifier {
	return &Notifier{
		session: session,
		cfg:     cfg,
	}
}

// resolveChannelID validates that targetChannelID is explicitly configured.
// It enforces the Fail-Fast principle by refusing to auto-discover or guess target channels.
func (n *Notifier) resolveChannelID(targetChannelID string) (string, error) {
	chID := targetChannelID
	if chID == "" {
		chID = n.cfg.DiscordChannelID
	}
	if chID == "" {
		return "", fmt.Errorf("target channel ID is empty; please specify DISCORD_CHANNEL_ID in .env")
	}

	if n.session == nil {
		return "", fmt.Errorf("discord session is nil")
	}

	return chID, nil
}

// GetResolvedChannelID returns the channel ID that will be used for notifications.
func (n *Notifier) GetResolvedChannelID(preferredChannelID string) string {
	chID, err := n.resolveChannelID(preferredChannelID)
	if err != nil {
		return ""
	}
	return chID
}

// UpdateStatusCard creates or edits the single static availability status card in DISCORD_CHANNEL_ID.
func (n *Notifier) UpdateStatusCard(online bool, details StatusDetails) error {
	if n.session == nil {
		return fmt.Errorf("discord session is nil")
	}

	channelID, err := n.resolveChannelID(n.cfg.DiscordChannelID)
	if err != nil {
		return fmt.Errorf("status channel resolution failed: %w", err)
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	embed := BuildStatusCardEmbed(online, details)
	botUserID := ""
	if n.session.State != nil && n.session.State.User != nil {
		botUserID = n.session.State.User.ID
	}

	// 1. Try editing known message ID if already tracked in memory
	if n.statusMsgID != "" {
		_, editErr := n.session.ChannelMessageEditEmbed(channelID, n.statusMsgID, embed)
		if editErr == nil {
			slog.Info("Edited static status card (in-memory ID)", "channel_id", channelID, "message_id", n.statusMsgID, "online", online)
			return nil
		}
	}

	// 2. Search recent channel messages for existing status card from JoblessYu
	msgs, fetchErr := n.session.ChannelMessages(channelID, 50, "", "", "")
	if fetchErr == nil {
		for _, msg := range msgs {
			if botUserID != "" && msg.Author != nil && msg.Author.ID != botUserID {
				continue
			}
			if len(msg.Embeds) > 0 {
				emb := msg.Embeds[0]
				if (emb.Footer != nil && strings.Contains(emb.Footer.Text, "JoblessYu Service Monitor")) ||
					strings.Contains(emb.Title, "ONLINE") || strings.Contains(emb.Title, "OFFLINE") {
					_, editErr := n.session.ChannelMessageEditEmbed(channelID, msg.ID, embed)
					if editErr == nil {
						n.statusMsgID = msg.ID
						slog.Info("Edited existing static status card", "channel_id", channelID, "message_id", msg.ID, "online", online)
						return nil
					}
				}
			}
		}
	}

	// 3. Search pinned messages if recent search didn't find it
	pinned, pinErr := n.session.ChannelMessagesPinned(channelID)
	if pinErr == nil && len(pinned) > 0 {
		for _, msg := range pinned {
			if botUserID != "" && msg.Author != nil && msg.Author.ID != botUserID {
				continue
			}
			_, editErr := n.session.ChannelMessageEditEmbed(channelID, msg.ID, embed)
			if editErr == nil {
				n.statusMsgID = msg.ID
				slog.Info("Edited pinned static status card", "channel_id", channelID, "message_id", msg.ID, "online", online)
				return nil
			}
		}
	}

	// 4. Create single new static status message if no existing status card was found
	msg, sendErr := n.session.ChannelMessageSendEmbed(channelID, embed)
	if sendErr != nil {
		slog.Warn("Failed to send status card embed", "channel_id", channelID, "err", sendErr)
		return sendErr
	}

	n.statusMsgID = msg.ID
	_ = n.session.ChannelMessagePin(channelID, msg.ID) // Best-effort pin (ignore 403)

	slog.Info("Created static status card", "channel_id", channelID, "message_id", msg.ID, "online", online)
	return nil
}

// PostDailyScrapeAnnouncement posts or edits the static 5:00 AM daily job update summary card.
func (n *Notifier) PostDailyScrapeAnnouncement(summary DailyScrapeSummary) error {
	return n.UpdateDailySummaryCard(summary)
}

// UpdateDailySummaryCard creates or edits the static daily pipeline summary card in DISCORD_CHANNEL_ID.
func (n *Notifier) UpdateDailySummaryCard(summary DailyScrapeSummary) error {
	if n.session == nil {
		return fmt.Errorf("discord session is nil")
	}

	channelID, err := n.resolveChannelID(n.cfg.DiscordChannelID)
	if err != nil {
		return fmt.Errorf("summary channel resolution failed: %w", err)
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	embed := BuildDailyAnnouncementEmbed(summary)
	botUserID := ""
	if n.session.State != nil && n.session.State.User != nil {
		botUserID = n.session.State.User.ID
	}

	var existingSummaryID string

	// 1. Check known message ID if already tracked in memory
	if n.dailySummaryMsgID != "" {
		existingSummaryID = n.dailySummaryMsgID
	}

	// 2. Search recent channel messages if not tracked in memory
	if existingSummaryID == "" {
		msgs, fetchErr := n.session.ChannelMessages(channelID, 50, "", "", "")
		if fetchErr == nil {
			for _, msg := range msgs {
				if botUserID != "" && msg.Author != nil && msg.Author.ID != botUserID {
					continue
				}
				if len(msg.Embeds) > 0 {
					emb := msg.Embeds[0]
					if strings.Contains(emb.Title, "SCRAPE SUMMARY") || strings.Contains(emb.Title, "DAILY PIPELINE SUMMARY") || strings.Contains(emb.Title, "Daily IT Job List Updated") ||
						(emb.Footer != nil && (strings.Contains(emb.Footer.Text, "JoblessYu Scrape Monitor") || strings.Contains(emb.Footer.Text, "JoblessYu Daily Monitor"))) {
						existingSummaryID = msg.ID
						break
					}
				}
			}
		}
	}

	// 3. Search pinned messages if recent search didn't find it
	if existingSummaryID == "" {
		pinned, pinErr := n.session.ChannelMessagesPinned(channelID)
		if pinErr == nil && len(pinned) > 0 {
			for _, msg := range pinned {
				if botUserID != "" && msg.Author != nil && msg.Author.ID != botUserID {
					continue
				}
				if len(msg.Embeds) > 0 && (strings.Contains(msg.Embeds[0].Title, "SCRAPE SUMMARY") || strings.Contains(msg.Embeds[0].Title, "DAILY PIPELINE SUMMARY") || strings.Contains(msg.Embeds[0].Title, "Daily IT Job List Updated")) {
					existingSummaryID = msg.ID
					break
				}
			}
		}
	}

	// 4. Ensure correct ordering: Card 2 must be newer than Card 1 (positioned below Status Card).
	// If existingSummaryID is older than n.statusMsgID, delete old Card 2 and re-send at bottom!
	if existingSummaryID != "" && n.statusMsgID != "" && existingSummaryID < n.statusMsgID {
		slog.Info("Daily summary card is older than status card, re-sending below status card", "old_summary_id", existingSummaryID, "status_id", n.statusMsgID)
		_ = n.session.ChannelMessageDelete(channelID, existingSummaryID)
		existingSummaryID = ""
		n.dailySummaryMsgID = ""
	}

	// 5. Edit existing or send new summary card
	if existingSummaryID != "" {
		_, editErr := n.session.ChannelMessageEditEmbed(channelID, existingSummaryID, embed)
		if editErr == nil {
			n.dailySummaryMsgID = existingSummaryID
			slog.Info("Edited static daily summary card", "channel_id", channelID, "message_id", existingSummaryID)
			return nil
		}
	}

	// Create new static daily summary message at the bottom of the channel
	msg, sendErr := n.session.ChannelMessageSendEmbed(channelID, embed)
	if sendErr != nil {
		slog.Warn("Failed to send daily summary embed", "channel_id", channelID, "err", sendErr)
		return sendErr
	}

	n.dailySummaryMsgID = msg.ID
	_ = n.session.ChannelMessagePin(channelID, msg.ID)

	slog.Info("Created static daily summary card at bottom of channel", "channel_id", channelID, "message_id", msg.ID)
	return nil
}

// PostDevLogWebhook posts structured dev operational logs to DISCORD_LOG_WEBHOOK_URL or fallback channel.
func (n *Notifier) PostDevLogWebhook(title string, description string, color int, fields map[string]string) {
	if n.cfg != nil && n.cfg.DiscordLogWebhookURL != "" {
		var discordFields []map[string]interface{}
		for k, v := range fields {
			discordFields = append(discordFields, map[string]interface{}{
				"name":   k,
				"value":  v,
				"inline": true,
			})
		}

		payload := map[string]interface{}{
			"embeds": []map[string]interface{}{
				{
					"title":       title,
					"description": description,
					"color":       color,
					"fields":      discordFields,
					"footer": map[string]string{
						"text": "JoblessYu Dev Log • " + time.Now().UTC().Format("15:04:05 UTC"),
					},
				},
			},
		}

		body, err := json.Marshal(payload)
		if err == nil {
			req, reqErr := http.NewRequest("POST", n.cfg.DiscordLogWebhookURL, bytes.NewBuffer(body))
			if reqErr == nil {
				req.Header.Set("Content-Type", "application/json")
				client := &http.Client{Timeout: 5 * time.Second}
				resp, doErr := client.Do(req)
				if doErr == nil {
					resp.Body.Close()
					return
				}
			}
		}
	}

	// Fallback if webhook is empty or failed: send directly to resolved server text channel
	channelID, err := n.resolveChannelID("")
	if err != nil || n.session == nil {
		return
	}

	var embedFields []*discordgo.MessageEmbedField
	for k, v := range fields {
		embedFields = append(embedFields, &discordgo.MessageEmbedField{
			Name:   k,
			Value:  v,
			Inline: true,
		})
	}

	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       color,
		Fields:      embedFields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "JoblessYu Dev Log • " + time.Now().UTC().Format("15:04:05 UTC"),
		},
	}
	_, _ = n.session.ChannelMessageSendEmbed(channelID, embed)
}
