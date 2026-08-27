package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	// Discord
	DiscordToken         string
	DiscordGuild         string
	DiscordChannelID     string
	DiscordLogWebhookURL string

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
		DiscordToken:         os.Getenv("DISCORD_BOT_TOKEN"),
		DatabaseURL:          os.Getenv("DATABASE_URL"),
		DiscordGuild:         os.Getenv("DISCORD_GUILD_ID"),
		DiscordChannelID:     os.Getenv("DISCORD_CHANNEL_ID"),
		DiscordLogWebhookURL: os.Getenv("DISCORD_LOG_WEBHOOK_URL"),

		GroqAPIKey: os.Getenv("GROQ_API_KEY"),
		AIModel:    os.Getenv("AI_MODEL"),
	}

	// Fallback check for legacy env variable names if DISCORD_CHANNEL_ID is not set
	if cfg.DiscordChannelID == "" {
		if v := os.Getenv("DISCORD_STATUS_CHANNEL_ID"); v != "" {
			cfg.DiscordChannelID = v
		} else if v := os.Getenv("DISCORD_ANNOUNCEMENT_CHANNEL_ID"); v != "" {
			cfg.DiscordChannelID = v
		}
	}

	if cfg.DiscordToken == "" {
		log.Fatal("config: DISCORD_BOT_TOKEN is not set")
	}
	if cfg.DatabaseURL == "" {
		log.Fatal("config: DATABASE_URL is not set")
	}
	if cfg.DiscordChannelID == "" {
		log.Fatal("config: DISCORD_CHANNEL_ID is not set in .env")
	}
	if cfg.DiscordGuild == "" {
		log.Println("config: DISCORD_GUILD_ID is not set; slash commands will be registered globally (note: Discord global commands take ~1 hour to propagate)")
	}

	// AI config defaults.
	if cfg.AIModel == "" {
		cfg.AIModel = "openai/gpt-oss-20b"
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
