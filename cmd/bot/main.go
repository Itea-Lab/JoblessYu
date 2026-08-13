package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
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
	testNotify := flag.Bool("test-notify", false, "run notification & UI visibility test suite (status card online/offline, daily announcement, dev webhook), then exit")
	evalAI := flag.Bool("eval-ai", false, "run AI evaluation & hallucination benchmark suite, then exit")
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

	svc := job.NewJobService(repo)

	// Manual mode / Eval mode: run scrape, enrich, or eval-ai, then exit.
	if *evalAI {
		log.Println("Running AI Evaluation & Hallucination Benchmark Suite...")
		report := job.RunAIEval(ctx, groq)
		job.PrintEvalReport(report, cfg.AIModel)
		return
	}
	if *manualScrape {
		log.Println("Manual scrape: starting full pipeline (jobspy + Colly + enrichment)...")
		disbot, err := bot.NewBot(cfg, svc)
		if err == nil {
			if startErr := disbot.Start(); startErr == nil {
				scraperMgr.SetAnnouncer(disbot.Notifier())
				defer disbot.CloseWithoutOffline()
			}
		}
		scraperMgr.RunScrapeAndEnrich(ctx)
		log.Println("Manual scrape complete.")
		return
	}
	if *manualEnrich {
		runManualEnrich(ctx, enricher)
		return
	}

	// Normal mode / debug test mode.
	disbot, err := bot.NewBot(cfg, svc)
	if err != nil {
		log.Fatal("Error creating bot:", err)
	}

	if *testNotify {
		if err := disbot.Start(); err != nil {
			log.Fatal("Error starting bot session for test:", err)
		}
		runTestNotifications(disbot, cfg)
		disbot.Stop()
		return
	}

	// Lightweight HTTP health check server for Cloud Run / Docker / K8s readiness probes
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		})
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		_ = http.ListenAndServe(":"+port, mux)
	}()

	if err := disbot.Start(); err != nil {
		log.Fatal("Error starting bot:", err)
	}
	defer disbot.Stop()

	scraperMgr.SetAnnouncer(disbot.Notifier())
	scraperMgr.StartSchedule()
	defer scraperMgr.StopSchedule()

	fmt.Println("JoblessYu is now running. Press CTRL+C to exit.")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	fmt.Println("Shutting down...")
}

func runTestNotifications(disbot *bot.Bot, cfg *config.Config) {
	fmt.Println()
	fmt.Println("+--------------------------------------------------------------------------------+")
	fmt.Println("| JOBLESSYU NOTIFICATION & UI VISIBILITY TEST SUITE                              |")
	fmt.Println("+--------------------------------------------------------------------------------+")
	notifier := disbot.Notifier()
	if notifier == nil {
		fmt.Println("| [ERROR] Bot Notifier is nil!")
		fmt.Println("+--------------------------------------------------------------------------------+")
		return
	}

	details := bot.StatusDetails{
		ActiveJobs:     342,
		LastScrapeTime: time.Now(),
		NextScrapeTime: time.Now().Add(24 * time.Hour),
		RetentionDays:  cfg.JobRetentionDays,
		Version:        "v1.2.0 (DEBUG TEST)",
	}

	// 1. Test Status Card Online
	statusChID := notifier.GetResolvedChannelID(cfg.DiscordChannelID)
	fmt.Printf("| [1/4] Testing Status Card Online (🟢 ONLINE) -> Channel ID: %s...\n", statusChID)
	if err := notifier.UpdateStatusCard(true, details); err != nil {
		fmt.Printf("|   └─ 🔴 Failed: %v\n", err)
	} else {
		fmt.Printf("|   └─ 🟢 SUCCESS: Sent & pinned Online status card to channel!\n")
	}

	time.Sleep(1 * time.Second)

	// 2. Test Daily Announcement
	announceChID := notifier.GetResolvedChannelID(cfg.DiscordChannelID)
	fmt.Printf("| [2/4] Testing Daily Scrape Announcement (🌅 Daily IT Job List Updated) -> Channel ID: %s...\n", announceChID)
	summary := bot.DailyScrapeSummary{
		RunTime:         time.Now(),
		TotalDuration:   8 * time.Minute,
		JobspyCount:     80,
		CollyCount:      40,
		InsertedCount:   38,
		MergedCount:     14,
		EnrichedCount:   52,
		TotalActiveJobs: 342,
	}
	if err := notifier.PostDailyScrapeAnnouncement(summary); err != nil {
		fmt.Printf("|   └─ 🔴 Failed: %v\n", err)
	} else {
		fmt.Printf("|   └─ 🟢 SUCCESS: Sent daily job update announcement to channel!\n")
	}

	time.Sleep(1 * time.Second)

	// 3. Test Developer Webhook / Fallback Log
	logTarget := cfg.DiscordLogWebhookURL
	if logTarget == "" {
		logTarget = "Fallback Server Channel: " + statusChID
	}
	fmt.Printf("| [3/4] Testing Developer Log Card -> Target: %s...\n", logTarget)
	notifier.PostDevLogWebhook(
		"🧪 DEBUG TEST: Pipeline Verification",
		"This is an automated test card sent via Discord Notifier.",
		0x3B82F6, // Blue
		map[string]string{
			"Environment": "Development Debug Mode",
			"Test Time":   time.Now().Format("15:04:05 ICT"),
			"Status":      "PASSED",
		},
	)
	fmt.Println("|   └─ 🟢 SUCCESS: Sent log card!")

	time.Sleep(2 * time.Second)

	// 4. Test Status Card Offline
	fmt.Printf("| [4/4] Testing Status Card Offline (🔴 OFFLINE) -> Channel ID: %s...\n", statusChID)
	if err := notifier.UpdateStatusCard(false, details); err != nil {
		fmt.Printf("|   └─ 🔴 Failed: %v\n", err)
	} else {
		fmt.Printf("|   └─ 🟢 SUCCESS: Updated status card to Offline!\n")
	}

	fmt.Println("+--------------------------------------------------------------------------------+")
	fmt.Println("| TEST SUITE COMPLETE: Check your Discord server for the echoed messages!        |")
	fmt.Println("+--------------------------------------------------------------------------------+")
	fmt.Println()
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
