.PHONY: dev bot test test-notify eval-ai lint lint-go lint-python scrape enrich migrate clean

bot:
	@go run ./cmd/bot || [ $$? -eq 1 ]  # suppress cosmetic exit code 1 from Ctrl+C

dev: bot

test:
	go test -race -cover ./...

# Test Discord notifications & UI visibility (Status Card Online/Offline, Daily Announcement, Dev Webhook)
test-notify:
	go run ./cmd/bot -test-notify

# Run AI Evaluation & Hallucination Benchmark Suite
eval-ai:
	go run ./cmd/bot -eval-ai

lint: lint-go lint-python

lint-go:
	go vet ./...

lint-python:
	ruff check scraper-python/ || true

# Full pipeline: jobspy (Indeed+LinkedIn) + Colly (ITViec) + AI enrichment
scrape:
	go run ./cmd/bot -scrape

# Enrichment only (no scraping)
enrich:
	go run ./cmd/bot -enrich

# DB migrations
migrate:
	@eval $$(scraper-python/.venv/bin/python3 -c "from dotenv import dotenv_values; [print(f'export {k}={v!r}') for k,v in dotenv_values('.env').items()]" 2>/dev/null || python3 -c "from dotenv import dotenv_values; [print(f'export {k}={v!r}') for k,v in dotenv_values('.env').items()]") && for f in migrations/*.sql; do echo "Applying $$f"; psql "$$DATABASE_URL" -f $$f; done

clean:
	rm -f jobs.json bot
