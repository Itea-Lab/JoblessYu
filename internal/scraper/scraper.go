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

// resolveScriptPath locates Python-Jobspy/JoblessYu.py by walking up from the
// current working directory until a go.mod is found, then joining the relative
// script path. Falls back to the relative path if no repo root is found.
func (m *ScraperManager) resolveScriptPath() string {
	dir, err := os.Getwd()
	if err != nil {
		return filepath.Join("Python-Jobspy", "JoblessYu.py")
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return filepath.Join(dir, "Python-Jobspy", "JoblessYu.py")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return filepath.Join("Python-Jobspy", "JoblessYu.py")
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
	_, err := m.cron.AddFunc("@every 6h", func() {
		m.runWithLock(m.ctx)
	})
	if err != nil {
		log.Println("Failed to schedule scraper:", err)
		return
	}

	log.Printf("Job scraper scheduled to run every 6 hours (using %s).", m.pythonExe)
	m.cron.Start()

	// Run once initially in background.
	go m.runWithLock(m.ctx)
}

// StopSchedule cancels any in-flight scrape and blocks until all scheduled and
// initial runs have fully drained.
func (m *ScraperManager) StopSchedule() {
	stopCtx := m.cron.Stop()
	m.cancel()
	<-stopCtx.Done()

	// Wait for any run launched outside the cron (the initial goroutine).
	m.runMu.Lock()
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
