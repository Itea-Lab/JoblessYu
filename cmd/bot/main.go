package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"JoblessYu/internal/bot"
	"JoblessYu/internal/config"
	"JoblessYu/internal/job"
	"JoblessYu/internal/scraper"
)

func main() {
	cfg := config.Load()

	ctx := context.Background()

	repo, err := job.NewJobRepository(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Error creating job repository:", err)
	}
	defer repo.Close()

	// Wire the extraction chain: Groq primary (accurate, may fail/rate-limit),
	// Regex fallback (always succeeds, less accurate). When GROQ_API_KEY is
	// not set, the chain degrades gracefully to regex-only.
	regex := job.NewRegexExtractor()
	var extractor job.Extractor = regex
	if cfg.GroqAPIKey != "" {
		groq := job.NewGroqExtractor(cfg.GroqAPIKey, cfg.AIModel)
		extractor = job.NewChainExtractor(groq, regex)
		log.Printf("AI extractor: Groq (model=%s) with regex fallback", cfg.AIModel)
	} else {
		log.Printf("AI extractor: regex only (GROQ_API_KEY not set)")
	}
	// Stop the Groq ticker on shutdown (no-op for regex-only).
	if closer, ok := extractor.(interface{ Close() }); ok {
		defer closer.Close()
	}

	svc := job.NewJobService(repo, extractor)

	disbot, err := bot.NewBot(cfg, svc)
	if err != nil {
		log.Fatal("Error creating bot:", err)
	}

	if err := disbot.Start(); err != nil {
		log.Fatal("Error starting bot:", err)
	}
	defer disbot.Stop()

	scraperMgr := scraper.NewScraperManager()
	scraperMgr.StartSchedule()
	defer scraperMgr.StopSchedule()

	fmt.Println("JoblessYu is now running. Press CTRL+C to exit.")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	fmt.Println("Shutting down...")
}
