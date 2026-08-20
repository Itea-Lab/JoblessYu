# Global Changelog

> Emergency backup context. Grand scheme from the foundation.
> Roll up progress-log entries here at slice boundaries so context survives session resets.

## [Slice T] — 2026-08-20

### Summary
P0 Runtime Safety & Long-Run Stability Pass. Fixed the Groq empty-response panic, made the parallel Colly scraper safe and cancellable, made the Postgres LISTEN loop reconnect correctly, serialized Discord hub refresh/debounce state, made bot shutdown idempotent, and added hard memory bounds to UI caches and captured Python subprocess output.

### Files touched
- modified: `internal/job/groq.go` and `internal/job/groq_test.go` (empty `choices` guard and regression test)
- modified: `internal/job/colly_scraper.go` and `internal/job/colly_scraper_test.go` (mutex-protected shared results and context cancellation test)
- modified: `internal/job/store.go` (fresh-connection LISTEN retry loop with context-aware backoff)
- modified: `internal/bot/bot.go` and `internal/bot/handlers.go` (serialized card refreshes, idempotent shutdown, bounded caches)
- created: `internal/bot/bot_test.go` (cache-cap regression test)
- modified: `internal/scraper/scraper.go` (64 KiB thread-safe tail capture for Python output)
- modified: `docs/AI/progress-log.md`, `docs/AI/structure.md`, and `docs/AI/modules.md` (runtime behavior and verification notes)

### Verification
- Go race-enabled tests: PASS
- Go vet: PASS
- Go build: PASS
- Python syntax check: PASS
- Ruff: one existing import-order issue remains

## [Slice S] — 2026-08-18 / 2026-08-20

### Summary
Groq Model Migration to `openai/gpt-oss-20b`, Multi-Key Pool & High-Density Prompt Compression. Migrated primary Groq AI LLM model from deprecated `llama-3.1-8b-instant` to `openai/gpt-oss-20b` (Groq's official direct non-reasoning replacement). Implemented Multi-Key pooling in `GroqExtractor` supporting comma-separated keys (`GROQ_API_KEY=key1,key2`) with round-robin balancing, per-key rate limiting, key-level TPD daily exhaustion isolation, and instant failover on 429 errors. Compressed `skills.md` from 9.8KB to 2.3KB (~400 tokens), cutting per-call token usage by ~75% while retaining 100% of 24 IT categories, Vietnamese seniority signals, tag categories, and schema constraints. Added TPD/RPD quota exhaustion interception in `enricher.go` with compound duration parsing (`16m3.36s`). Tuned throttle to 15s with `lastCall` timestamp-based adaptive rate-limiting in `groq.go`. Implemented `sanitizeJD()` to strip raw HTML tags, `<script>`/JSON-LD blobs, and decode HTML entities.

### Files touched
- modified: `internal/config/config.go` (updated default `AIModel` to `openai/gpt-oss-20b`)
- modified: `.env` & `.env.example` (updated `AI_MODEL=openai/gpt-oss-20b`)
- modified: `internal/job/enricher.go` (added TPD daily quota fallback and compound retry parser)
- modified: `internal/job/groq.go` (added `sanitizeJD()`, `lastCall` adaptive throttle, tuned 15s delay & 800 max tokens)
- modified: `internal/job/skills.md` (added rule 5 forbidding `<think>` tags)
- modified: `README.md` (updated model references in documentation)
- modified: `docs/AI/modules.md` (updated Groq model specs)
- modified: `docs/AI/progress-log.md` (recorded Slice S progress)
- modified: `docs/AI/changelog.md` (this entry)

---

## [Slice Q & R] — 2026-08-14

### Summary
Strict Asset Validation, Fail-Fast Refactoring & Loophole Patches. Enforced strict validation for `DISCORD_CHANNEL_ID` in `config.Load()` (`log.Fatal`), eliminating unsafe channel guessing. Enhanced `DISCORD_GUILD_ID` log message to warn about Discord's 1-hour global command caching propagation window. Updated `bot.Start()` to return `ApplicationCommandBulkOverwrite` errors instead of swallowing them. Exported `ErrDisabledAPIKey` in `groq.go` and updated `enricher.go` to handle disabled AI mode cleanly without retries or fake network logs. Fixed redundant `break` statement in `enricher.go`. Added `HealthCheckPeriod` (1m) and `MaxConnLifetime` (30m) to `pgxpool.Config` in `store.go` to prune dead TCP sockets caused by Neon serverless DB idle suspends. Bounded `handleSearchJobs` with a 10-second `context.WithTimeout` deadline in `handlers.go`.

### Files touched
- modified: `internal/config/config.go` (added `DISCORD_CHANNEL_ID` check, enhanced `DISCORD_GUILD_ID` warning log)
- modified: `internal/bot/bot.go` (propagated slash command registration errors)
- modified: `internal/bot/notifier.go` (removed auto-discovery guessing, enforced strict channel resolution)
- modified: `internal/job/groq.go` (exported `ErrDisabledAPIKey`)
- modified: `internal/job/enricher.go` (handled `ErrDisabledAPIKey` without retries, removed redundant `break`)
- modified: `internal/job/store.go` (added `HealthCheckPeriod` & `MaxConnLifetime` to `pgxpool.Config`)
- modified: `internal/bot/handlers.go` (added 10s context deadline to `handleSearchJobs`)
- modified: `.env` & `.env.example` (standardized to `DISCORD_GUILD_ID` and `DISCORD_CHANNEL_ID`)
- modified: `docs/AI/modules.md` (updated architecture notes)
- modified: `docs/AI/progress-log.md` (recorded Slice Q & R progress)
- modified: `docs/AI/changelog.md` (this entry)

---

## [Slice P] — 2026-08-11

### Summary
macOS Terminal Quit Signal Handling (`SIGHUP` / `Cmd+Q`). Registered `syscall.SIGHUP` and `syscall.SIGQUIT` in `signal.Notify` in `cmd/bot/main.go`. Closing terminal windows or quitting terminal via `Cmd+Q` now triggers the exact same graceful shutdown sequence as `Ctrl+C`, ensuring Card 1 updates to `🔴 OFFLINE` in Discord.

### Files touched
- modified: `cmd/bot/main.go` (registered `SIGHUP` and `SIGQUIT` in `signal.Notify`)
- modified: `docs/AI/progress-log.md` (recorded Slice P progress)
- modified: `docs/AI/changelog.md` (this entry)

---

## [Slice O] — 2026-08-11

### Summary
Multi-Checkbox `/jobs` Search UI/UX, Fixed-Height Select Menu Layout & `skills.md` Alignment. Upgraded `/jobs` filter dropdowns (`select_position`, `select_level`, `select_location`, `select_type`) to use multi-select checkboxes (`MinValues: 0`, `MaxValues: 3..5`). Shortened chip labels (`Web Dev`, `Mobile Dev`, `IT Management`) and moved detailed role descriptions into `Description` fields inside dropdown items to keep select menu height fixed (~40px single-line) without flexing or vertical expanding. Enforced mutual exclusivity for "All" in `normalizeSelectionValues`. Expanded Seniority Levels (`Intern`, `Fresher`, `Junior`, `Middle`, `Senior`, `Lead / Manager`) to match `skills.md` and Locations (`Ho Chi Minh`, `Ha Noi`, `Da Nang`, `Remote / WFH`) to match Card 2 metrics. Built dynamic slice-based SQL query generator (`FetchRawJobs`) in `store.go`. Fully documented all 12 Makefile targets in `README.md`.

### Files touched
- modified: `internal/bot/embeds.go` (configured multi-select select menus with short chip labels, rich descriptions, and optimal `MaxValues`)
- modified: `internal/bot/handlers.go` (updated `handleSweeperComponent` and `normalizeSelectionValues` to handle multi-select choices and mutual exclusivity)
- modified: `internal/job/entry.go` (expanded `JobQuery` struct with slice fields `Levels`, `JobTypes`, `Locations`, `Expertises`)
- modified: `internal/job/store.go` (updated `FetchRawJobs` query builder for multi-value SQL filtering)
- modified: `internal/bot/embeds_test.go` (added unit tests for multi-select component generation and slice label formatters)
- modified: `README.md` (documented all 12 Makefile targets)
- modified: `docs/AI/progress-log.md` (recorded Slice O progress)
- modified: `docs/AI/changelog.md` (this entry)

---

## [Slice N] — 2026-08-11

### Summary
User-Centric Discord Hub Redesign, Multi-Keyword Database Breakdown, Real-Time Hybrid DB Listener & Production Containerization. Redesigned Card 2 (`🌅 SCRAPE SUMMARY`) to remove technical dev clutter (`duplicates linked`, `AI Categorized`) and present high-value job seeker insights (**Timestamps**, **Fresh Roles Today**, **Total Active Pool**, **Experience Level Breakdown** (`Intern / Fresher`, `Junior / Mid`, `Senior`, `Lead / Manager`), and **Top Locations** (`Ho Chi Minh`, `Ha Noi`, `Da Nang`, `Remote`)). Implemented multi-keyword recognition in `GetPipelineSummaryStats` handling Vietnamese diacritics (`Hà Nội`, `HN`, `Hồ Chí Minh`, `HCM`, `SG`, `Đà Nẵng`, `Remote`, `Junior / Mid`, `Senior`, `Lead`). Added Postgres `LISTEN jobs_changed` + 30s count monitor in `bot.go` and static image containerization with HTTP `/healthz` probe.

### Files touched
- modified: `internal/bot/embeds.go` (redesigned `BuildDailyAnnouncementEmbed` for user-centric level and location breakdowns)
- modified: `internal/job/store.go` (expanded `GetPipelineSummaryStats` ILIKE queries for multi-keyword location & level recognition)
- modified: `internal/bot/notifier.go` (updated `DailyScrapeSummary` struct with breakdown fields)
- modified: `internal/bot/bot.go` (updated `fetchDailySummaryDetails` and hybrid real-time DB listener)
- modified: `internal/scraper/scraper.go` (updated `runScrapeAndEnrich` summary population)
- modified: `Dockerfile` (multi-stage Go + Python hybrid runtime build)
- modified: `cmd/bot/main.go` (added HTTP `/healthz` endpoint on port 8080)
- created: `migrations/007_add_notify_trigger.sql` (Postgres `notify_jobs_changed()` trigger)

---

## [Slice M] — 2026-08-11

### Summary
Discord Dual-Static Hub Architecture, Real-Time DB Interlink, and Groq 3-Job Batch Tuning. Implemented 2 static pinned cards (`Card 1: Status` and `Card 2: Daily Summary`) in the Hub channel with automatic message ordering cleanup and live 24h DB metrics queries. Updated Groq LLM extraction batch size to 3 jobs (`batchChunkSize = 3`) to strictly stay under Groq Free Tier's 6,000 TPM limit (~2,425 tokens/call with >3,500 tokens of headroom). Executed migration `006_schema_cleanup.sql` and `VACUUM FULL jobs;`.

### Files touched
- modified: `internal/job/enricher.go` (updated `batchChunkSize` to 3 jobs per batch)
- modified: `internal/bot/notifier.go` (added `dailySummaryMsgID`, `UpdateDailySummaryCard` with auto-reordering, ICT time formatting)
- modified: `internal/bot/embeds.go` (updated `BuildStatusCardEmbed` and `BuildDailyAnnouncementEmbed` for dual static cards layout)
- modified: `internal/bot/bot.go` (added `fetchDailySummaryDetails` and updated `bot.Start` to initialize both static cards on boot)
- modified: `internal/job/store.go` (added `GetPipelineSummaryStats` live 24h query method)
- modified: `internal/job/service.go` (added `GetPipelineSummaryStats` delegation method)
- created: `migrations/006_schema_cleanup.sql` (consolidated `job_type` and dropped `ai_model`/`job_type_normalized` bloat)

---

## [Slice L] — 2026-08-10

### Summary
JoblessYu Performance Optimization Suite. Implemented 10-job LLM batching (`ExtractBatch`), 30/30/30 scrape targets, Postgres trigram GIN indexes (`005_add_trgm_indexes.sql`), Colly scraper concurrency tuning (4 workers), and `pgxpool` connection pool management.

### Files touched
- modified: `internal/job/groq.go` (implemented `ExtractBatch` and `callGroqBatch`)
- modified: `internal/job/enricher.go` (updated `BatchEnricher.Run` to process 10-job chunks with fallback)
- modified: `internal/job/colly_scraper.go` (updated `itviecMaxJobs` to 30 and Colly parallelism to 4)
- modified: `scraper-python/JoblessYu.py` (updated JobSpy `results_wanted` limit to 30)
- modified: `internal/scraper/scraper.go` (updated log targets to 30/30/30)
- created: `migrations/005_add_trgm_indexes.sql` (added `pg_trgm` GIN indexes on `jobs(title)`)
- modified: `internal/job/store.go` (configured `pgxpool` MaxConns/MinConns limits)

---

## [Slice K] — 2026-08-10

### Summary
AI Evaluation & Hallucination Benchmark Harness (`make eval-ai` / `go run ./cmd/bot -eval-ai`). Built automated benchmark dataset and evaluation suite (`RunAIEval`, `PrintEvalReport`) measuring Level, Expertise, and Location accuracy rates, alongside detecting hallucinations (bogus categories or false `Intern` classifications).

### Files touched
- created: `internal/job/eval.go` (`RunAIEval`, `GetBenchmarkDataset`, `PrintEvalReport`)
- created: `internal/job/eval_test.go` (unit test executing evaluation engine against regex fallback extractor)
- modified: `cmd/bot/main.go` (added `-eval-ai` CLI flag)
- modified: `Makefile` (added `eval-ai` target)
- modified: `docs/AI/progress-log.md` (recorded Slice K entry)
- modified: `docs/AI/changelog.md` (this entry)

---

## [Slice J] — 2026-08-10

### Summary
Deterministic Location Normalizer & Hallucination Guardrails (`NormalizeLocation`). Solved location misclassifications (e.g. ITViec sidebar `Country: Vietnam` overriding specific street addresses like `Thanh Xuân, Ha Noi`). Automatically scans raw location strings and job description body text for specific city/district keywords (`Hanoi`, `Ho Chi Minh`, `Da Nang`, `Remote`).

### Files touched
- created: `internal/job/location.go` (`NormalizeLocation` helper with Hanoi, HCM, Da Nang, Remote keyword dictionaries)
- created: `internal/job/location_test.go` (unit tests covering ITViec Thanh Xuan Hanoi JD, HCM districts, Da Nang, Remote, and generic fallback)
- modified: `internal/job/colly_scraper.go` (applied `NormalizeLocation` against page body text when scraping ITViec)
- modified: `internal/job/store.go` (applied `NormalizeLocation` on `UpsertJobs` and `FetchRawJobs`)
- modified: `docs/AI/future.md` (updated AI Location Guardrails to Completed Slice J)
- modified: `docs/AI/progress-log.md` (recorded Slice J entry)

---

## [Slice I] — 2026-08-10

### Summary
Discord Announcement & UI Visibility System (Phase 4 UI). Added static availability status card (`🟢 ONLINE` / `🔴 OFFLINE` Abyss Bot pattern), public 5:00 AM daily job scrape update announcements, developer log webhooks with server text channel auto-discovery fallback, and a CLI test suite (`make test-notify`).

### Files touched
- created: `internal/bot/notifier.go` (`Notifier` struct, `UpdateStatusCard`, `PostDailyScrapeAnnouncement`, `PostDevLogWebhook`, `resolveChannelID` fallback)
- created: `internal/bot/embeds_test.go` (unit tests for status card and announcement embed builders)
- modified: `internal/bot/embeds.go` (added `BuildStatusCardEmbed`, `BuildDailyAnnouncementEmbed`)
- modified: `internal/bot/bot.go` (wired `Notifier` struct, `Start()` online card hook, `Stop()` offline card hook)
- modified: `internal/config/config.go` (added `DiscordStatusChannelID`, `DiscordAnnouncementChannelID`, `DiscordLogWebhookURL`)
- modified: `internal/scraper/scraper.go` (added `Announcer` interface, triggered announcements & dev webhooks upon 5:00 AM pipeline completion)
- modified: `cmd/bot/main.go` (wired `scraperMgr.SetAnnouncer(disbot.Notifier())`, added `-test-notify` CLI flag & `runTestNotifications` suite)
- modified: `Makefile` (added `test-notify` target)
- modified: `.env.example` (documented optional channel IDs & webhook env vars)
- modified: `docs/AI/future.md` (updated UI status card & announcement status to Completed)
- modified: `docs/AI/progress-log.md` (recorded 2026-08-10 Slice I entry)
- modified: `docs/AI/structure.md` (updated layout diagram & design decisions)
- modified: `docs/AI/changelog.md` (this entry)

### Decisions
- Abyss Bot Pattern: Static availability card pinned in `#joblessyu-status` dynamically updates to `🟢 ONLINE` on boot and `🔴 OFFLINE` on graceful shutdown.
- Server Auto-Discovery: If channel IDs are empty in `.env`, `resolveChannelID` automatically discovers the connected server's primary text channel (`#general`) so notifications deliver out-of-the-box.
- Public Daily Announcements: 5:00 AM ICT pipeline completion posts a rich summary embed detailing total scraped, fresh added, merged duplicates, AI enriched, and active pool metrics.

---

## [Slice H] — 2026-08-05

### Summary
Scraper capacity scaling to 40-40-40 (Indeed, LinkedIn, ITViec), Option 2 box-drawing terminal logger with zero emojis, full deduplication & queue metrics transparency, and Windows `.venv\Scripts\python.exe` path resolution fix with Python 3.13 + Numpy 2.5.1 stability updates.

### Files touched
- modified: `scraper-python/JoblessYu.py` (results_wanted=40, added inserted_count & merged_count reporting, removed emoticons)
- modified: `internal/job/colly_scraper.go` (itviecMaxJobs=40)
- modified: `internal/job/store.go` (added UpsertStats, CountEnrichmentStats, updated UpsertJobs)
- modified: `internal/scraper/scraper.go` (Option 2 box logger, Windows Scripts/python.exe lookup, deduplication & queue stats output)
- modified: `README.md` (updated architecture diagram, job counts 40-40-40, Windows PowerShell venv setup)
- modified: `docs/AI/progress-log.md` (recorded 2026-08-05 Slice H updates)
- modified: `docs/AI/structure.md` (updated layout & design decisions)
- modified: `docs/AI/modules.md` (updated scraper limits & Windows venv)
- modified: `docs/AI/future.md` (updated cross-site deduplication status to Completed)
- modified: `docs/AI/changelog.md` (this entry)

### Decisions
- Scraper Range: 40-40-40 target (40 Indeed + 40 LinkedIn + 40 ITViec) to maximize fresh daily tech listings.
- No Emojis: Replaced all emojis in CLI logs with clean ASCII box drawing cards and summary tables.
- Deduplication Transparency: `UpsertJobs` returns `Inserted` vs `Merged` counts to explain exact numbers before AI batch enrichment.
- Windows Cross-Platform: `resolvePythonExe()` checks `.venv\Scripts\python.exe` first on Windows hosts.

---

## [Slice G] — 2026-07-31

### Summary
Scraper cron scheduler timezone fix and real-time Python output streaming. Fixed a location configuration issue in `cron.New()` where cron schedule expressions were evaluated against `time.Local` (causing daily scrape to run at 22:00 ICT / 10:00 PM ICT instead of 05:00 AM ICT). Streamed Python subprocess `stdout` and `stderr` to `os.Stdout`/`os.Stderr` so JobSpy scraping progress is visible live in logs.

### Files touched
- modified: `internal/scraper/scraper.go` (added `cron.WithLocation(time.UTC)`, refactored `runPythonScraper` to stream stdout/stderr via `io.MultiWriter`)
- modified: `docs/AI/progress-log.md` (recorded 2026-07-31 cron & streaming updates)
- modified: `docs/AI/changelog.md` (this entry)

### Decisions
- UTC Location: `cron.New(cron.WithLocation(time.UTC))` guarantees that `"0 22 * * *"` evaluates strictly against 22:00 UTC (5:00 AM ICT) regardless of host system timezone.
- Streamed Subprocess Output: `io.MultiWriter` allows real-time stdout visibility while preserving string capture for regex job count parsing.

---

## [Slice F] — 2026-07-30

### Summary
Feature pass & UI overhaul. Adopted ITViec's canonical 24-category IT expertise index across the Discord UI, Python JobSpy scraper, Go Colly scraper, and Groq AI classifier. Made the `/jobs` slash command and job result cards 100% ephemeral (`"Only you can see this"`) for total user privacy and zero channel clutter. Added self-healing pagination cache fallback, cross-site job deduplication (7-day window with `alternate_urls`), job retention set to 30 days, location diacritic matching for Vietnamese cities, `Fresher` level support, and `job_type` normalization (`Full-time`, `Part-time`, `Contract`, `All`).

### Files touched
- modified: `JoblessYu.md` (formatted user guide & ephemeral privacy features)
- modified: `README.md` (updated architecture diagram, 24 expertise categories, ephemeral privacy details, retention=30)
- modified: `docs/AI/modules.md` (updated 24 categories, deduplication, ephemeral features)
- modified: `docs/AI/structure.md` (updated file layout and design decisions)
- modified: `internal/job/expertise.go` (updated `ExpertiseCategories` to 24 categories with `embedded_iot` as "Embedded, IoT & Robotics")
- modified: `internal/job/expertise_test.go` (updated unit tests for 24 categories)
- modified: `internal/job/schema.go` (updated Groq AI schema prompt to 24 categories)
- modified: `internal/job/skills.md` (updated bilingual AI system prompt for 24 categories and Contract job type)
- modified: `internal/job/colly_scraper.go` (expanded DOM selectors `.paragraph-content`, `.job-details__paragraph`, `.job-details__overview` + paragraph aggregator fallback)
- modified: `internal/job/store.go` (added location diacritic matching, default limit=1000, `job_type_normalized` query matching)
- modified: `internal/bot/embeds.go` (24 position options + All Positions = 25 options total, added `formatJobTypePill`, level & job type dropdown menus)
- modified: `internal/bot/handlers.go` (100% ephemeral `/jobs` responses & followup cards, `mapJobTypeToQuery`, `mapLevelToQuery` with Fresher)
- modified: `scraper-python/JoblessYu.py` (canonical ITViec search terms, non-IT title exclusion filter, `normalize_job_type`)
- modified: `internal/scraper/scraper.go` (fixed `StopSchedule` IDE warning)

### Decisions
- Discord Select Menu Limit: Combined `Embedded Systems` and `IoT & Robotics` into `Embedded, IoT & Robotics` to fit 24 categories + `All Positions` = 25 options (exact Discord maximum).
- Ephemeral Privacy: `/jobs` filter menu and result cards use `MessageFlagsEphemeral` so interactions are visible strictly to the prompter (`"Only you can see this"`).
- Cross-Site Deduplication: 7-day `dedup_hash` (Company + Title + JobType) lookup merges duplicate URLs into `alternate_urls` JSONB.

---

## [Slice A] — 2026-07-20

### Summary
Cleanup and hygiene pass. Removed dead code (`!testembed`, `jobs.json` write, `IntentsMessageContent`), renamed `Python-Jobspy/` → `scraper-python/`, pinned Python deps, added project tooling (Makefile, CI workflow, PR template, CONTRIBUTING.md), created local AI docs structure (`docs/AI/` gitignored), and protected personal files (`neoenv.txt`, `Note.md`, `GroqImplement.md`) from accidental commits. No behavior change — bot still works exactly as before, just cleaner.

### Files touched
- modified: `.gitignore` (added docs/AI/, neoenv.txt, GroqImplement.md, /bot, backup.sql)
- modified: `README.md` (path updates Python-Jobspy → scraper-python, removed DISCORD_CHANNEL_ID, added GROQ_API_KEY, updated install instructions to use requirements.txt)
- modified: `internal/bot/bot.go` (removed !testembed handler + IntentsMessageContent intent)
- modified: `internal/scraper/scraper.go` (path reference Python-Jobspy → scraper-python)
- renamed: `Python-Jobspy/JoblessYu.py` → `scraper-python/JoblessYu.py` (also removed jobs.to_json line)
- created: `.env.example`
- created: `Makefile`
- created: `scraper-python/requirements.txt`
- created: `.github/workflows/ci.yml`
- created: `.github/pull_request_template.md`
- created: `CONTRIBUTING.md`
- created: `docs/AI/progress-log.md`
- created: `docs/AI/structure.md`
- created: `docs/AI/modules.md`
- created: `docs/AI/changelog.md` (this file)
- untouched: `Note.md` (personal, gitignored)
- untouched: `neoenv.txt` (credentials, gitignored)
- untouched: `GroqImplement.md` (Groq reference snippet, gitignored)

### Decisions
- Trunk-based branching: `main` + short feature branches, squash-merge. `test-commit` branch to be deleted.
- `DISCORD_CHANNEL_ID` dropped (unused). `DISCORD_GUILD_ID` kept (needed for slash command registration).
- `DISCORD_CHANNEL_ID` was never read by Go code — confirmed via grep before removal.
- `IntentsMessageContent` removed because only `!testembed` used it; slash commands don't need message content intent.
- `jobs.json` write removed because Go never reads it — DB (Neon) is the single source of truth.
- `qwen/qwen3.6-27b` chosen as Groq model (per user's GroqImplement.md). Does NOT support strict JSON mode — Slice D will use best-effort JSON + retry.
- Scraper schedule change deferred to Slice C (currently still `@every 6h`; will become weekly Monday 09:00 ICT = cron `0 2 * * 1` UTC).

### Verification
- `go vet ./...`: clean
- `go test -race -cover ./...`: PASS (domain 95.2% coverage, service 66.2%)
- `go build ./cmd/bot`: PASS
- `python -m py_compile scraper-python/JoblessYu.py`: PASS
- Git status: 4 modified, 1 renamed, 7 new files. Personal files (neoenv.txt, Note.md, GroqImplement.md, docs/AI/) correctly gitignored.

### Known issues
- `scraper-python/.venv` has stale hardcoded paths from old `Python-Jobspy/` location. User must recreate the venv.
- `backup.sql` exists in repo root (from user's earlier pg_dump). Added to .gitignore. Can be deleted.
- `bot` binary exists from `go build` verification. Added to .gitignore. Can be deleted via `make clean`.
- Branch consolidation (PR base-JoblessYu → main, delete test-commit) not yet done — deferred to user.

## [Slice B] — 2026-07-20

### Summary
Modular restructure with zero behavior change. Extracted the `ai.Extractor` interface and `domain.JobMeta`/`domain.JobQuery` types so the service layer no longer hardcodes detection logic. Moved all regex detection (level, type, tags) from `internal/service/` into the new `internal/ai/` package as `RegexExtractor` — the first `Extractor` implementation. Slice D will add `GroqExtractor` as the primary implementation with `RegexExtractor` as fallback, swapping in by changing one line in `main.go` with zero edits to service or bot layers. That's the modular contract this slice establishes.

### Files touched
- modified: `cmd/bot/main.go` (imports `internal/ai`, wires `ai.NewRegexExtractor()` into service)
- modified: `internal/bot/handlers.go` (builds `domain.JobQuery` from slash command options instead of passing 4 args)
- modified: `internal/domain/job.go` (added `JobMeta`, `JobQuery` types + `ApplyMeta` method)
- modified: `internal/repository/job_postgres.go` (`FetchRawJobs` signature: `(ctx, domain.JobQuery)` instead of 4 args; added configurable `Limit`)
- modified: `internal/service/job_service.go` (rewritten as thin orchestrator: fetch → extract → apply → filter; defines `jobStore` interface; accepts `ai.Extractor`)
- modified: `internal/service/job_service_test.go` (detection tests moved to `ai/fallback_test.go`; `ApplyMeta` tests moved to `domain/job_test.go`; kept `TestLevelUnknownConstant` as service filter contract)
- deleted: `internal/service/keywords.go` (moved to `internal/ai/keywords.go`)
- deleted: `internal/service/keywords_test.go` (moved to `internal/ai/fallback_test.go`)
- created: `internal/ai/extractor.go` (`Extractor` interface)
- created: `internal/ai/fallback.go` (`RegexExtractor` implementing `Extractor`; moved from `service/job_service.go` `detectJobMeta`)
- created: `internal/ai/keywords.go` (`keywordCategories` + `DetectTags`; moved from `service/keywords.go`)
- created: `internal/ai/fallback_test.go` (all detection + keyword tests moved from service)
- created: `internal/ai/skills.md` (placeholder; Slice D fills it with the Groq system prompt)
- created: `internal/domain/job_test.go` (`ApplyMeta` tests)

### Decisions
- **Interface location**: `ai.Extractor` lives in the `ai` package (co-located with implementations). `service.jobStore` (unexported) lives in `service` (consumer-side, per Go's "accept interfaces" convention). `repository.JobRepository` satisfies `jobStore` structurally — no explicit `implements` keyword needed.
- **Query type**: `domain.JobQuery` replaces 4 separate args (`level, jobType, location string, includeUnknown bool`). Lives in `domain` (shared, no dependency direction issues). Added `Limit int` field (defaults to 20 when zero).
- **JobMeta shape**: Includes `Salary`, `Remote`, `Summary` fields that regex can't populate. They're zero-valued by `RegexExtractor` and populated by `GroqExtractor` in Slice D. This avoids a type change in Slice D.
- **ApplyMeta semantics**: DB value wins for Level/Type (only fill if empty). Tags always overwritten (not yet stored in DB — Slice C adds a `tags` JSONB column).
- **`domain/levels.go` UNCHANGED**: `LevelRules`, `FindLevelWord`, `LevelWordAlternationForRegex`, `NoExpAlternationForRegex` still the single source of truth for level patterns. Used by both `ai/fallback.go` (Go regex) and `repository/job_postgres.go` (Postgres regex). Slice C may simplify the repository's SQL predicates since AI will own level detection, but that's a behavior change deferred to Slice C.
- **No new dependencies added**: Slice B is pure refactor. `go.mod` unchanged.

### Verification
- `go vet ./...`: clean
- `go test -race -cover ./...`: PASS
  - `internal/ai`: 94.4% coverage (new package, moved detection logic + tests)
  - `internal/domain`: 96.2% coverage (up from 95.2% pre-Slice-B — added ApplyMeta tests)
  - `internal/service`: 0% coverage (thin orchestrator now; needs mock `jobStore` for real coverage — Slice D concern when testing AI+regex fallback chain)
- `go build ./cmd/bot`: PASS
- Behavior identical to pre-Slice-B: regex still does all classification. Bot boots, `/jobs` works against existing Neon data.

### Known issues
- `internal/service` has 0% coverage. The `FetchAndProcessJobs` method needs a mock `jobStore` to test — deferred to Slice D when we add the Groq extractor and need to verify the fallback chain.
- `internal/ai/skills.md` is a placeholder. Slice D fills it with the full categorization guide and loads it via `//go:embed`.
- Branch consolidation still pending (carried over from Slice A).

## [Slice B.5] — 2026-07-20

### Summary
Entanglement cleanup. Fixed 4 critical/high issues identified in the Slice B review: (1) `domain/levels.go` was a fake domain module containing a regex engine — moved the engine to `ai/fallback.go` as private, leaving domain as pure data; (2) `ApplyMeta` was a business rule living on a domain type — inlined it in `service.FetchAndProcessJobs` so domain stays pure and Slice D can flip the rule with a 3-line edit; (3) `bot.Bot.jobService` was a concrete type — replaced with a consumer-side interface so bot handlers become testable without a real Postgres; (4) `domain.JobEntry` had no `ID` field — added it + updated the repository SELECT/scan so Slice D's `MarkAIProcessed(ctx, jobID, meta)` can identify rows. Zero behavior change; bot still works identically.

### Files touched
- modified: `internal/domain/levels.go` (stripped to pure data: `LevelRule` struct, `LevelRules` slice, `LevelUnknown` const — no `regexp` import, no functions, ~50 lines from 119)
- modified: `internal/domain/levels_test.go` (kept only data tests: `TestLevelRulesSingleSource`, `TestLevelUnknownConstant`; removed `TestFindLevelWord`, `TestFindLevelWord_InternalNotIntern`, `TestNoExpAlternationForRegex` — moved to ai)
- modified: `internal/domain/job.go` (removed `ApplyMeta` method; added `ID int64` field to `JobEntry`)
- deleted: `internal/domain/job_test.go` (3 `ApplyMeta` tests no longer apply — logic is inline in service now)
- modified: `internal/ai/fallback.go` (absorbed regex engine from domain as private: `findLevelWord`, `levelWordAlternation`, `noExpAlternation`, `compilePerRuleWordRE`, `levelWordRE`, `perRuleWordRE`; ~40 lines added)
- modified: `internal/ai/fallback_test.go` (added `TestFindLevelWord`, `TestFindLevelWord_InternalNotIntern`, `TestNoExpAlternationForRegex` — moved from domain)
- modified: `internal/bot/bot.go` (added `jobService` interface; changed `Bot.jobService` field from `*service.JobService` to `jobService`; changed `NewBot` signature; removed `internal/service` import)
- modified: `internal/repository/job_postgres.go` (added `id` to SELECT, scan, and `domain.JobEntry` assignment)
- modified: `internal/service/job_service.go` (inlined `ApplyMeta` logic — 3 lines replacing `j.ApplyMeta(meta)` call)
- modified: `internal/service/job_service_test.go` (updated stale comment about ApplyMeta tests location)

### Decisions
- **Domain purity enforced**: `domain/levels.go` now contains only data declarations. No `regexp` import, no `var` blocks with closures, no functions. This means domain can be imported by any package (including repository and ai) without risk of init-time side effects or circular deps.
- **Repository still reads `domain.LevelRules` strings**: this is data reading (not running Go regex), so it's fine. Repository uses the strings to build Postgres `\m...\M` regex predicates. Slice C eliminates this coupling when the `level` column replaces the SQL regex predicates — AI writes level into the column, repository filters on the column value, no regex needed in SQL.
- **`ApplyMeta` inlined, not extracted to a private helper**: the merge rule is applied in exactly one place. YAGNI — extract a function only when there are 2+ call sites. If Slice D needs to flip the rule (AI wins instead of DB wins), it's a 3-line edit in one place, not a function signature change.
- **`bot.jobService` interface is unexported**: bot is a leaf package; no other package needs to satisfy the interface. Unexported keeps the API surface tight while still enabling mock-based tests.
- **`JobEntry.ID int64`**: zero when not from DB (e.g. in tests). Repository always populates it. Slice D's `MarkAIProcessed` will receive `j.ID` to identify which row to update.
- **Config additions deferred to Slice C**: adding `AIProvider`/`GroqAPIKey`/`AIModel` to `config.Config` is a Slice C concern (it's a config change, not a structural fix). Keeping B.5 focused on the 4 structural issues.

### Verification
- `go vet ./...`: clean
- `go test -race -cover ./...`: PASS
  - `internal/ai`: 94.7% coverage (up from 94.4% — absorbed moved `findLevelWord` tests)
  - `internal/domain`: [no statements] — correct, pure data package now (no functions to cover)
  - `internal/service`: 0% coverage (thin orchestrator — Slice D adds mock-based tests)
- `go build ./cmd/bot`: PASS
- Behavior identical to pre-Slice-B.5: bot boots, `/jobs` works against existing Neon data.

### Known issues
- Repository still has package-init `var` blocks (lines 25-51) that read `domain.LevelRules` and build Postgres regex maps. These are data-only (no Go regex compilation), so they're safe, but they're the LAST cross-cutting regex coupling. Slice D eliminates them when the `level` column replaces the SQL regex predicates.
- Branch consolidation still pending (carried over from Slice A).
- `cmd/bot/main.go` unchanged — `*service.JobService` satisfies `bot.jobService` structurally, so the swap is transparent at the call site.

## [Slice C] — 2026-07-20

### Summary
Neon backend migrations. Extracted DB schema from Python's runtime `CREATE TABLE` into version-controlled SQL files (`migrations/001_init.sql`, `migrations/002_add_ai_columns.sql`). Added 8 AI-enrichment columns + 3 indexes. Added AI config fields to `config.Config` (`AIProvider`, `GroqAPIKey`, `AIModel`). Added `MarkAIProcessed` method to repository (Slice D calls it to persist AI results). Made Python scraper params env-driven (`SEARCH_TERM`, `RESULTS_WANTED`, etc.). Changed cron from `@every 6h` to weekly Monday 09:00 ICT. Removed startup scrape goroutine. Zero behavior change — existing regex filtering still works because the new DB columns start NULL.

### Files touched
- created: `migrations/001_init.sql` (canonical schema, extracted from Python)
- created: `migrations/002_add_ai_columns.sql` (8 columns + 3 indexes)
- modified: `internal/config/config.go` (added `AIProvider`, `GroqAPIKey`, `AIModel` fields + defaults + validation)
- modified: `internal/repository/job_postgres.go` (added `encoding/json` import + `MarkAIProcessed` method)
- modified: `internal/scraper/scraper.go` (cron `@every 6h` → `0 2 * * 1`; removed startup scrape goroutine; fixed stale comments)
- modified: `scraper-python/JoblessYu.py` (hardcoded params → env-driven with defaults)
- modified: `.env.example` (added 6 scraper env vars)

### Decisions
- **Behavior preserved on FetchRawJobs**: the WHERE clause and SELECT are UNCHANGED. The regex-based level predicates still work because new DB columns start NULL. If I had changed the WHERE to `WHERE level = $1`, all existing rows (with NULL level) would be filtered out — breaking `/jobs level:Senior`. The WHERE simplification is deferred to Slice D after AI backfills existing rows.
- **MarkAIProcessed takes a `model` param**: identifies which extractor produced the result ("regex", "qwen/qwen3.6-27b"). Stored in `ai_model` column for audit. This makes the method reusable for any extractor without coupling to a specific AI provider.
- **Nil tags → `{}` JSONB**: `json.Marshal(nil)` returns `"null"` which is valid JSONB but semantically wrong for an empty map. Explicit check ensures `{}` is written instead.
- **Scraper env vars inherited from parent process**: Go's `exec.CommandContext` inherits the parent's environment, so the bot's `.env` (loaded by `godotenv`) is automatically available to the Python subprocess. No explicit env passing needed.
- **No `/scrape now` admin command**: deferred. Manual scrape via `make scrape` is sufficient. Adding a Discord slash command would require coupling bot → scraper (via channel or direct reference), which adds complexity for a feature that's only needed during development.
- **Weekly cron**: `0 2 * * 1` = Monday 02:00 UTC = Monday 09:00 ICT (ICT = UTC+7). Weekly instead of 4×/day — reduces Groq API usage by 28×, stays well within free tier limits.

### Verification
- `go vet ./...`: clean
- `go test -race -cover ./...`: PASS (all packages, no coverage change)
- `go build ./cmd/bot`: PASS
- `python -m py_compile scraper-python/JoblessYu.py`: PASS
- **Migrations NOT yet applied to Neon** — user must run `make migrate` before Slice D.

### Known issues
- Migrations need to be applied to Neon: `make migrate` (requires `DATABASE_URL` in `.env`).
- `FetchRawJobs` WHERE clause still uses regex predicates (deferred to Slice D after AI backfill).
- `levelWordPatterns`/`levelYearPatterns`/`levelNoExpPatterns` maps still in repository (deferred to Slice D).
- Branch consolidation still pending (carried over from Slice A).

## [Slice D] — 2026-07-20

### Summary
AI core with Groq. Implemented `GroqExtractor` calling Groq's OpenAI-compatible API (model: `qwen/qwen3.6-27b`) with the full categorization guide in `skills.md`. Built `ChainExtractor` for hybrid AI-primary + regex-fallback — when Groq fails or rate-limits, regex takes over so the bot always returns results. Added lazy enrichment: un-enriched rows are extracted on first `/jobs` query, persisted via `MarkAIProcessed`, and skipped on future queries. Added 6-second throttling to stay under Groq's 8,000 TPM free-tier limit. When `GROQ_API_KEY` is not set, the chain degrades gracefully to regex-only.

### Files touched
- created: `internal/ai/groq.go` (GroqExtractor: go-openai client, throttle, retry, JSON validation)
- created: `internal/ai/chain.go` (ChainExtractor: primary + fallback)
- created: `internal/ai/schema.go` (JSON schema description for prompt)
- created: `internal/ai/chain_test.go` (chain fallback logic tests)
- created: `internal/ai/groq_test.go` (truncate helper test)
- modified: `internal/ai/skills.md` (filled in full categorization guide — was placeholder)
- modified: `internal/ai/fallback.go` (set `Model: "regex"` in return)
- modified: `internal/ai/fallback_test.go` (added Model field assertion)
- modified: `internal/domain/job.go` (added `Model` to JobMeta, `AIProcessed` to JobEntry)
- modified: `internal/repository/job_postgres.go` (added `level` + `ai_processed_at IS NOT NULL` to SELECT + scan; updated `MarkAIProcessed` signature to use `meta.Model`)
- modified: `internal/service/job_service.go` (added `MarkAIProcessed` to jobStore interface; skip enriched rows; persist results)
- modified: `cmd/bot/main.go` (wire ChainExtractor(Groq, Regex) or regex-only)
- modified: `go.mod` / `go.sum` (added `github.com/sashabaranov/go-openai v1.41.2`)

### Decisions
- **JSON object mode, not strict schema**: `qwen/qwen3.6-27b` is a Groq preview model that does NOT support `strict: true` JSON schema mode. We use `response_format: { type: "json_object" }` (guarantees valid JSON, not schema adherence) + describe the schema in the prompt + validate on the client side + retry once on parse failure. If retry fails, ChainExtractor falls back to regex.
- **Lazy enrichment, not eager**: un-enriched rows are extracted on first `/jobs` query, not at scrape time. This keeps the scraper simple (Python just writes raw rows) and defers AI cost to when a user actually wants to see jobs. Each job is enriched exactly once (persisted in DB).
- **6-second throttle**: Groq free tier = 30 RPM / 8,000 TPM. With ~800 tokens/JD, the TPM limit binds first: 8,000 / 800 = 10 calls/min max. 6s spacing keeps us safely under that with headroom. First `/jobs` on 20 fresh jobs takes ~2 minutes — acceptable for a Discord interaction (15-min timeout).
- **`Model` field in JobMeta, not separate return value**: keeps the `Extractor` interface as `Extract(ctx, title, desc) (JobMeta, error)` — no signature change. The service passes `meta.Model` to `MarkAIProcessed`, which stores it in the `ai_model` DB column for audit.
- **`AIProcessed` flag from SQL, not separate query**: `ai_processed_at IS NOT NULL` in the SELECT returns a bool directly. No extra round-trip to check enrichment status.
- **Regex predicates KEPT in WHERE clause**: the `levelWordPatterns`/`levelYearPatterns`/`levelNoExpPatterns` maps are still present. They serve as a SQL-level pre-filter for un-enriched rows. Once all rows are enriched (steady state), these predicates match on the `level` column too. Defer deletion to a future cleanup slice — removing them now would break filtering during the transition window.
- **Graceful degradation**: when `GROQ_API_KEY` is not set, `main.go` wires regex-only (no ChainExtractor). The bot works identically to pre-Slice-D. This lets devs run the bot without a Groq account.

### Verification
- `go vet ./...`: clean
- `go test -race -cover ./...`: PASS
  - `internal/ai`: 69.4% coverage (chain.go 100%, groq.go 0% — needs integration tests)
  - `internal/domain`: [no statements] (pure data)
  - `internal/service`: 0% (needs mock jobStore — interface now has 2 methods)
- `go build ./cmd/bot`: PASS
- **Migrations must be applied before runtime**: `make migrate` (Slice C created the files, Slice D depends on the columns existing)

### Known issues
- `groq.go` has 0% test coverage — needs either a mock HTTP server or integration tests with a real `GROQ_API_KEY`. Deferred to future slice.
- `service/job_service.go` has 0% coverage — needs a mock `jobStore` implementing both `FetchRawJobs` and `MarkAIProcessed`. Deferred to future slice.
- Repository WHERE clause still uses regex predicates (kept intentionally — see Decisions above).
- `levelWordPatterns`/`levelYearPatterns`/`levelNoExpPatterns` maps still in `repository/job_postgres.go` (kept intentionally — see Decisions above).
- Branch consolidation still pending (carried over from Slice A).
- **User must run `make migrate` before starting the bot** — the SELECT now fetches `level` and `ai_processed_at` columns that don't exist until migrations are applied.

## [Slice E] — 2026-07-20

### Summary
CI/CD polish — the final slice. Fixed the Makefile `migrate` target to auto-load `.env` using Python's dotenv parser (root cause: Neon connection strings contain `&` which breaks shell sourcing). Suppressed cosmetic `Error 1` on `make bot` Ctrl+C. Added `staticcheck` linter to CI (caught and fixed 2 real issues on first run). Added Go module caching + pip caching to CI (~40s total speedup per run). Added `ruff` linter for Python. All 5 slices (A → E) now complete.

### Files touched
- modified: `Makefile` (rewrote `migrate` target to use Python dotenv; added `|| [ $$? -eq 1 ]` to `bot` target; added `lint-go` + `lint-python` targets; refactored `lint` to run both)
- modified: `.github/workflows/ci.yml` (added `cache: true` to setup-go; added `cache: pip` + `cache-dependency-path` to setup-python; added staticcheck step; added `pip install ruff` + `ruff check` steps)
- modified: `internal/bot/handlers.go` (removed redundant nil check before `len()` — S1009)
- modified: `internal/scraper/scraper.go` (added `//lint:ignore SA2001` comment for intentional empty critical section)

### Decisions
- **Python dotenv for `.env` loading in Makefile**: the root cause of the `make migrate` failure was that Neon connection strings contain `&` (in `sslmode=require&channel_binding=require`), which shell sourcing interprets as a background operator. Tried `set -a && . ./.env && set +a` first — failed because of the `&`. Switched to Python's `dotenv_values()` which parses `.env` the same way Go's `godotenv` does (handles `&`, `=`, quotes, comments). This is a dependency we already have (Python + dotenv are installed for the scraper), so no new deps.
- **`|| [ $$? -eq 1 ]` for bot target**: `go run` exits with code 1 when the program is killed by SIGINT (Ctrl+C). This is standard Go behavior, not a bug. The `|| [ $$? -eq 1 ]` makes make ignore that specific exit code. The bot's shutdown logic (`Shutting down...` + deferred `Stop()` calls) runs correctly regardless.
- **Staticcheck via `dominikh/staticcheck-action@v1`**: pre-built action, no need to `go install` in CI. Uses the Go from `setup-go` (via `install-go: false`). Catches issues `go vet` misses: ineffective assignments, unused code, API misuse, style issues.
- **Ruff for Python linting**: fast (Rust-based), catches unused imports + undefined names + style issues. Installed via `pip install ruff` (not in `requirements.txt` because it's a dev tool, not a runtime dep). Local `make lint-python` uses `|| true` so it doesn't fail if ruff isn't installed locally.
- **Go module caching via `cache: true`**: `actions/setup-go@v5` has built-in caching based on `go.sum` hash. No manual `actions/cache` step needed. Saves ~30s per CI run.
- **Pip caching via `cache: pip`**: `actions/setup-python@v5` has built-in caching. Uses `cache-dependency-path` to key on `requirements.txt`. Saves ~10s per CI run.
- **Kept `//lint:ignore SA2001` instead of rewriting scraper**: the empty critical section (`lock; unlock`) is an intentional pattern to wait for an in-flight `runWithLock` to complete. Rewriting it (e.g., with a `sync.WaitGroup`) would be a behavior change, not a refactor. The `//lint:ignore` comment documents the intent and suppresses the warning.

### Verification
- `go vet ./...`: clean
- `go test -race -cover ./...`: PASS (no coverage change)
- `go build ./cmd/bot`: PASS
- `staticcheck ./...`: clean (0 issues after fixes)
- `make migrate`: works without `export $(...)` workaround
- `make lint`: runs both `go vet` and `ruff check`
- `python -m py_compile scraper-python/JoblessYu.py`: PASS

### Known issues
- `groq.go` still 0% test coverage (carried over from Slice D — needs mock HTTP server).
- `service/job_service.go` still 0% coverage (carried over from Slice D — needs mock `jobStore`).
- Repository WHERE clause still uses regex predicates (kept intentionally until all rows AI-enriched).
- Branch consolidation still pending (carried over from Slice A).
- Local `make lint-python` requires `ruff` installed (`pip install ruff`) — uses `|| true` so it doesn't fail if missing.

## [Slice F] — 2026-07-20

### Summary
Read-path corrections. Fixed 3 critical bugs found by ChatGPT review: (1) enriched jobs lost tags/summary/salary/remote on 2nd query (SELECT didn't fetch them), (2) AI-normalized `job_type_normalized` was written but never read, (3) SQL regex pre-filter blocked AI from seeing relevant jobs. Added `AIEnabled` flag to `JobQuery` to skip the regex pre-filter when AI is active. Updated embed to display AI summary, salary, and remote badge.

### Files touched
- modified: `internal/domain/job.go` (added Summary, Salary, Remote, AIModel to JobEntry; added AIEnabled to JobQuery)
- modified: `internal/repository/job_postgres.go` (expanded SELECT + scan; COALESCE for job_type_normalized; skip regex pre-filter when AIEnabled)
- modified: `internal/bot/handlers.go` (set AIEnabled from config)
- modified: `internal/bot/embeds.go` (display summary, salary, remote in embed)

### Verification
- `go vet ./...`: clean
- `go test -race -cover ./...`: PASS
- `go build ./cmd/bot`: PASS
- `staticcheck ./...`: clean

## [Slice G] — 2026-07-20

### Summary
Package-by-feature refactor. Collapsed 4 packages (`domain/`, `service/`, `repository/`, `ai/`) into a single `internal/job/` package. 7 → 4 packages. Zero behavior change — pure code movement + package/import renames. All interfaces (`Extractor`, `jobStore`, `bot.jobService`) preserved. Motivated by Slice F findings: 2 of 3 critical bugs were structural symptoms of the modular split (code for one concept spread across 4 packages). Now adding a field to `JobEntry` touches 1 package instead of 4.

### Files touched
- moved: 16 files from `domain/`, `service/`, `repository/`, `ai/` → `internal/job/`
- modified: 15 Go files (package declarations renamed to `package job`; internal references cleaned)
- modified: 4 external files (imports updated: `cmd/bot/main.go`, `internal/bot/{bot,handlers,embeds}.go`)
- deleted: `service_test.go` (duplicate `TestLevelUnknownConstant` — compile error avoided)
- deleted: 4 empty directories (`domain/`, `service/`, `repository/`, `ai/`)

### Decisions
- **Conservative type names**: kept `JobEntry`, `JobMeta`, `JobQuery` (didn't rename to `Entry`, `Meta`, `Query`). Less churn, easier to review. Renaming can be a follow-up if redundancy bothers.
- **Parameter rename in embeds.go**: `job` → `j` to avoid shadowing the `job` package name within the function body.
- **Deleted duplicate test**: `TestLevelUnknownConstant` existed in both `levels_test.go` and `service_test.go`. Same package now, Go wouldn't compile. Kept the one in `levels_test.go` (more appropriate location).
- **Removed custom `min()`**: Go 1.26 has builtin `min()`. The custom one in `groq_test.go` shadowed it unnecessarily.
- **Interfaces preserved**: `Extractor` (3 impls), `jobStore` (testing seam), `bot.jobService` (testing seam). The refactor removed layer boundaries, not testing seams.

### Verification
- `go vet ./...`: clean
- `go test -race -cover ./...`: PASS (`internal/job` 41.0% — same tests, bigger denominator)
- `go build ./cmd/bot`: PASS
- `staticcheck ./...`: clean (0 issues)

### Known issues
- `store.go` and `service.go` have 0% coverage (need mock tests — future slice)
- `groq.go` has 0% coverage (needs mock HTTP server — future slice)
- Repository WHERE clause still uses regex predicates when AI is disabled (kept intentionally)
- Branch consolidation still pending (carried over from Slice A)

---

## Roadmap status — all slices complete

| Slice | Status | Summary |
|---|---|---|
| **A** — Cleanup & Hygiene | ✅ Complete | Dead code removed, `Python-Jobspy/` → `scraper-python/`, tooling added, CI workflow created, docs/AI/ structure |
| **B** — Modular Restructure | ✅ Complete | `ai.Extractor` interface, `domain.JobMeta`/`JobQuery` types, `service.jobStore` interface, regex moved to `ai/fallback.go` |
| **B.5** — Entanglement Cleanup | ✅ Complete | `domain/levels.go` stripped to pure data, `ApplyMeta` inlined in service, `bot.jobService` interface, `JobEntry.ID` field |
| **C** — Neon Backend | ✅ Complete | Migrations extracted, AI columns added, cron weekly, scraper params env-driven, `MarkAIProcessed` method |
| **D** — AI Core with Groq | ✅ Complete | `GroqExtractor` with throttling + retry, `ChainExtractor` (AI + regex fallback), `skills.md` categorization guide, lazy enrichment |
| **E** — CI/CD Polish | ✅ Complete | Makefile fixes, staticcheck + ruff linters, Go module + pip caching, 2 staticcheck issues fixed |
| **F** — Read-Path Corrections | ✅ Complete | Fixed 3 critical bugs (tags lost on reread, job_type_normalized ignored, regex pre-filter blocked AI); added summary/salary/remote to embed |
| **G** — Package-by-Feature | ✅ Complete | Collapsed 4 packages → 1 (`internal/job/`); 7 → 4 packages; zero behavior change; interfaces preserved |

**Next phases**:
- **Phase 2 — UI**: Discord Components V2 redesign + expertise dropdown integration
- **Phase 3 — Backend**: REST API extraction, saved jobs, Discord OAuth, web dashboard

---

## Phase 2 — Hybrid Scraper + Expertise + Bilingual AI — 2026-07-25

### Phase 2A: Expertise Logic
- New `internal/job/expertise.go` — 13 broad categories with keyword detection
- New `migrations/003_add_expertise.sql` — expertise column + index
- Modified: `entry.go` (Expertise field on JobEntry, JobMeta, JobQuery), `schema.go` (JSON schema), `skills.md` (AI prompt), `store.go` (SELECT/WHERE/UPDATE), `regex.go` (DetectExpertise in fallback), `service.go` (expertise filter)
- Detection: AI (Groq) primary + keyword (`DetectExpertise`) fallback with word-boundary regexes
- Tests: `expertise_test.go` — 14 cases including false-positive regression tests (html/ml, linux/ux, iraq/qa)

### Phase 2B: Scraper Migration (Firecrawl → jobspy + Colly)
- Restored `scraper-python/JoblessYu.py` from git history (jobspy for Indeed + LinkedIn)
- New `internal/job/colly_scraper.go` (Colly for ITViec — two-step: listing → individual pages)
- Deleted `internal/job/firecrawl.go`
- Removed `FirecrawlAPIKey` from `config.go` and `.env.example`
- ScraperManager: daily 5AM ICT cron calls jobspy subprocess + Colly + enricher
- ITViec freshness filter: parses "Posted X hours/days ago" from HTML, skips jobs > 24h old
- Remote badge capture: ITViec working model badge ("At office"/"Remote"/"Hybrid") captured directly from HTML, more reliable than AI guessing
- Python scraper: `results_wanted=20` (per-site), `hours_old=24`, `search_term="IT"`
- Python ON CONFLICT: resets all AI fields when description changes (matches Go UpsertJobs)

### Phase 2C: Bilingual skills.md
- Vietnamese title mapping: "Chuyên viên" → Junior, "Thực tập sinh" → Intern, "Trưởng phòng" → Senior
- Vietnamese experience phrases: "Ưu tiên kinh nghiệm" → Junior (not Fresher), "Không yêu cầu kinh nghiệm" → Fresher
- Skill-based level inference: cloud architecture → Junior minimum, team leadership → Senior
- Age limits explicitly ignored ("Dưới 35 tuổi" is not a level signal)
- Summary language matches JD language (Vietnamese JD → Vietnamese summary)
- Added Vietnamese salary format ("triệu"), remote phrase ("làm việc từ xa")
- Added OpenStack, VMware to tags (common in VN enterprise)

### Phase 2D: Audit Fix Pass (16 issues)
- **H5**: DeleteOldJobs SQL crash — `($1 || ' days')::interval` → `$1 * interval '1 day'`
- **H3+H4**: Expertise validation + default in `groq.go` — `IsValidExpertise()` + `DetectExpertise()` fallback
- **C1**: Regex fallback in enricher — only triggers on network errors (not 429/400/parse errors)
- **H1+H2**: DetectExpertise word-boundary regexes — `strings.Contains` → `\b` regexes (fixes "ml" matching "html")
- **M5**: Short-circuit Groq when API key empty — avoids 41s/job waste
- **M2**: Remote preservation in UpsertJobs — `jobs.remote OR EXCLUDED.remote` (once true, stays true)
- **M1**: Enricher error classification — explicit cases for 429, 400, 401/403, JSON parse, network
- **M3**: firstCall thread-safety — `bool` → `atomic.Bool`
- **M4**: Untruncated description for DetectExpertise — pass original (pre-truncation) to keyword fallback
- **M4**: Python raise on DB failure — `raise` after error print
- **M2+M3**: Deduplicate Python runner — main.go delegates to `ScraperManager.RunScrapeAndEnrich()`
- **M4**: resolveScriptPath dead code — `exec.LookPath` → `os.Stat`
- **L1-L6**: Dead code removal, stale comments, gofmt, slog standardization

### Phase 2E: Logging Cleanup
- 4-phase structured output: `[1/4]` `[2/4]` `[3/4]` `[4/4]`
- Python stdout captured silently (job count parsed via regex)
- Colly per-URL logs removed (was ×20 noise lines)
- Enricher per-job logs: `INFO` → `DEBUG` (batch summary stays `INFO`)
- All `log.*` calls in scraper.go → `slog.*` (structured logging)
- Deleted jobs: silent (count shown in batch summary)
- Warnings (429, failures, errors): still visible

### Phase 2F: Documentation
- README.md: full overhaul with current architecture, expertise table, tech stack
- CONTRIBUTING.md: recreated for 3-person team with full setup + architecture
- docs/AI/modules.md: updated deps (colly added, firecrawl removed), rate limits, expertise
- docs/AI/structure.md: updated file tree (new files added, deleted removed, descriptions updated)
- docs/AI/changelog.md: this entry

---

## Post-Slice H — Model Switch: qwen → llama-3.1-8b-instant — 2026-07-24

### Summary
Switched Groq model from `qwen/qwen3.6-27b` to `llama-3.1-8b-instant` after testing revealed the Qwen preview model fails JSON validation on every call (HTTP 400 "Failed to validate JSON"). Llama 3.1 8B is a production model with reliable `json_object` mode support. Updated default in `config.go`, `.env`, `.env.example`, and all code comments referencing the old model. Also updated `docs/AI/modules.md` Groq model section.

### Why
- `qwen/qwen3.6-27b` returns HTTP 400 on every call — `json_object` response format validation fails server-side
- Combined with JDs averaging ~1,142 tokens (4x the original ~300 estimate), this caused 100% fallback to regex
- `llama-3.1-8b-instant` is a Groq production model with reliable JSON output, 560 t/s speed, 131K context window

---

## [Slice J] — Hybrid DB Real-Time Sync, Card 2 Scrape Summary & Docker CI/CD Readiness — 2026-08-11

### Summary
Reverted Groq AI enrichment to single-job requests with an 18s delay to eliminate 413/429 rate limit errors. Streamlined Discord Hub to exactly 2 static cards (`🟢 ONLINE` and `🌅 SCRAPE SUMMARY`). Implemented hybrid real-time DB synchronization using Postgres `LISTEN/NOTIFY` triggers combined with a 30-second count change monitor and 500ms debouncing. Updated `Dockerfile` for multi-stage Go + Python runtime and added `/healthz` HTTP health probe.

### Files touched
- modified: `internal/job/enricher.go` (single-job LLM processing with 18s throttle)
- modified: `internal/bot/embeds.go` (renamed Card 2 title to `🌅 SCRAPE SUMMARY`)
- modified: `internal/bot/notifier.go` (updated card title matching)
- modified: `internal/scraper/scraper.go` (removed devlog webhook, updated card updates on pipeline finish)
- modified: `cmd/bot/main.go` (added `/healthz` HTTP server, updated `manualScrape` exit safeguard)
- modified: `internal/bot/bot.go` (implemented `CloseWithoutOffline` and Hybrid DB Sync monitor)
- modified: `internal/job/store.go` & `service.go` (added `ListenForJobChanges` method)
- created: `migrations/007_add_notify_trigger.sql` (Postgres `LISTEN/NOTIFY` event trigger)
- modified: `Dockerfile` (multi-stage Go 1.24 + Python 3.11 hybrid runtime)
- modified: `docs/AI/progress-log.md`
- modified: `docs/AI/changelog.md` (this file)

### Verification
- `go test -v -race -cover ./...`: PASS (100% test pass rate)
- `make scrape`: PASS (Updated Card 1 & Card 2 live in Discord)
- Migration 007 active on Neon DB.

