package bot

import (
	"log"
	"sync"
	"time"

	"JoblessYu/internal/config"
	"JoblessYu/internal/domain"
	"JoblessYu/internal/service"

	"github.com/bwmarrin/discordgo"
)

const (
	jobsCacheTTL        = 30 * time.Minute
	jobsCacheSweepEvery = 1 * time.Minute
)

type cachedJobs struct {
	jobs       []domain.JobEntry
	insertedAt time.Time
}

type Bot struct {
	session    *discordgo.Session
	cfg        *config.Config
	jobService *service.JobService

	jobsCache map[string]cachedJobs
	cacheLock sync.RWMutex

	stopJanitor chan struct{}
}

var commands = []*discordgo.ApplicationCommand{
	{
		Name:        "jobs",
		Description: "Browse IT Support jobs with pagination",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "level",
				Description: "Job experience level (optional)",
				Required:    false,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "Intern", Value: "Intern"},
					{Name: "Fresher", Value: "Fresher"},
					{Name: "Junior", Value: "Junior"},
					{Name: "Senior", Value: "Senior"},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "type",
				Description: "Job type (optional)",
				Required:    false,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "Fulltime", Value: "Fulltime"},
					{Name: "Parttime", Value: "Parttime"},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "location",
				Description: "Job location (optional)",
				Required:    false,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "HCM (Ho Chi Minh)", Value: "HCM"},
					{Name: "HN (Ha Noi)", Value: "HN"},
				},
			},
		},
	},
}

func NewBot(cfg *config.Config, jobService *service.JobService) (*Bot, error) {
	token := "Bot " + cfg.DiscordToken
	session, err := discordgo.New(token)
	if err != nil {
		return nil, err
	}

	session.Identify.Intents =
		discordgo.IntentsGuildMessages |
			discordgo.IntentsDirectMessages |
			discordgo.IntentsGuilds |
			discordgo.IntentsMessageContent

	bot := &Bot{
		session:     session,
		cfg:         cfg,
		jobService:  jobService,
		jobsCache:   make(map[string]cachedJobs),
		stopJanitor: make(chan struct{}),
	}

	bot.session.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.Bot {
			return
		}
		if m.Content == "!testembed" {
			embed := &discordgo.MessageEmbed{
				Title:       "✅ Embed Test",
				Description: "If you can see this, embeds are working correctly.",
				Color:       0x57F287,
			}
			s.ChannelMessageSendEmbed(m.ChannelID, embed)
		}
	})

	bot.session.AddHandler(bot.handleInteraction)

	go bot.cacheJanitor()

	return bot, nil
}

func (b *Bot) cacheJanitor() {
	ticker := time.NewTicker(jobsCacheSweepEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			b.evictExpired()
		case <-b.stopJanitor:
			return
		}
	}
}

func (b *Bot) evictExpired() {
	now := time.Now()

	b.cacheLock.Lock()
	defer b.cacheLock.Unlock()

	for id, c := range b.jobsCache {
		if now.Sub(c.insertedAt) > jobsCacheTTL {
			delete(b.jobsCache, id)
		}
	}
}

func (b *Bot) Start() error {
	if err := b.session.Open(); err != nil {
		return err
	}

	// BulkOverwrite the guild command set atomically. Handles additions,
	// removals, renames, option edits, and choice updates in one call, so
	// stale options (e.g. a renamed `time` -> `type`) don't linger in Discord.
	if _, err := b.session.ApplicationCommandBulkOverwrite(
		b.session.State.User.ID, b.cfg.DiscordGuild, commands,
	); err != nil {
		log.Println("Failed to overwrite slash commands:", err)
	}

	return nil
}

func (b *Bot) Stop() {
	close(b.stopJanitor)
	b.session.Close()
}
