# JoblessYu-base-JoblessYu — AI Context Map

> **Stack:** raw-http | none | unknown | go

> 0 routes | 1 models | 0 components | 20 lib files | 13 env vars | 0 middleware | 100% test coverage
> **Token savings:** this file is ~2,000 tokens. Without it, AI exploration would cost ~15,500 tokens. **Saves ~13,500 tokens per conversation.**
> **Last scanned:** 2026-08-23 05:00 — re-run after significant changes

---

# Schema

### jobs
- id: bigint(auto) (pk)
- job_id: text (fk)
- site: text (required)
- job_url: text (required)
- title: text
- company: text
- location: text
- job_type: text
- description: text
- fetched_at: timestamp(tz) (required)

---

# Libraries

- `cmd\lambda-pipeline\main.go` — class Event, class Response
- `internal\bot\bot.go` — function NewBot: (cfg *config.Config, jobService jobService) (*Bot, error), class Bot
- `internal\bot\embeds.go` — function BuildStatusCardEmbed: (online bool, details StatusDetails) *discordgo.MessageEmbed, function BuildDailyAnnouncementEmbed: (summary DailyScrapeSummary) *discordgo.MessageEmbed
- `internal\bot\notifier.go`
  - function CalculateNextScrapeTime: (now time.Time) time.Time
  - function NewNotifier: (session *discordgo.Session, cfg *config.Config) *Notifier
  - class StatusDetails
  - class DailyScrapeSummary
  - class Notifier
- `internal\config\config.go` — function Load: () *Config, class Config
- `internal\job\colly_scraper.go` — function NewCollyScraper: () *CollyScraper, class CollyScraper
- `internal\job\enricher.go` — function NewBatchEnricher: (store enricherStore, extractor Extractor) *BatchEnricher, class BatchEnricher
- `internal\job\entry.go`
  - class JobSource
  - class JobEntry
  - class JobMeta
  - class JobQuery
- `internal\job\eval.go`
  - function GetBenchmarkDataset: () []EvalBenchmark
  - function RunAIEval: (ctx context.Context, extractor Extractor) EvalReport
  - function PrintEvalReport: (report EvalReport, modelName string)
  - class EvalBenchmark
  - class EvalReport
  - class EvalDetail
- `internal\job\expertise.go`
  - function ExpertiseLabel: (value string) string
  - function IsValidExpertise: (value string) bool
  - function DetectExpertise: (title, description string) string
  - class ExpertiseCategory
- `internal\job\extractor.go`
  - class JobBatchItem
  - interface Extractor
  - interface BatchExtractor
- `internal\job\groq.go` — function NewGroqExtractor: (apiKey, model string) *GroqExtractor, class GroqExtractor
- `internal\job\keywords.go` — function DetectTags: (text string) map[string][]string, class KeywordCategory
- `internal\job\levels.go` — class LevelRule
- `internal\job\location.go` — function NormalizeLocation: (rawLoc, description string) string
- `internal\job\regex.go` — function NewRegexExtractor: () *RegexExtractor, class RegexExtractor
- `internal\job\service.go` — function NewJobService: (store jobStore) *JobService, class JobService
- `internal\job\store.go`
  - function ComputeDedupHash: (company, title, jobType string) string
  - function NewJobRepository: (ctx context.Context, dbURL string) (*JobRepository, error)
  - class JobRepository
  - class UpsertStats
  - class PipelineStats
- `internal\scraper\scraper.go`
  - function NewScraperManager: (repo *job.JobRepository, collyScraper *job.CollyScraper, enricher *job.BatchEnricher, retentionDays int) *ScraperManager
  - class ScraperManager
  - interface Announcer
- `scraper-python\JoblessYu.py`
  - function compute_dedup_hash: (company, title, job_type)
  - function normalize_job_type: (raw_type)
  - function save_jobs_to_neon: (jobs_df)
  - function JobScan: ()

---

# Config

## Environment Variables

- `AI_MODEL` (has default) — .env.example
- `DATABASE_URL` (has default) — .env.example
- `DISCORD_ANNOUNCEMENT_CHANNEL_ID` **required** — internal\config\config.go
- `DISCORD_BOT_TOKEN` (has default) — .env.example
- `DISCORD_CHANNEL_ID` (has default) — .env.example
- `DISCORD_GUILD_ID` (has default) — .env.example
- `DISCORD_LOG_WEBHOOK_URL` **required** — .env.example
- `DISCORD_PUBLIC_KEY` **required** — cmd\lambda-bot\main.go
- `DISCORD_STATUS_CHANNEL_ID` **required** — internal\config\config.go
- `GROQ_API_KEY` (has default) — .env.example
- `JOB_RETENTION_DAYS` (has default) — .env.example
- `PORT` (has default) — .env.example
- `SCRAPE_TARGET_LIMIT` (has default) — .env.example

## Config Files

- `.env.example`
- `go.mod`

---

# Dependency Graph

## Most Imported Files (change these carefully)

- `JoblessYu/internal/job` — imported by **7** files
- `JoblessYu/internal/config` — imported by **6** files
- `log/slog` — imported by **6** files
- `encoding/json` — imported by **5** files
- `JoblessYu/internal/bot` — imported by **4** files
- `net/http` — imported by **3** files
- `JoblessYu/internal/scraper` — imported by **2** files
- `encoding/hex` — imported by **2** files
- `path/filepath` — imported by **2** files
- `os/signal` — imported by **1** files
- `crypto/ed25519` — imported by **1** files
- `net/url` — imported by **1** files
- `sync/atomic` — imported by **1** files
- `crypto/md5` — imported by **1** files
- `database/sql` — imported by **1** files
- `os/exec` — imported by **1** files

## Import Map (who imports what)

- `JoblessYu/internal/job` ← `cmd\bot\main.go`, `cmd\lambda-bot\main.go`, `cmd\lambda-pipeline\main.go`, `internal\bot\bot.go`, `internal\bot\embeds.go` +2 more
- `JoblessYu/internal/config` ← `cmd\bot\main.go`, `cmd\lambda-bot\main.go`, `cmd\lambda-pipeline\main.go`, `cmd\migrate\main.go`, `internal\bot\bot.go` +1 more
- `log/slog` ← `internal\bot\notifier.go`, `internal\job\colly_scraper.go`, `internal\job\enricher.go`, `internal\job\groq.go`, `internal\job\store.go` +1 more
- `encoding/json` ← `cmd\lambda-bot\main.go`, `cmd\lambda-pipeline\main.go`, `internal\bot\notifier.go`, `internal\job\groq.go`, `internal\job\store.go`
- `JoblessYu/internal/bot` ← `cmd\bot\main.go`, `cmd\lambda-bot\main.go`, `cmd\lambda-pipeline\main.go`, `internal\scraper\scraper.go`
- `net/http` ← `cmd\bot\main.go`, `cmd\lambda-bot\main.go`, `internal\bot\notifier.go`
- `JoblessYu/internal/scraper` ← `cmd\bot\main.go`, `cmd\lambda-pipeline\main.go`
- `encoding/hex` ← `cmd\lambda-bot\main.go`, `internal\job\store.go`
- `path/filepath` ← `cmd\migrate\main.go`, `internal\scraper\scraper.go`
- `os/signal` ← `cmd\bot\main.go`

---

# Test Coverage

> **100%** of routes and models are covered by tests
> 9 test files found

## Covered Models

- jobs

---

# CI/CD Pipelines

## GitHub Actions (1 workflow)

| Workflow | Triggers | Jobs | Deploy | Environments |
|---|---|---|---|---|
| CI | pull_request, push | 2 | — | — |

### CI

> `.github/workflows/ci.yml`

- **go** on `ubuntu-latest` — 6 steps
  - `actions/checkout@v4`
  - `actions/setup-go@v5`
  - `dominikh/staticcheck-action@v1`
- **python** on `ubuntu-latest` — 6 steps
  - `actions/checkout@v4`
  - `actions/setup-python@v5`

---
_Source: .github/workflows/ci.yml_
_Generated by codesight-cicd-plugin_

---

_Generated by [codesight](https://github.com/Houseofmvps/codesight) — see your codebase clearly_