.PHONY: dev bot test lint lint-go lint-python scrape migrate clean

# Export .env vars into the shell environment. Uses Python's dotenv
# (already a project dependency) to parse correctly — handles values
# with &, =, spaces, and quotes the same way Go's godotenv does.
# Usage in targets: $$(eval $$($(ENVLOAD))) then use $$VAR_NAME.
ENVLOAD := python3 -c "from dotenv import dotenv_values; [print(f'export {k}={v!r}') for k,v in dotenv_values('.env').items()]"

bot:
	@go run ./cmd/bot || [ $$? -eq 1 ]  # suppress cosmetic exit code 1 from Ctrl+C

dev: bot

test:
	go test -race -cover ./...

lint: lint-go lint-python

lint-go:
	go vet ./...

lint-python:
	ruff check scraper-python/ || true  # don't fail CI if ruff isn't installed locally

scrape:
	cd scraper-python && source .venv/bin/activate && python JoblessYu.py

migrate:
	@eval $$($(ENVLOAD)) && for f in migrations/*.sql; do echo "Applying $$f"; psql "$$DATABASE_URL" -f $$f; done

clean:
	rm -f jobs.json
