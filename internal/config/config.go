package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	// Discord
	DiscordToken string
	DiscordGuild string

	// Database
	DatabaseURL string

	// AI (Slice D wires these; Slice C just loads them)
	AIProvider string // "groq" (default), future: "google"
	GroqAPIKey string
	AIModel    string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("Note: .env file not found, using system env variables")
	}

	cfg := &Config{
		DiscordToken: os.Getenv("DISCORD_BOT_TOKEN"),
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		DiscordGuild: os.Getenv("DISCORD_GUILD_ID"),

		AIProvider: os.Getenv("AI_PROVIDER"),
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

	// AI config defaults. Slice D uses these; Slice C just validates.
	if cfg.AIProvider == "" {
		cfg.AIProvider = "groq"
	}
	if cfg.AIModel == "" {
		cfg.AIModel = "qwen/qwen3.6-27b"
	}
	if cfg.GroqAPIKey == "" {
		log.Println("config: GROQ_API_KEY is not set; AI extractor will fall back to regex (Slice D)")
	}

	return cfg
}
