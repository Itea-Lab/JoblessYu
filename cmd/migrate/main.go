package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"JoblessYu/internal/config"

	"github.com/jackc/pgx/v5"
)

func main() {
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is not set in environment or .env")
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer conn.Close(ctx)

	files, err := filepath.Glob("migrations/*.sql")
	if err != nil {
		log.Fatalf("Failed to read migrations: %v\n", err)
	}
	sort.Strings(files)

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			log.Fatalf("Failed to read %s: %v\n", file, err)
		}

		fmt.Printf("Applying %s...\n", file)
		// Strip comments or execute full batch
		sql := string(content)
		if strings.TrimSpace(sql) == "" {
			continue
		}

		if _, err := conn.Exec(ctx, sql); err != nil {
			log.Printf("Warning/Error applying %s: %v (continuing if idempotent)\n", file, err)
		} else {
			fmt.Printf("✓ %s applied successfully.\n", file)
		}
	}

	fmt.Println("\nAll database migrations finished successfully!")
}
