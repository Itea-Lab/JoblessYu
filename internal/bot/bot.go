package bot

import (
	"context"
	"log"
	"sync"
	"time"

	"JoblessYu/internal/config"
	"JoblessYu/internal/job"

	"github.com/bwmarrin/discordgo"
)

const (
	jobsCacheTTL        = 30 * time.Minute
	jobsCacheSweepEvery = 1 * time.Minute
)

type cachedJobs struct {
	jobs       []job.JobEntry
	insertedAt time.Time
}

// jobService is the contract Bot needs from the service layer. Defined
// here (consumer-side) per Go's "accept interfaces" convention; the
// concrete *service.JobService satisfies it structurally. This seam lets
// bot handlers be unit-tested with a fake — no real Postgres or extractor
// needed.
type jobService interface {
	FetchAndProcessJobs(ctx context.Context, q job.JobQuery) ([]job.JobEntry, error)
}

type Bot struct {
	session    *discordgo.Session
	cfg        *config.Config
	jobService jobService

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
			{
				Type:        discordgo.ApplicationCommandOptionBoolean,
				Name:        "include_unknown",
				Description: "Include jobs whose level could not be detected (default false)",
				Required:    false,
			},
		},
	},
}

func NewBot(cfg *config.Config, jobService jobService) (*Bot, error) {
	token := "Bot " + cfg.DiscordToken
	session, err := discordgo.New(token)
	if err != nil {
		return nil, err
	}

	session.Identify.Intents =
		discordgo.IntentsGuildMessages |
			discordgo.IntentsDirectMessages |
			discordgo.IntentsGuilds

	bot := &Bot{
		session:     session,
		cfg:         cfg,
		jobService:  jobService,
		jobsCache:   make(map[string]cachedJobs),
		stopJanitor: make(chan struct{}),
	}

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
