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

	return &Config{
		DiscordToken: os.Getenv("DISCORD_BOT_TOKEN"),
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		DiscordGuild: os.Getenv("DISCORD_GUILD_ID"),
	}
}
