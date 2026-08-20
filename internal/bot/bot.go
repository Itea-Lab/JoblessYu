package bot

import (
	"context"
	"fmt"
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
	jobsCacheMaxEntries = 128
	uiStateMaxEntries   = 256
)

type cachedJobs struct {
	jobs        []job.JobEntry
	currentPage int
	insertedAt  time.Time
}

type criteriaState struct {
	Positions []string
	Levels    []string
	Locations []string
	JobTypes  []string
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
	stopOnce    sync.Once
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
				summary.InternCount = pStats.InternCount
				summary.JuniorCount = pStats.JuniorCount
				summary.SeniorCount = pStats.SeniorCount
				summary.LeadCount = pStats.LeadCount
				summary.HCMCount = pStats.HCMCount
				summary.HanoiCount = pStats.HanoiCount
				summary.DaNangCount = pStats.DaNangCount
				summary.RemoteCount = pStats.RemoteCount
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
	b.trimCachesLocked()
}

// trimCachesLocked enforces hard entry limits in addition to the TTL. A busy
// Discord server can create many independent ephemeral searches before the
// one-minute janitor runs; without a cap, each search retains up to 1,000 full
// JobEntry values (including descriptions) in memory.
func (b *Bot) trimCachesLocked() {
	for len(b.jobsCache) > jobsCacheMaxEntries {
		var oldestID string
		var oldest time.Time
		for id, entry := range b.jobsCache {
			if oldestID == "" || entry.insertedAt.Before(oldest) {
				oldestID = id
				oldest = entry.insertedAt
			}
		}
		if oldestID == "" {
			break
		}
		delete(b.jobsCache, oldestID)
	}

	for len(b.uiState) > uiStateMaxEntries {
		var oldestID string
		var oldest time.Time
		for id, entry := range b.uiState {
			if oldestID == "" || entry.insertedAt.Before(oldest) {
				oldestID = id
				oldest = entry.insertedAt
			}
		}
		if oldestID == "" {
			break
		}
		delete(b.uiState, oldestID)
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
		return fmt.Errorf("failed to register slash commands with Discord: %w", err)
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
				updateRequests := make(chan struct{}, 1)
				requestUpdate := func() {
					select {
					case updateRequests <- struct{}{}:
					default:
						// A refresh is already queued or being debounced.
					}
				}

				refreshCardsIfChanged := func() {
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
				refreshCardsIfChanged()

				// 1. Postgres LISTEN event listener (push notifications)
				listenerCtx, listenerCancel := context.WithCancel(context.Background())
				defer listenerCancel()
				go func() {
					select {
					case <-b.stopJanitor:
						listenerCancel()
					case <-listenerCtx.Done():
					}
				}()
				if statsService, ok := b.jobService.(*job.JobService); ok {
					go statsService.ListenForJobChanges(listenerCtx, requestUpdate)
				}

				// 2. 30s fail-safe polling ticker (handles Neon PgBouncer transaction pooler drops)
				var debounceTimer *time.Timer
				var debounceC <-chan time.Time
				for {
					select {
					case <-ticker.C:
						refreshCardsIfChanged()
					case <-updateRequests:
						if debounceTimer == nil {
							debounceTimer = time.NewTimer(500 * time.Millisecond)
						} else {
							if !debounceTimer.Stop() {
								select {
								case <-debounceTimer.C:
								default:
								}
							}
							debounceTimer.Reset(500 * time.Millisecond)
						}
						debounceC = debounceTimer.C
					case <-debounceC:
						debounceC = nil
						refreshCardsIfChanged()
					case <-b.stopJanitor:
						if debounceTimer != nil {
							debounceTimer.Stop()
						}
						return
					}
				}
			}()
		}
	}

	return nil
}

func (b *Bot) signalStop() bool {
	stopped := false
	b.stopOnce.Do(func() {
		close(b.stopJanitor)
		stopped = true
	})
	return stopped
}

func (b *Bot) CloseWithoutOffline() {
	if !b.signalStop() {
		return
	}
	b.session.Close()
}

func (b *Bot) Stop() {
	if !b.signalStop() {
		return
	}

	// Update static availability card to Offline with real DB stats
	if b.notifier != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		details := b.fetchStatusDetails(ctx)
		cancel()
		_ = b.notifier.UpdateStatusCard(false, details)
	}

	b.session.Close()
}
