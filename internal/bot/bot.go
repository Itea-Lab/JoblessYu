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
	jobs        []job.JobEntry
	currentPage int
	insertedAt  time.Time
}

type criteriaState struct {
	PositionValue string
	LevelValue    string
	LocationValue string
	JobTypeValue  string
}

type cachedCriteria struct {
	state      criteriaState
	insertedAt time.Time
}

// jobService is the contract Bot needs from the service layer. Defined
// here (consumer-side) per Go's "accept interfaces" convention; the
// concrete *service.JobService satisfies it structurally. This seam lets
// bot handlers be unit-tested with a fake — no real Postgres or extractor
// needed.
type jobService interface {
	FetchAndProcessJobs(ctx context.Context, q job.JobQuery) ([]job.JobEntry, error)
	GetScrapeStats(ctx context.Context) (int, time.Time, error)
}

type Bot struct {
	session    *discordgo.Session
	cfg        *config.Config
	jobService jobService
	notifier   *Notifier

	jobsCache map[string]cachedJobs
	uiState   map[string]cachedCriteria
	cacheLock sync.RWMutex

	stopJanitor chan struct{}
}

var commands = []*discordgo.ApplicationCommand{
	{
		Name:        "jobs",
		Description: "Open interactive job filters and browse results",
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

	notifier := NewNotifier(session, cfg)

	bot := &Bot{
		session:     session,
		cfg:         cfg,
		jobService:  jobService,
		notifier:    notifier,
		jobsCache:   make(map[string]cachedJobs),
		uiState:     make(map[string]cachedCriteria),
		stopJanitor: make(chan struct{}),
	}

	bot.session.AddHandler(bot.handleInteraction)

	go bot.cacheJanitor()

	return bot, nil
}

func (b *Bot) Notifier() *Notifier {
	return b.notifier
}

func (b *Bot) fetchStatusDetails(ctx context.Context) StatusDetails {
	details := StatusDetails{
		RetentionDays:  b.cfg.JobRetentionDays,
		Version:        "v1.2.0",
		NextScrapeTime: CalculateNextScrapeTime(time.Now()),
	}

	if b.jobService != nil {
		active, lastScrape, err := b.jobService.GetScrapeStats(ctx)
		if err == nil {
			details.ActiveJobs = active
			details.LastScrapeTime = lastScrape
		} else {
			log.Println("bot: failed to query scrape stats for status card:", err)
		}
	}
	return details
}

func (b *Bot) fetchDailySummaryDetails(ctx context.Context) DailyScrapeSummary {
	summary := DailyScrapeSummary{}
	if b.jobService != nil {
		if statsService, ok := b.jobService.(*job.JobService); ok {
			pStats, err := statsService.GetPipelineSummaryStats(ctx)
			if err == nil {
				summary.RunTime = pStats.LastScrapeTime
				summary.TotalActiveJobs = pStats.TotalActiveJobs
				summary.Recent24hAdded = pStats.Recent24hAdded
				summary.Recent24hEnriched = pStats.Recent24hEnriched
				summary.RecentJobspyCount = pStats.RecentJobspyCount
				summary.RecentCollyCount = pStats.RecentCollyCount
			}
		}
	}
	return summary
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
	for id, c := range b.uiState {
		if now.Sub(c.insertedAt) > jobsCacheTTL {
			delete(b.uiState, id)
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

	// Update static availability card (Card 1) & daily summary card (Card 2) with real DB stats
	if b.notifier != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			details := b.fetchStatusDetails(ctx)
			if err := b.notifier.UpdateStatusCard(true, details); err != nil {
				log.Println("Note: Status card update:", err)
			}
			summary := b.fetchDailySummaryDetails(ctx)
			if summary.TotalActiveJobs == 0 {
				summary.TotalActiveJobs = details.ActiveJobs
			}
			if err := b.notifier.UpdateDailySummaryCard(summary); err != nil {
				log.Println("Note: Daily summary card update:", err)
			}
		}()

		// Hybrid Real-time DB Sync: combines Postgres LISTEN triggers with a 30s count change monitor
		if b.jobService != nil {
			go func() {
				lastKnownCount := -1
				ticker := time.NewTicker(30 * time.Second)
				defer ticker.Stop()

				updateCardsIfChanged := func() {
					rCtx, rCancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer rCancel()
					details := b.fetchStatusDetails(rCtx)
					if details.ActiveJobs != lastKnownCount {
						lastKnownCount = details.ActiveJobs
						_ = b.notifier.UpdateStatusCard(true, details)
						summary := b.fetchDailySummaryDetails(rCtx)
						if summary.TotalActiveJobs == 0 {
							summary.TotalActiveJobs = details.ActiveJobs
						}
						_ = b.notifier.UpdateDailySummaryCard(summary)
					}
				}

				// Initial check to prime lastKnownCount
				updateCardsIfChanged()

				// 1. Postgres LISTEN event listener (push notifications)
				if statsService, ok := b.jobService.(*job.JobService); ok {
					ctx, cancel := context.WithCancel(context.Background())
					go func() {
						<-b.stopJanitor
						cancel()
					}()
					var debounceTimer *time.Timer
					go statsService.ListenForJobChanges(ctx, func() {
						if debounceTimer != nil {
							debounceTimer.Stop()
						}
						debounceTimer = time.AfterFunc(500*time.Millisecond, func() {
							updateCardsIfChanged()
						})
					})
				}

				// 2. 30s fail-safe polling ticker (handles Neon PgBouncer transaction pooler drops)
				for {
					select {
					case <-ticker.C:
						updateCardsIfChanged()
					case <-b.stopJanitor:
						return
					}
				}
			}()
		}
	}

	return nil
}

func (b *Bot) CloseWithoutOffline() {
	close(b.stopJanitor)
	b.session.Close()
}

func (b *Bot) Stop() {
	close(b.stopJanitor)

	// Update static availability card to Offline with real DB stats
	if b.notifier != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		details := b.fetchStatusDetails(ctx)
		cancel()
		_ = b.notifier.UpdateStatusCard(false, details)
	}

	b.session.Close()
}
