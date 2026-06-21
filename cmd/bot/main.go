package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"JoblessYu/internal/bot"
	"JoblessYu/internal/config"
	"JoblessYu/internal/repository"
	"JoblessYu/internal/scraper"
	"JoblessYu/internal/service"
)

func main() {
	cfg := config.Load()

	repo := repository.NewJobRepository(cfg.DatabaseURL)
	svc := service.NewJobService(repo)

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

	fmt.Println("JoblessYu Vessel is now running. Press CTRL+C to exit.")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	fmt.Println("Shutting down...")
}
