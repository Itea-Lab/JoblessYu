package scraper

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/robfig/cron/v3"
)

type ScraperManager struct {
	cron *cron.Cron
}

func NewScraperManager() *ScraperManager {
	return &ScraperManager{
		cron: cron.New(),
	}
}

func (m *ScraperManager) StartSchedule() {
	// Schedule to run every 6 hours
	_, err := m.cron.AddFunc("@every 6h", m.RunScraper)
	if err != nil {
		log.Println("Failed to schedule scraper:", err)
		return
	}

	log.Println("Job scraper scheduled to run every 6 hours.")
	m.cron.Start()

	// Run once initially in background
	go m.RunScraper()
}

func (m *ScraperManager) StopSchedule() {
	m.cron.Stop()
}

func (m *ScraperManager) RunScraper() {
	log.Println("Starting Python job scraper...")
	
	scriptPath := filepath.Join("Python-Jobspy", "JoblessYu.py")
	cmd := exec.Command("python", scriptPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	err := cmd.Run()
	if err != nil {
		log.Printf("Python scraper encountered an error: %v", err)
		return
	}

	log.Printf("Python scraper finished successfully.")
}
