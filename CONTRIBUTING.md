# Contributing to JoblessYu

Thanks for contributing! This guide covers setup, architecture, and conventions for our 3-person team.

## Quick start

```bash
# 1. Clone
git clone https://github.com/Itea-Lab/JoblessYu.git
cd JoblessYu

# 2. Go deps
go mod download

# 3. Python scraper deps
python3 -m venv scraper-python/.venv
source scraper-python/.venv/bin/activate
pip install -r scraper-python/requirements.txt

# 4. Copy .env.example → .env, fill in credentials
cp .env.example .env

# 5. Apply DB migrations
make migrate

# 6. Run the bot
make bot
```

## Required credentials

| Service | Where to get it | Env var |
|---|---|---|
| Discord bot token | https://discord.com/developers/applications | `DISCORD_BOT_TOKEN` |
| Discord guild ID | Server settings → Developer Mode → copy ID | `DISCORD_GUILD_ID` |
| Neon Postgres URL | https://neon.tech (free tier) | `DATABASE_URL` |
| Groq API key | https://console.groq.com (free tier) | `GROQ_API_KEY` |

## Architecture

```
┌──────────────────────────────────────────────────────────┐
│                    DAILY 5:00 AM ICT                      │
│                                                          │
│  ┌─────────────────┐  ┌──────────────────┐             │
│  │ Python jobspy   │  │ Go Colly          │             │
│  │ Indeed+LinkedIn │  │ ITViec            │             │
│  │ 40 jobs → DB    │  │ 20 jobs → DB      │             │
│  └─────────────────┘  └──────────────────┘             │
│           └──────────┬──────────────┘                   │
│                      ▼                                  │
│         ┌────────────────────────┐                     │
│         │ BatchEnricher (Go)      │                     │
│         │ Groq AI classification  │                     │
│         │ 60 jobs × 18s = ~18 min │                     │
│         └────────────────────────┘                     │
│                      ▼                                  │
│         Neon Postgres (ai_processed_at = NOW())        │
└──────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────┐
│                  USER types /jobs                        │
│                                                          │
│  Discord Bot → service.FetchAndProcessJobs()             │
│              → store.FetchRawJobs()                      │
│              → SELECT WHERE ai_processed_at IS NOT NULL   │
│              → instant response (no AI calls)            │
└──────────────────────────────────────────────────────────┘
```

### Package layout

```
cmd/bot/main.go              # Entry point — wires everything
internal/
├── bot/                     # Discord adapter
│   ├── bot.go               # Bot struct, session, slash command registration
│   ├── handlers.go           # /jobs command, pagination, filter panel
│   └── embeds.go            # Discord embed builder
├── config/config.go         # .env loading
├── job/                     # All job logic (package-by-feature)
│   ├── entry.go             # JobEntry, JobMeta, JobQuery structs
│   ├── expertise.go         # 13 expertise categories + keyword detection
│   ├── enricher.go          # BatchEnricher — daily AI enrichment with retry
│   ├── groq.go              # GroqExtractor — AI classification via Groq API
│   ├── regex.go             # RegexExtractor — fallback when Groq is down
│   ├── colly_scraper.go     # ITViec scraper (Go-native, Colly framework)
│   ├── store.go             # Neon Postgres repository (CRUD + upsert)
│   ├── service.go           # JobService — pure DB read (no AI calls)
│   ├── skills.md            # AI system prompt (bilingual EN+VI)
│   ├── schema.go            # JSON schema description for Groq
│   ├── keywords.go          # Tag scanner (8 categories)
│   └── levels.go            # Level rules (Intern/Fresher/Junior/Senior)
├── scraper/scraper.go       # Cron scheduler — jobspy + Colly + enrich + cleanup
scraper-python/
├── JoblessYu.py             # jobspy scraper (Indeed + LinkedIn)
└── requirements.txt         # Python deps
migrations/
├── 001_init.sql             # Base jobs table
├── 002_add_ai_columns.sql   # AI enrichment columns
└── 003_add_expertise.sql    # Expertise column + index
```

## Development commands

```bash
make bot       # Start bot + cron
make scrape    # Full pipeline: jobspy + Colly + enrichment
make enrich    # Enrichment only (process pending jobs)
make test      # Go tests with race detector + coverage
make lint      # go vet + ruff check
make migrate   # Apply SQL migrations
make clean     # Remove build artifacts
```

## Adding a database migration

1. Create `migrations/004_your_change.sql`
2. Use `IF NOT EXISTS` / `ADD COLUMN IF NOT EXISTS` (idempotent)
3. Run `make migrate` to apply
4. Commit the `.sql` file

Example:
```sql
-- 004_add_salary_range.sql
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS salary_min INTEGER;
CREATE INDEX IF NOT EXISTS idx_jobs_salary_min ON jobs(salary_min);
```

## Code style

- **Go formatting**: Run `gofmt -w .` before committing
- **Logging**: Use `log/slog` (structured logging) — not `log.Printf`
- **Comments**: Explain *why*, not *what*. Don't add comments unless the logic is non-obvious
- **Error handling**: Always wrap errors with context (`fmt.Errorf("scrape: %w", err)`)
- **Tests**: Run `make test` before pushing. Add tests for new logic in `internal/job/`

## Git workflow

1. Create a branch: `git checkout -b feature/your-feature`
2. Commit with clear messages:
   ```
   feat: add expertise filter to Discord UI
   fix: groq 400 error on long JDs
   refactor: consolidate Python runner into ScraperManager
   ```
3. Push: `git push origin feature/your-feature`
4. Create a PR on GitHub
5. Wait for CI (Go vet + test + staticcheck, Python lint)

## Key design decisions

- **Python stays for scraping**: jobspy handles LinkedIn/LinkedIn anti-bot — no Go alternative exists for free
- **AI enrichment is batch, not lazy**: jobs are enriched at 5AM, not on-demand — `/jobs` is always instant
- **Regex is last resort**: only triggers when Groq is unreachable (network errors), not for 429/400
- **Remote badge from HTML**: ITViec's working model badge is more reliable than AI guessing from JD text
- **Bilingual AI prompt**: `skills.md` handles Vietnamese titles, experience phrases, and skill-based inference
