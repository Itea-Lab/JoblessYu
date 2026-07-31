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

	"JoblessYu/internal/job"
)

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

	slog.Info("Scraper scheduled: daily 05:00 ICT (jobspy + Colly + enrich), weekly Mon 04:55 ICT (cleanup)", "retention_days", m.retentionDays)
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

// runScrapeAndEnrich runs the full daily pipeline with clean phased output:
// 1. Python jobspy (Indeed + LinkedIn) → writes to DB
// 2. Go Colly (ITViec) → returns jobs → Go upserts
// 3. Upload to DB
// 4. Batch AI enrichment
func (m *ScraperManager) runScrapeAndEnrich(ctx context.Context) {
	m.runMu.Lock()
	defer m.runMu.Unlock()

	// Phase 1: jobspy (Indeed + LinkedIn)
	slog.Info("[1/4] Scraping Indeed + LinkedIn (jobspy)...")
	jobspyStart := time.Now()
	jobspyCount, err := m.runPythonScraper(ctx)
	if err != nil {
		slog.Warn("jobspy error", "err", err)
	}
	slog.Info("jobspy done", "jobs", jobspyCount, "elapsed", time.Since(jobspyStart).Round(time.Second))

	// Phase 2: Colly (ITViec)
	slog.Info("[2/4] Scraping ITViec (Colly)...")
	collyStart := time.Now()
	itviecJobs, err := m.collyScraper.ScrapeITViec(ctx)
	if err != nil {
		slog.Warn("Colly error", "err", err)
	}
	slog.Info("Colly done", "jobs", len(itviecJobs), "elapsed", time.Since(collyStart).Round(time.Second))

	// Phase 3: Upload to DB
	slog.Info("[3/4] Uploading to Neon DB...")
	uploaded, err := m.repo.UpsertJobs(ctx, itviecJobs)
	if err != nil {
		slog.Warn("Upload error", "err", err)
	}
	slog.Info("Upload done", "upserted", uploaded)

	// Phase 4: AI Enrichment
	slog.Info("[4/4] AI Enrichment (Groq)...")
	enrichCtx, enrichCancel := context.WithTimeout(ctx, 2*time.Hour)
	defer enrichCancel()

	enrichStart := time.Now()
	if err := m.enricher.Run(enrichCtx); err != nil {
		slog.Warn("Enrichment error", "err", err)
	}
	slog.Info("Enrichment done", "elapsed", time.Since(enrichStart).Round(time.Second))
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

	slog.Info("Starting cleanup", "retention_days", m.retentionDays)
	deleted, err := m.repo.DeleteOldJobs(ctx, m.retentionDays)
	if err != nil {
		slog.Warn("Cleanup error", "err", err)
		return
	}
	slog.Info("Cleanup done", "deleted", deleted)
}
