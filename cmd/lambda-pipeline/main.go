package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"JoblessYu/internal/bot"
	"JoblessYu/internal/config"
	"JoblessYu/internal/job"
	"JoblessYu/internal/scraper"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/bwmarrin/discordgo"
)

type Event struct {
	Action string `json:"action"`
}

type Response struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func handler(ctx context.Context, rawEvent json.RawMessage) (Response, error) {
	var evt Event
	if len(rawEvent) > 0 {
		_ = json.Unmarshal(rawEvent, &evt)
	}
	if evt.Action == "" {
		evt.Action = "scrape_and_enrich"
	}

	cfg := config.Load()

	repo, err := job.NewJobRepository(ctx, cfg.DatabaseURL)
	if err != nil {
		return Response{Status: "error", Message: fmt.Sprintf("failed to init repository: %v", err)}, err
	}
	defer repo.Close()

	// 1. Handle Weekly Retention Cleanup Action
	if evt.Action == "cleanup" {
		log.Printf("Executing retention cleanup: purging jobs older than %d days...", cfg.JobRetentionDays)
		deleted, err := repo.DeleteOldJobs(ctx, cfg.JobRetentionDays)
		if err != nil {
			log.Printf("Error during retention cleanup: %v", err)
			return Response{Status: "error", Message: err.Error()}, err
		}
		msg := fmt.Sprintf("Retention cleanup complete: purged %d stale jobs", deleted)
		log.Println(msg)
		return Response{Status: "success", Message: msg}, nil
	}

	// 2. Handle Daily Scraping & AI Enrichment Action
	log.Println("Starting Serverless Scraping & AI Enrichment Pipeline...")

	session, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		log.Printf("Warning: failed to create discord session: %v", err)
	}

	notifier := bot.NewNotifier(session, cfg)
	groq := job.NewGroqExtractor(cfg.GroqAPIKey, cfg.AIModel)
	defer groq.Close()

	collyScraper := job.NewCollyScraper()
	enricher := job.NewBatchEnricher(repo, groq)
	scraperMgr := scraper.NewScraperManager(repo, collyScraper, enricher, cfg.JobRetentionDays)

	if notifier != nil {
		scraperMgr.SetAnnouncer(notifier)
	}

	scraperMgr.RunScrapeAndEnrich(ctx)

	log.Println("Serverless Scraping & AI Enrichment Pipeline finished successfully.")
	return Response{
		Status:  "success",
		Message: "Pipeline executed successfully",
	}, nil
}

func main() {
	lambda.Start(handler)
}
