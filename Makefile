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

# Build AWS Lambda ARM64 binaries and zip packages
build-lambda:
	powershell -ExecutionPolicy Bypass -Command "if (!(Test-Path bin)) { New-Item -ItemType Directory -Path bin }; \$$env:GOOS='linux'; \$$env:GOARCH='arm64'; \$$env:CGO_ENABLED='0'; go build -tags lambda.norpc -ldflags='-s -w' -o bin/bootstrap ./cmd/lambda-bot; Compress-Archive -Path bin/bootstrap -DestinationPath bin/bot.zip -Force; Remove-Item bin/bootstrap; go build -tags lambda.norpc -ldflags='-s -w' -o bin/bootstrap ./cmd/lambda-pipeline; Compress-Archive -Path bin/bootstrap -DestinationPath bin/pipeline.zip -Force; Remove-Item bin/bootstrap; Write-Host 'Lambda packages built in bin/'"


# Terraform commands
tf-init:
	cd terraform/environments/prod && terraform init

tf-plan:
	cd terraform/environments/prod && terraform plan

tf-apply:
	cd terraform/environments/prod && terraform apply

clean:
	rm -f jobs.json bot bin/bot.zip bin/pipeline.zip

