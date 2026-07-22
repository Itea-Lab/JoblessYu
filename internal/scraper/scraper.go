package scraper

import (
	"context"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

type ScraperManager struct {
	cron       *cron.Cron
	scriptPath string
	pythonExe  string

	ctx    context.Context
	cancel context.CancelFunc

	runMu sync.Mutex
}

func NewScraperManager() *ScraperManager {
	ctx, cancel := context.WithCancel(context.Background())
	m := &ScraperManager{
		cron:   cron.New(),
		ctx:    ctx,
		cancel: cancel,
	}
	m.scriptPath = m.resolveScriptPath()
	m.pythonExe = m.resolvePythonExe()
	return m
}

// resolveScriptPath locates scraper-python/JoblessYu.py by walking up from the
// current working directory until a go.mod is found, then joining the relative
// script path. Falls back to the relative path if no repo root is found.
func (m *ScraperManager) resolveScriptPath() string {
	dir, err := os.Getwd()
	if err != nil {
		return filepath.Join("scraper-python", "JoblessYu.py")
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
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

// resolvePythonExe picks the first available Python interpreter on PATH.
func (m *ScraperManager) resolvePythonExe() string {
	for _, name := range []string{"python", "python3", "py"} {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}
	log.Printf("No python interpreter found on PATH; falling back to %q", "python")
	return "python"
}

func (m *ScraperManager) StartSchedule() {
	// Weekly: Monday 09:00 ICT = 02:00 UTC (ICT = UTC+7).
	_, err := m.cron.AddFunc("0 2 * * 1", func() {
		m.runWithLock(m.ctx)
	})
	if err != nil {
		log.Println("Failed to schedule scraper:", err)
		return
	}

	log.Printf("Job scraper scheduled weekly (Mon 09:00 ICT) using %s.", m.pythonExe)
	m.cron.Start()

	// No startup scrape — the bot restarts should not trigger an unintended
	// scrape. The weekly cron is the only automatic trigger. To run a
	// manual scrape, use: make scrape
}

// StopSchedule cancels any in-flight scrape and blocks until all scheduled
// runs have fully drained.
func (m *ScraperManager) StopSchedule() {
	stopCtx := m.cron.Stop()
	m.cancel()
	<-stopCtx.Done()

	// Wait for any in-flight run to finish by acquiring the lock.
	// The empty critical section is intentional — we just need to block
	// until runMu is released by an in-flight runWithLock.
	m.runMu.Lock()
	//lint:ignore SA2001 intentional empty critical section — waits for in-flight run
	m.runMu.Unlock()
}

func (m *ScraperManager) runWithLock(ctx context.Context) {
	m.runMu.Lock()
	defer m.runMu.Unlock()
	m.runScrape(ctx)
}

func (m *ScraperManager) runScrape(ctx context.Context) {
	if _, err := os.Stat(m.scriptPath); err != nil {
		log.Printf("Python scraper script not found at %q: %v", m.scriptPath, err)
		return
	}

	log.Println("Starting Python job scraper...")

	cmd := exec.CommandContext(ctx, m.pythonExe, m.scriptPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	start := time.Now()
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			log.Printf("Python scraper aborted: %v", ctx.Err())
			return
		}
		log.Printf("Python scraper encountered an error: %v", err)
		return
	}

	log.Printf("Python scraper finished successfully in %s.", time.Since(start).Round(time.Second))
}
