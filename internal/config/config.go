package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DiscordToken string
	DatabaseURL  string
	DiscordGuild string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("Note: .env file not found, using system env variables")
	}

	cfg := &Config{
		DiscordToken: os.Getenv("DISCORD_BOT_TOKEN"),
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		DiscordGuild: os.Getenv("DISCORD_GUILD_ID"),
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

	return cfg
}
