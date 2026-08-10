package scraper

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/robfig/cron/v3"

	"JoblessYu/internal/bot"
	"JoblessYu/internal/job"
)

// Announcer interface decouples ScraperManager from Discord notification logic.
type Announcer interface {
	PostDailyScrapeAnnouncement(summary bot.DailyScrapeSummary) error
	UpdateStatusCard(online bool, details bot.StatusDetails) error
	PostDevLogWebhook(title string, description string, color int, fields map[string]string)
}

// ScraperManager orchestrates the daily hybrid scraping pipeline:
//   - Python jobspy (Indeed + LinkedIn) — subprocess, writes to DB directly
//   - Go Colly (ITViec) — native, returns jobs for Go to upsert
//   - Batch AI enrichment — runs after both scrapers complete
//
// Schedule: daily 5:00 AM ICT (scrape + enrich), weekly Mon 4:55 AM (cleanup).
type ScraperManager struct {
	cron *cron.Cron

	collyScraper *job.CollyScraper
	enricher     *job.BatchEnricher
	repo         *job.JobRepository
	announcer    Announcer

	pythonScript  string
	pythonExe     string
	retentionDays int

	ctx    context.Context
	cancel context.CancelFunc
	runMu  sync.Mutex
}

func NewScraperManager(
	repo *job.JobRepository,
	collyScraper *job.CollyScraper,
	enricher *job.BatchEnricher,
	retentionDays int,
) *ScraperManager {
	ctx, cancel := context.WithCancel(context.Background())
	m := &ScraperManager{
		cron:          cron.New(cron.WithLocation(time.UTC)),
		collyScraper:  collyScraper,
		enricher:      enricher,
		repo:          repo,
		retentionDays: retentionDays,
		ctx:           ctx,
		cancel:        cancel,
	}
	m.pythonScript = m.resolveScriptPath()
	m.pythonExe = m.resolvePythonExe()
	return m
}

func (m *ScraperManager) SetAnnouncer(a Announcer) {
	m.announcer = a
}

// resolveScriptPath locates scraper-python/JoblessYu.py by walking up
// from the current working directory until go.mod is found.
func (m *ScraperManager) resolveScriptPath() string {
	dir, err := os.Getwd()
	if err != nil {
		return filepath.Join("scraper-python", "JoblessYu.py")
	}
	for {
		if info, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil && !info.IsDir() {
			return filepath.Join(dir, "scraper-python", "JoblessYu.py")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return filepath.Join("scraper-python", "JoblessYu.py")
}

// resolvePythonExe picks the virtual environment Python interpreter if present,
// otherwise falls back to system Python on PATH.
func (m *ScraperManager) resolvePythonExe() string {
	scriptDir := filepath.Dir(m.pythonScript)
	for _, venvPath := range []string{
		filepath.Join(scriptDir, ".venv", "Scripts", "python.exe"),
		filepath.Join(scriptDir, ".venv", "Scripts", "python"),
		filepath.Join(scriptDir, "venv", "Scripts", "python.exe"),
		filepath.Join(scriptDir, "venv", "Scripts", "python"),
		filepath.Join(scriptDir, ".venv", "bin", "python3"),
		filepath.Join(scriptDir, ".venv", "bin", "python"),
		filepath.Join(scriptDir, "venv", "bin", "python3"),
		filepath.Join(scriptDir, "venv", "bin", "python"),
	} {
		if info, err := os.Stat(venvPath); err == nil && !info.IsDir() {
			return venvPath
		}
	}

	for _, name := range []string{"python3", "python", "py"} {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}
	slog.Warn("No python interpreter found on PATH; falling back to python3")
	return "python3"
}

func (m *ScraperManager) StartSchedule() {
	// Weekly cleanup: Monday 4:55 AM ICT = Sunday 21:55 UTC (ICT = UTC+7).
	_, err := m.cron.AddFunc("55 21 * * 0", func() {
		m.runCleanup(m.ctx)
	})
	if err != nil {
		slog.Warn("Failed to schedule cleanup cron", "err", err)
	}

	// Daily scrape: 5:00 AM ICT = 22:00 UTC previous day.
	_, err = m.cron.AddFunc("0 22 * * *", func() {
		m.runScrapeAndEnrich(m.ctx)
	})
	if err != nil {
		slog.Warn("Failed to schedule scrape cron", "err", err)
	}

	fmt.Println("+--------------------------------------------------------------------------------+")
	fmt.Println("| JOBLESSYU SCRAPER SCHEDULE ACTIVATED                                           |")
	fmt.Println("+--------------------------------------------------------------------------------+")
	fmt.Println("| Daily Scrape:   05:00 ICT (22:00 UTC) (30 ITViec + 30 Indeed + 30 LinkedIn)    |")
	fmt.Printf("| Weekly Cleanup: Mon 04:55 ICT (Sun 21:55 UTC) (Retention: %d days)             |\n", m.retentionDays)
	fmt.Println("+--------------------------------------------------------------------------------+")
	m.cron.Start()
}

func (m *ScraperManager) StopSchedule() {
	stopCtx := m.cron.Stop()
	m.cancel()
	<-stopCtx.Done()
	m.runMu.Lock()
	defer m.runMu.Unlock()
}

// RunScrapeAndEnrich runs the full daily pipeline (exported for manual use).
func (m *ScraperManager) RunScrapeAndEnrich(ctx context.Context) {
	m.runScrapeAndEnrich(ctx)
}

// runScrapeAndEnrich runs the full daily pipeline with formatted box-drawing logs:
// 1. Python jobspy (Indeed + LinkedIn) -> writes to DB
// 2. Go Colly (ITViec) -> returns jobs -> Go upserts
// 3. Upload to DB
// 4. Batch AI enrichment
func (m *ScraperManager) runScrapeAndEnrich(ctx context.Context) {
	m.runMu.Lock()
	defer m.runMu.Unlock()

	pipelineStart := time.Now()
	timestampStr := time.Now().UTC().Format("2006-01-02 15:04:05 UTC")

	fmt.Println()
	fmt.Println("+--------------------------------------------------------------------------------+")
	fmt.Println("| PIPELINE RUN: Daily Job Scrape & AI Enrichment                                 |")
	fmt.Printf("| Triggered: %-67s |\n", timestampStr)
	fmt.Println("+--------------------------------------------------------------------------------+")

	// Phase 1: jobspy (Indeed + LinkedIn)
	fmt.Println()
	fmt.Println("+--- [1/4] SCRAPING (Indeed & LinkedIn via JobSpy) -------------------------------+")
	fmt.Println("| Target: 30 Indeed + 30 LinkedIn jobs                                           |")
	jobspyStart := time.Now()
	jobspyCount, jobspyErr := m.runPythonScraper(ctx)
	jobspyDuration := time.Since(jobspyStart).Round(100 * time.Millisecond)
	jobspyStatus := "SUCCESS"
	if jobspyErr != nil {
		jobspyStatus = "FAILED"
		fmt.Printf("| Warning: JobSpy error: %v\n", jobspyErr)
	}
	fmt.Printf("| Status: %-7s | Completed in %-6s | Scraped: %d jobs                          |\n", jobspyStatus, jobspyDuration.String(), jobspyCount)
	fmt.Println("+--------------------------------------------------------------------------------+")

	// Phase 2: Colly (ITViec)
	fmt.Println()
	fmt.Println("+--- [2/4] SCRAPING (ITViec via Colly) ------------------------------------------+")
	fmt.Println("| Target: 30 ITViec jobs (Freshness <= 24h)                                      |")
	collyStart := time.Now()
	itviecJobs, collyErr := m.collyScraper.ScrapeITViec(ctx)
	collyDuration := time.Since(collyStart).Round(100 * time.Millisecond)
	collyStatus := "SUCCESS"
	if collyErr != nil {
		collyStatus = "FAILED"
		fmt.Printf("| Warning: Colly error: %v\n", collyErr)
	}
	fmt.Printf("| Status: %-7s | Completed in %-6s | Scraped: %d jobs                          |\n", collyStatus, collyDuration.String(), len(itviecJobs))
	fmt.Println("+--------------------------------------------------------------------------------+")

	// Phase 3: Upload to DB
	fmt.Println()
	fmt.Println("+--- [3/4] DATABASE UPSERT (Neon PostgreSQL) ------------------------------------+")
	dbStart := time.Now()
	stats, dbErr := m.repo.UpsertJobs(ctx, itviecJobs)
	dbDuration := time.Since(dbStart).Round(100 * time.Millisecond)
	dbStatus := "SUCCESS"
	if dbErr != nil {
		dbStatus = "FAILED"
		fmt.Printf("| Warning: Upload error: %v\n", dbErr)
	}

	unenrichedCount, enrichedCount, _ := m.repo.CountEnrichmentStats(ctx)
	fmt.Printf("| ITViec DB Upsert: %d new inserted, %d merged into alternate_urls              |\n", stats.Inserted, stats.Merged)
	fmt.Printf("| DB Enrichment Queue: %d pending AI enrichment, %d already enriched              |\n", unenrichedCount, enrichedCount)
	fmt.Printf("| Status: %-7s | Completed in %-6s                                            |\n", dbStatus, dbDuration.String())
	fmt.Println("+--------------------------------------------------------------------------------+")

	// Phase 4: AI Enrichment
	fmt.Println()
	fmt.Println("+--- [4/4] AI ENRICHMENT (Groq llama-3.1-8b-instant) ---------------------------+")
	fmt.Printf("| Queue: %d jobs pending AI enrichment (%d previously enriched jobs skipped)      |\n", unenrichedCount, enrichedCount)
	enrichCtx, enrichCancel := context.WithTimeout(ctx, 2*time.Hour)
	defer enrichCancel()

	enrichStart := time.Now()
	enrichErr := m.enricher.Run(enrichCtx)
	enrichDuration := time.Since(enrichStart).Round(100 * time.Millisecond)
	enrichStatus := "SUCCESS"
	if enrichErr != nil {
		enrichStatus = "FAILED"
		fmt.Printf("| Warning: Enrichment error: %v\n", enrichErr)
	}
	fmt.Printf("| Status: %-7s | Completed in %-6s                                            |\n", enrichStatus, enrichDuration.String())
	fmt.Println("+--------------------------------------------------------------------------------+")

	// Summary Table
	totalDuration := time.Since(pipelineStart).Round(100 * time.Millisecond)
	fmt.Println()
	fmt.Println("+--------------------------------------------------------------------------------+")
	fmt.Println("| JOBLESSYU PIPELINE SUMMARY REPORT                                              |")
	fmt.Println("+----------------------+---------+--------------------------------+--------------+")
	fmt.Println("| Step                 | Status  | Detail                         | Duration     |")
	fmt.Println("+----------------------+---------+--------------------------------+--------------+")
	fmt.Printf("| JobSpy (Py)          | %-7s | %-30s | %-12s |\n", jobspyStatus, fmt.Sprintf("%d jobs scraped", jobspyCount), jobspyDuration.String())
	fmt.Printf("| Colly (Go)           | %-7s | %-30s | %-12s |\n", collyStatus, fmt.Sprintf("%d jobs scraped", len(itviecJobs)), collyDuration.String())
	fmt.Printf("| Neon DB Upsert       | %-7s | %-30s | %-12s |\n", dbStatus, fmt.Sprintf("%d new, %d merged", stats.Inserted, stats.Merged), dbDuration.String())
	fmt.Printf("| Groq AI Enrichment   | %-7s | %-30s | %-12s |\n", enrichStatus, fmt.Sprintf("%d jobs in batch", unenrichedCount), enrichDuration.String())
	fmt.Println("+----------------------+---------+--------------------------------+--------------+")
	fmt.Printf("| TOTAL EXECUTION TIME: %-56s |\n", totalDuration.String())
	fmt.Println("+--------------------------------------------------------------------------------+")
	fmt.Println()

	// Trigger Discord Announcements & Dev Webhook Logs
	if m.announcer != nil {
		totalActive := unenrichedCount + enrichedCount
		summary := bot.DailyScrapeSummary{
			RunTime:         pipelineStart,
			TotalDuration:   totalDuration,
			JobspyCount:     jobspyCount,
			CollyCount:      len(itviecJobs),
			InsertedCount:   stats.Inserted,
			MergedCount:     stats.Merged,
			EnrichedCount:   unenrichedCount,
			TotalActiveJobs: totalActive,
		}
		if err := m.announcer.PostDailyScrapeAnnouncement(summary); err != nil {
			slog.Warn("Failed to post daily scrape announcement", "err", err)
		}

		statusDetails := bot.StatusDetails{
			ActiveJobs:     totalActive,
			LastScrapeTime: time.Now(),
			NextScrapeTime: bot.CalculateNextScrapeTime(time.Now()),
			RetentionDays:  m.retentionDays,
			Version:        "v1.2.0",
		}
		if err := m.announcer.UpdateStatusCard(true, statusDetails); err != nil {
			slog.Warn("Failed to update status card after pipeline completion", "err", err)
		}
	}
}

// pythonJobCountRe matches "40 jobs upserted to Neon." from jobspy output.
var pythonJobCountRe = regexp.MustCompile(`(\d+) jobs? upserted`)

// runPythonScraper runs the Python jobspy script as a subprocess.
// Streams stdout/stderr to os.Stdout/os.Stderr so logs are visible,
// while capturing output to return the number of jobs scraped.
// The script writes directly to Neon DB via psycopg.
func (m *ScraperManager) runPythonScraper(ctx context.Context) (int, error) {
	cmd := exec.CommandContext(ctx, m.pythonExe, m.pythonScript)
	var outBuf bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &outBuf)
	cmd.Stderr = io.MultiWriter(os.Stderr, &outBuf)

	err := cmd.Run()
	output := outBuf.String()

	// Parse job count from output (e.g. "40 jobs upserted to Neon.")
	count := 0
	if matches := pythonJobCountRe.FindStringSubmatch(output); len(matches) >= 2 {
		count, _ = strconv.Atoi(matches[1])
	}

	if err != nil {
		if ctx.Err() != nil {
			return count, fmt.Errorf("jobspy aborted: %w", ctx.Err())
		}
		return count, fmt.Errorf("jobspy error: %w (output: %s)", err, output)
	}

	return count, nil
}

// runCleanup deletes jobs older than the retention period.
func (m *ScraperManager) runCleanup(ctx context.Context) {
	m.runMu.Lock()
	defer m.runMu.Unlock()

	fmt.Println()
	fmt.Println("+--------------------------------------------------------------------------------+")
	fmt.Println("| CLEANUP RUN: Weekly Job Retention Maintenance                                  |")
	fmt.Printf("| Retention Window: %d Days                                                     |\n", m.retentionDays)
	deleted, err := m.repo.DeleteOldJobs(ctx, m.retentionDays)
	if err != nil {
		fmt.Printf("| Status: FAILED | Error: %v\n", err)
	} else {
		fmt.Printf("| Status: SUCCESS | Deleted: %d old jobs                                        |\n", deleted)
	}
	fmt.Println("+--------------------------------------------------------------------------------+")
	fmt.Println()
}
