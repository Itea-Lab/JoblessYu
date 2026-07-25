package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"JoblessYu/internal/bot"
	"JoblessYu/internal/config"
	"JoblessYu/internal/job"
	"JoblessYu/internal/scraper"
)

func main() {
	manualScrape := flag.Bool("scrape", false, "run a manual scrape (jobspy + Colly + enrichment), then exit")
	manualEnrich := flag.Bool("enrich", false, "run a manual enrichment batch, then exit")
	flag.Parse()

	cfg := config.Load()

	ctx := context.Background()

	repo, err := job.NewJobRepository(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Error creating job repository:", err)
	}
	defer repo.Close()

	groq := job.NewGroqExtractor(cfg.GroqAPIKey, cfg.AIModel)
	defer groq.Close()

	collyScraper := job.NewCollyScraper()
	enricher := job.NewBatchEnricher(repo, groq)
	scraperMgr := scraper.NewScraperManager(repo, collyScraper, enricher, cfg.JobRetentionDays)

	// Manual mode: run scrape or enrich, then exit (no bot, no cron).
	if *manualScrape {
		log.Println("Manual scrape: starting full pipeline (jobspy + Colly + enrichment)...")
		scraperMgr.RunScrapeAndEnrich(ctx)
		log.Println("Manual scrape complete.")
		return
	}
	if *manualEnrich {
		runManualEnrich(ctx, enricher)
		return
	}

	// Normal mode: start bot + cron.
	svc := job.NewJobService(repo)

	disbot, err := bot.NewBot(cfg, svc)
	if err != nil {
		log.Fatal("Error creating bot:", err)
	}

	if err := disbot.Start(); err != nil {
		log.Fatal("Error starting bot:", err)
	}
	defer disbot.Stop()

	scraperMgr.StartSchedule()
	defer scraperMgr.StopSchedule()

	fmt.Println("JoblessYu is now running. Press CTRL+C to exit.")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	fmt.Println("Shutting down...")
}

func runManualEnrich(ctx context.Context, enricher *job.BatchEnricher) {
	log.Println("Manual enrichment: starting...")
	enrichCtx, cancel := context.WithTimeout(ctx, 2*time.Hour)
	defer cancel()
	if err := enricher.Run(enrichCtx); err != nil {
		log.Fatalf("Enrichment failed: %v", err)
	}
	log.Println("Manual enrichment complete.")
}
