package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	// Discord
	DiscordToken string
	DiscordGuild string

	// Database
	DatabaseURL string

	// AI
	GroqAPIKey string
	AIModel    string

	// Job retention
	JobRetentionDays int
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("Note: .env file not found, using system env variables")
	}

	cfg := &Config{
		DiscordToken: os.Getenv("DISCORD_BOT_TOKEN"),
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		DiscordGuild: os.Getenv("DISCORD_GUILD_ID"),

		GroqAPIKey: os.Getenv("GROQ_API_KEY"),
		AIModel:    os.Getenv("AI_MODEL"),
	}

	if cfg.DiscordToken == "" {
		log.Fatal("config: DISCORD_BOT_TOKEN is not set")
	}
	if cfg.DatabaseURL == "" {
		log.Fatal("config: DATABASE_URL is not set")
	}
	if cfg.DiscordGuild == "" {
		log.Println("config: DISCORD_GUILD_ID is not set; slash commands will be registered globally")
	}

	// AI config defaults.
	if cfg.AIModel == "" {
		cfg.AIModel = "llama-3.1-8b-instant"
	}
	if cfg.GroqAPIKey == "" {
		log.Println("config: GROQ_API_KEY is not set; AI enrichment will be skipped")
	}

	cfg.JobRetentionDays = 30
	if v := os.Getenv("JOB_RETENTION_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.JobRetentionDays = n
		}
	}

	return cfg
}
