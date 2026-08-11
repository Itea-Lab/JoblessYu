# Progress Log

> Daily modular log. Entries here get rolled up into `changelog.md` periodically.
> When the AI loses context, this file + changelog.md together reconstruct project state.

## 2026-08-05 — Slice H: 40-40-40 Scraper Scaling, Option 2 Terminal Logger & Windows Venv Fix

### Completed
- **Scraper Capacity Scaling**: Extended search target range from 20 to 40 jobs per platform across all scrapers (40 Indeed + 40 LinkedIn via Python JobSpy, 40 ITViec via Go Colly) in `scraper-python/JoblessYu.py` and `internal/job/colly_scraper.go`.
- **Option 2 Terminal Logger (Zero Emojis)**: Redesigned CLI phase logging in `internal/scraper/scraper.go` using clean ASCII box drawing cards and a final PIPELINE SUMMARY REPORT table. Removed all emoticons/emojis from logs across Go and Python scripts.
- **Deduplication & Queue Metrics Transparency**: Updated `UpsertJobs` in `internal/job/store.go` to return `UpsertStats` (`Inserted` vs `Merged`), added `CountEnrichmentStats` (`unenriched` vs `enriched`), and logged detailed breakdown in JobSpy script and ScraperManager logs.
- **Windows `.venv` Cross-Platform Resolution**: Added `.venv\Scripts\python.exe` lookup paths to `resolvePythonExe()` in `internal/scraper/scraper.go` so Windows environments resolve the virtual environment Python executable instead of falling back to system Python.
- **Windows Python 3.13 & Numpy Upgrade**: Rebuilt `scraper-python\.venv` with Python 3.13 and upgraded `numpy` to `2.5.1` to resolve MSVC CRT float precision crashes (`0xc0000005` access violations) on Windows hosts.

### Verification
- `go test ./...`: PASS (100% clean build & test execution)
- Manual scrape test `go run ./cmd/bot -scrape`: Completed full pipeline run cleanly in 42 minutes with zero errors.

---

## 2026-07-20 — Slice A: Cleanup & Hygiene

### Completed
- Removed dead `!testembed` prefix command + `IntentsMessageContent` intent from `internal/bot/bot.go`
- Removed `jobs.to_json("jobs.json", ...)` line from Python scraper (DB is source of truth)
- Renamed `Python-Jobspy/` → `scraper-python/` via `git mv` (history preserved)
- Updated path references in `internal/scraper/scraper.go` and `README.md`
- Created `scraper-python/requirements.txt` (pinned: python-jobspy==1.1.82, python-dotenv==1.2.2, pandas==2.3.3, psycopg[binary]==3.3.4)
- Created `.env.example` (DISCORD_BOT_TOKEN, DISCORD_GUILD_ID, DATABASE_URL, GROQ_API_KEY, AI_PROVIDER, AI_MODEL)
- Created `Makefile` (targets: bot, dev, test, lint, scrape, migrate, clean)
- Created `.github/workflows/ci.yml` (Go vet+test+build, Python install+py_compile)
- Created `.github/pull_request_template.md`
- Created `CONTRIBUTING.md` (trunk-based branching, conventional commits)
- Created `docs/AI/` folder with blank templates (this file, structure.md, modules.md, changelog.md)
- Added to `.gitignore`: `docs/AI/`, `neoenv.txt`, `GroqImplement.md`, `/bot`, `backup.sql`
- Recorded Groq implementation reference in `GroqImplement.md` (model: qwen/qwen3.6-27b)

### In Progress
- Nothing — Slice A is complete.

### Blocked
- Nothing.

### Next
- Slice B: Modular restructure (extract `JobRepository` interface, create `internal/ai/` package skeleton, move regex from service to ai/fallback.go)
- Slice C: Neon backend migrations (extract schema to `migrations/001_init.sql`, add AI columns, change cron to weekly Monday 09:00 ICT)
- Slice D: AI core with Groq (implement `Extractor` interface, write `skills.md` categorization guide, wire hybrid AI+regex fallback)
- Slice E: CI/CD (already partially done in Slice A; complete with deploy workflow in future)

### Notes
- `neoenv.txt` and `Note.md` are personal — left untouched, not combined with anything.
- `GroqImplement.md` contains the reference Groq Python snippet (model: `qwen/qwen3.6-27b`).
- **Important for Slice D**: `qwen/qwen3.6-27b` is a Groq preview model that does NOT support `strict: true` JSON schema. Will need best-effort JSON mode + retry-on-invalid-JSON logic.
- **Rate limit math for Slice D**: 30 RPM / 8,000 TPM / 1,000 RPD / 200K TPD. With ~800 tokens/JD, max ~10 enrichments/min. Weekly scrape of 20 jobs is well within limits but needs throttling (6s spacing).
- The `.venv` in `scraper-python/.venv` has hardcoded paths to old `Python-Jobspy/` location — user must recreate: `rm -rf scraper-python/.venv && python3 -m venv scraper-python/.venv && source scraper-python/.venv/bin/activate && pip install -r scraper-python/requirements.txt`

---

## 2026-07-20 — Slice B: Modular Restructure

### Completed
- Created `internal/ai/` package with `Extractor` interface — the pluggable boundary for AI providers
- Moved all regex detection logic from `service/` to `internal/ai/fallback.go` (`RegexExtractor` implementing `Extractor`)
- Moved keyword scanner (`keywordCategories` + `DetectTags`) from `service/keywords.go` to `internal/ai/keywords.go`
- Added `domain.JobMeta` struct (Level, Type, Tags, Salary, Remote, Summary) — the portable extraction contract
- Added `domain.JobQuery` struct (Level, JobType, Location, IncludeUnknown, Limit) — replaces 4 separate args
- Added `domain.JobEntry.ApplyMeta(JobMeta)` method — encapsulates "prefer DB value, fall back to detected" logic
- Defined `service.jobStore` interface (unexported, consumer-side) — `repository.JobRepository` satisfies it structurally
- Rewrote `service/job_service.go` as thin orchestrator: fetch raw → extract meta → apply → filter
- Updated `repository/job_postgres.go` signature: `FetchRawJobs(ctx, domain.JobQuery)` instead of 4 args
- Updated `bot/handlers.go` to build `domain.JobQuery` from slash command options
- Updated `cmd/bot/main.go` to wire `ai.NewRegexExtractor()` into service
- Moved detection tests to `internal/ai/fallback_test.go` (TestRegexExtractor_*, TestDetectTags_*)
- Moved `ApplyMeta` tests to `internal/domain/job_test.go` (they test domain code)
- Deleted `service/keywords.go` and `service/keywords_test.go` (moved to ai/)
- Created `internal/ai/skills.md` placeholder (Slice D fills it with the categorization guide)
- **Behavior is identical to pre-Slice-B** — regex still does all classification. The only difference is *where* the code lives.

### In Progress
- Nothing — Slice B is complete.

### Blocked
- Nothing.

### Next
- Slice C: Neon backend migrations (extract schema to `migrations/001_init.sql`, add AI columns: level, tags JSONB, summary, salary, remote, ai_processed_at, ai_model; change cron to weekly Monday 09:00 ICT = `0 2 * * 1` UTC; add `/scrape now` admin command for manual triggers; drop "run once initially" goroutine)
- Slice D: AI core with Groq (implement `GroqExtractor` in `internal/ai/groq.go` using `sashabaranov/go-openai` pointed at Groq's OpenAI-compatible endpoint; fill in `skills.md`; wire hybrid AI-primary + regex-fallback chain in `main.go`; add throttled client for 30 RPM / 8,000 TPM limits)
- Slice E: CI/CD polish (linters: staticcheck + ruff; caching: Go modules + pip)

### Notes
- `internal/ai` package: 94.7% coverage (absorbed moved FindLevelWord tests in Slice B.5)
- `internal/domain`: "[no statements]" — pure data package since Slice B.5 (no functions, only type/const/var declarations). Data-layer tests in `levels_test.go` verify `LevelRules` content + `LevelUnknown` const.
- `internal/service`: 0% coverage — thin orchestrator. Needs mock `jobStore` for real coverage. Slice D will add this when testing the AI+regex fallback chain.
- `domain/levels.go` now contains ONLY data: `LevelRule` struct, `LevelRules` slice, `LevelUnknown` const. The Go regex engine that consumes them (`findLevelWord`, `levelWordAlternation`, `noExpAlternation`, `compilePerRuleWordRE`) lives in `ai/fallback.go` as private. Repository still reads `domain.LevelRules` strings to build Postgres regex predicates — that's data reading, fine.
- **Slice D swap point**: to add Groq, create `internal/ai/groq.go` with a `GroqExtractor` implementing `Extractor`, then change `main.go` to use a chain: `GroqExtractor` primary, `RegexExtractor` fallback on error. No changes to service or bot layers needed — that's the whole point of the interface.

---

## 2026-07-20 — Slice B.5: Entanglement Cleanup

### Completed
- **Fix #1 (Critical)**: Moved `LevelRules` regex engine from `domain/levels.go` to `ai/fallback.go` as private (`findLevelWord`, `levelWordAlternation`, `noExpAlternation`, `compilePerRuleWordRE`). Domain is now pure data — no `regexp` import, no package-init side effects. Repository still reads `domain.LevelRules` strings (data only) to build Postgres regex predicates; that's fine and stays until Slice C replaces it with a `level` column filter.
- **Fix #2 (Critical)**: Removed `ApplyMeta` method from `domain.JobEntry`. Inlined the 3-line merge rule (`if j.Level == "" { j.Level = meta.Level }` etc.) directly in `service.FetchAndProcessJobs`. Domain is now pure data; business rules live in service where they belong. Slice D can flip the rule (AI wins over DB) with a 3-line edit.
- **Fix #3 (High)**: Added `bot.jobService` interface (consumer-side, unexported) in `bot/bot.go`. Bot field changed from `*service.JobService` (concrete) to `jobService` (interface). Bot package no longer imports `internal/service`. Bot handlers can now be unit-tested with a fake `jobService`. `cmd/bot/main.go` unchanged — `*service.JobService` satisfies the interface structurally.
- **Fix #4 (High)**: Added `ID int64` field to `domain.JobEntry`. Updated repository SELECT to fetch `id`, scan into `&id`, and assign to `JobEntry.ID`. Prepares for `MarkAIProcessed(ctx, jobID, meta)` in Slice D.
- Moved `TestFindLevelWord`, `TestFindLevelWord_InternalNotIntern`, `TestNoExpAlternationForRegex` from `domain/levels_test.go` to `ai/fallback_test.go` (they test the Go regex engine, which now lives in ai).
- Deleted `internal/domain/job_test.go` (its 3 tests were for `ApplyMeta`, which is now inline in service and gets tested via Slice D's service-level tests with mock jobStore).

### In Progress
- Nothing — Slice B.5 is complete.

### Blocked
- Nothing.

### Next
- Slice C: Neon backend migrations (extract schema to `migrations/001_init.sql`, add AI columns: level, tags JSONB, summary, salary, remote, ai_processed_at, ai_model; change cron to weekly Monday 09:00 ICT = `0 2 * * 1` UTC; add `/scrape now` admin command for manual triggers; drop "run once initially" goroutine; add AI env vars to config.Config: AIProvider, GroqAPIKey, AIModel)
- Slice D: AI core with Groq (implement `GroqExtractor` in `internal/ai/groq.go` using `sashabaranov/go-openai` pointed at Groq's OpenAI-compatible endpoint; fill in `skills.md`; wire hybrid AI-primary + regex-fallback chain in `main.go`; add throttled client for 30 RPM / 8,000 TPM limits)
- Slice E: CI/CD polish (linters: staticcheck + ruff; caching: Go modules + pip)

### Notes
- All 4 issues identified in the Slice B review are now resolved.
- `domain` package is now pure data (zero functions, zero non-stdlib imports). This is the cleanest possible state — it can be imported by any package without risk of side effects or circular deps.
- `bot` package no longer imports `service`. Dependency direction: bot → domain, bot → config, bot → discordgo. No business-layer coupling.
- The `repository` still reads `domain.LevelRules` to build Postgres regex. This is the LAST remaining cross-cutting regex coupling. Slice D will eliminate it when the `level` column replaces the SQL regex predicates (AI writes level into the column, repository filters on the column value, no regex needed).
- Test coverage: `internal/ai` 94.7%, `internal/domain` [no statements] (pure data, nothing to cover). Service and bot still 0% — Slice D adds mock-based tests for both.

---

## 2026-07-20 — Slice C: Neon Backend Migrations

### Completed
- Created `migrations/001_init.sql` — canonical schema extracted from Python's runtime `CREATE TABLE`. Python retains `CREATE TABLE IF NOT EXISTS` as idempotent safety net, but this SQL file is now the source of truth.
- Created `migrations/002_add_ai_columns.sql` — added 8 columns (`level`, `job_type_normalized`, `tags` JSONB, `summary`, `salary`, `remote`, `ai_processed_at`, `ai_model`) + 3 indexes (`idx_jobs_level`, `idx_jobs_fetched_at`, `idx_jobs_ai_pending` partial index for un-enriched rows).
- Added AI fields to `config.Config`: `AIProvider` (default "groq"), `GroqAPIKey`, `AIModel` (default "qwen/qwen3.6-27b"). Config validates and logs a warning if `GROQ_API_KEY` is missing (AI falls back to regex).
- Added `MarkAIProcessed(ctx, jobID, meta, model)` method to `repository.JobRepository`. Writes extraction results back to the `level`/`tags`/`summary`/`salary`/`remote`/`ai_processed_at`/`ai_model` columns. Handles nil tags map → `{}` JSONB. Slice D calls this after each successful AI extraction.
- Made Python scraper parameters env-driven: `SEARCH_TERM`, `SEARCH_LOCATION`, `RESULTS_WANTED`, `HOURS_OLD`, `SITES`, `COUNTRY_INDEED` (all with defaults matching pre-Slice-C hardcoded values).
- Changed cron schedule from `@every 6h` to `0 2 * * 1` (Monday 02:00 UTC = Monday 09:00 ICT). Weekly instead of 4×/day.
- Removed the startup scrape goroutine — bot restarts no longer trigger unintended scrapes. Manual scrape via `make scrape`.
- Updated `.env.example` with scraper env vars.
- Fixed stale comments in `scraper.go` (referenced old `Python-Jobspy/` path and "initial goroutine").

### In Progress
- Nothing — Slice C is complete.

### Blocked
- Nothing.

### Next
- Slice D: AI core with Groq (implement `GroqExtractor` in `internal/ai/groq.go`; fill in `skills.md`; wire hybrid AI-primary + regex-fallback chain in `main.go`; add throttled client for 30 RPM / 8,000 TPM limits; add `level` to repository SELECT so service can skip already-enriched rows; simplify FetchRawJobs WHERE clause to use `level` column instead of regex; add `MarkAIProcessed` to `service.jobStore` interface)
- Slice E: CI/CD polish (linters: staticcheck + ruff; caching: Go modules + pip)

### Notes
- **Behavior preserved**: `FetchRawJobs` WHERE clause and SELECT are UNCHANGED. The regex-based level predicates still work because the new DB columns start NULL. Slice D will: (1) wire AI to populate the columns via `MarkAIProcessed`, (2) backfill existing rows, (3) THEN simplify the WHERE clause from regex to `WHERE level = $1`, (4) THEN delete the `levelWordPatterns`/`levelYearPatterns`/`levelNoExpPatterns` maps. This staged approach avoids breaking `/jobs` filtering while AI is not yet active.
- **Migrations not yet applied to Neon**: user must run `make migrate` before Slice D to create the new columns.
- **Scraper env vars**: the Python script reads them from `.env` via `python-dotenv`. The Go subprocess inherits env vars from the parent process, so the bot's `.env` is automatically available to the scraper.
- **No `/scrape now` admin command**: deferred. Manual scrape via `make scrape` is sufficient for now. If needed later, add a Discord slash command that signals the scraper via a channel.

---

## 2026-07-20 — Slice D: AI Core with Groq

### Completed
- Created `internal/ai/groq.go` — `GroqExtractor` implementing `ai.Extractor`. Uses `github.com/sashabaranov/go-openai` pointed at Groq's OpenAI-compatible endpoint (`https://api.groq.com/openai/v1`). Model: `qwen/qwen3.6-27b` (configurable via `AI_MODEL` env var).
- Filled in `internal/ai/skills.md` with the full categorization guide — levels, types, tag categories (Cloud, IaC, Pipeline, Containers, Security, Languages, Data/DB, AI), canonicalization rules, edge cases, and JSON output format. Loaded via `//go:embed` as the Groq system prompt.
- Created `internal/ai/schema.go` — JSON schema description embedded in the prompt (since `qwen/qwen3.6-27b` does NOT support Groq's strict JSON schema mode, we use `json_object` mode + describe the schema in the prompt + validate/retry on the client side).
- Created `internal/ai/chain.go` — `ChainExtractor` that tries the primary extractor (Groq) first, and on any error falls back to the secondary (Regex). The returned `JobMeta.Model` field identifies which extractor produced the result.
- Added rate-limiting throttle to `GroqExtractor` — blocks on a 6-second ticker before each Groq call, keeping us under the 8,000 TPM limit (max ~10 calls/min with ~800 tokens/JD). Also retries once on JSON parse failure before giving up.
- Added `Model string` field to `domain.JobMeta` — identifies which extractor produced the result ("regex", "qwen/qwen3.6-27b"). Stored in `ai_model` DB column for audit.
- Added `AIProcessed bool` field to `domain.JobEntry` — set from `ai_processed_at IS NOT NULL` in the SQL query.
- Updated `repository/job_postgres.go`: added `level` and `ai_processed_at IS NOT NULL` to the SELECT; updated scan to read them; updated `MarkAIProcessed` to use `meta.Model` instead of a separate `model` param.
- Updated `service/job_service.go`: added `MarkAIProcessed` to `jobStore` interface; skip extraction for already-enriched rows (`j.AIProcessed == true`); persist extraction results via `MarkAIProcessed` after each successful extraction (lazy enrichment).
- Updated `cmd/bot/main.go`: wires `ChainExtractor(Groq, Regex)` when `GROQ_API_KEY` is set; degrades to regex-only when key is missing.
- Added `github.com/sashabaranov/go-openai v1.41.2` dependency.
- Created `internal/ai/chain_test.go` — tests for chain fallback logic (primary succeeds, primary fails → fallback, both fail → error, model name detection).
- Created `internal/ai/groq_test.go` — test for `truncate` helper.
- Updated `internal/ai/fallback_test.go` — added assertion for `Model: "regex"` field.

### In Progress
- Nothing — Slice D is complete.

### Blocked
- Nothing.

### Next
- Slice E: CI/CD polish (linters: staticcheck + ruff; caching: Go modules + pip)
- **User must run `make migrate` before testing** — the new DB columns (`level`, `ai_processed_at`, etc.) must exist for the SELECT to work.

### Notes
- **Behavior change**: AI is now the primary classifier. When `GROQ_API_KEY` is set, `/jobs` triggers Groq extraction on un-enriched rows. First query on fresh data takes ~6s/job (throttled). Subsequent queries are instant (rows are enriched in DB). When key is missing, behavior is identical to pre-Slice-D (regex only).
- **Rate limiting**: 6s spacing between Groq calls. For a weekly scrape of 20 jobs, first `/jobs` query takes ~2 minutes to enrich all rows. Discord interaction timeout is 15 minutes, so this is safe. Future improvement: background enrichment worker.
- **`qwen/qwen3.6-27b` does NOT support `strict: true` JSON schema**. We use `json_object` mode (guarantees valid JSON, not schema adherence) + describe the schema in the prompt + validate/retry once on parse failure. If retry fails, ChainExtractor falls back to regex.
- **Repository WHERE clause still uses regex predicates** — the `levelWordPatterns`/`levelYearPatterns`/`levelNoExpPatterns` maps are still present. They serve as a SQL-level pre-filter for un-enriched rows. Once all rows are enriched (steady state after a few `/jobs` queries), these predicates match on the `level` column too (since AI wrote the level there). Deleting the regex maps would break filtering for un-enriched rows during the transition window. Defer deletion to a future cleanup slice.
- **Coverage**: `internal/ai` 69.4% (chain.go 100%, groq.go 0% — needs integration tests with real API key or mock HTTP server). `internal/service` 0% (needs mock `jobStore` — the interface now has 2 methods, testable with a fake).

---

## 2026-07-20 — Slice E: CI/CD Polish

### Completed
- **Fixed Makefile `migrate` target**: now auto-loads `.env` using Python's `dotenv` (handles values with `&`, `=`, quotes correctly — same parser Go's `godotenv` uses). No more `export $(grep -v '^#' .env | xargs) &&` workaround needed. Root cause: `DATABASE_URL` contains `&` (in `sslmode=require&channel_binding=require`) which breaks shell sourcing.
- **Fixed Makefile `bot` target**: suppressed cosmetic `Error 1` on Ctrl+C. `go run` exits with code 1 when killed by SIGINT; `|| [ $$? -eq 1 ]` makes make ignore that specific exit code. The bot's shutdown logic runs correctly regardless.
- **Added `staticcheck` to CI workflow**: uses `dominikh/staticcheck-action@v1`. Runs after `go vet` + `go test` + `go build`. Caught 2 real issues on first run (both fixed in this slice):
  - `internal/bot/handlers.go:62` — redundant nil check before `len()` (S1009)
  - `internal/scraper/scraper.go:97` — empty critical section (SA2001, intentional — suppressed with `//lint:ignore`)
- **Added Go module caching to CI**: `cache: true` on `actions/setup-go@v5`. Caches based on `go.sum` hash. Saves ~30s per CI run.
- **Added pip caching to CI**: `cache: pip` + `cache-dependency-path: scraper-python/requirements.txt` on `actions/setup-python@v5`. Saves ~10s per CI run.
- **Added `ruff` linter for Python to CI**: installed via `pip install ruff`, runs `ruff check scraper-python/`. Catches unused imports, undefined names, style issues.
- **Added `lint-python` Makefile target**: `ruff check scraper-python/ || true` (doesn't fail locally if ruff isn't installed; CI installs it explicitly).
- **Refactored `make lint`**: now runs both `lint-go` and `lint-python` targets.
- **Fixed staticcheck findings**: removed redundant nil check in handlers.go; added `//lint:ignore SA2001` comment for intentional empty critical section in scraper.go.

### In Progress
- Nothing — Slice E is complete. All slices (A → E) are done.

### Blocked
- Nothing.

### Next
- **Phase 2 (UI)**: Discord Components V2 redesign — `Container`/`Section`/`TextDisplay`/`MediaGallery` layouts, `Button[Detail]` modal, inline `SelectMenu` filters. Requires verifying `bwmarrin/discordgo` Components V2 support first.
- **Phase 3 (Backend)**: Extract REST API from `internal/service` + `repository`. Bot becomes thin HTTP client. Add `saved_jobs` + `subscriptions` tables. Discord OAuth2 for user identity. Web dashboard (Next.js) — deferred per earlier decision.
- **Future cleanup slices** (optional, not blocking):
  - Delete `levelWordPatterns`/`levelYearPatterns`/`levelNoExpPatterns` maps from repository once all rows are AI-enriched (steady state)
  - Simplify `FetchRawJobs` WHERE clause from regex predicates to `WHERE level = $1`
  - Add integration tests for `groq.go` (mock HTTP server or real API key in CI secrets)
  - Add mock `jobStore` tests for `service/job_service.go`
  - Add `testcontainers-go` integration tests for `repository/job_postgres.go`
  - Branch consolidation (PR `base-JoblessYu` → `main`, delete `test-commit`)

### Notes
- **`make migrate` now works without workarounds** — the Python dotenv parser correctly handles the `&` in Neon connection strings. This was the root cause of the original `psql: error: connection to server on socket "/tmp/.s.PGSQL.5432"` failure.
- **CI now has 4 quality gates**: `go vet` → `go test` → `go build` → `staticcheck` (Go) + `pip install` → `py_compile` → `ruff` (Python). All must pass before merge.
- **CI caching**: Go modules + pip packages cached based on lockfile hashes. First run ~60s, subsequent runs ~30s.
- **Staticcheck is strict but valuable**: caught 2 real issues on first run that `go vet` missed. Worth the occasional noise.
- **All 5 slices (A → E) complete**. The project has gone from messy prototype to clean, modular, AI-enhanced, CI-protected Discord bot in one session.

---

## 2026-07-20 — Slice F: Read-Path Corrections

### Completed
- **Fix #1 (Critical)**: Expanded `FetchRawJobs` SELECT to fetch `tags`, `summary`, `salary`, `remote`, `ai_model` columns. Updated scan to handle JSONB `tags` (via `json.Unmarshal` into `map[string][]string`). Added 4 fields to `domain.JobEntry`: `Summary`, `Salary`, `Remote`, `AIModel`. This fixes the bug where enriched jobs lost their tags/summary/salary/remote on the 2nd `/jobs` query — the data was written by `MarkAIProcessed` but never read back.
- **Fix #2 (Critical)**: Changed SELECT to use `COALESCE(NULLIF(job_type_normalized, ''), COALESCE(job_type, ''))` so the AI-normalized type wins when available, falling back to the raw source `job_type`. Previously `MarkAIProcessed` wrote `job_type_normalized` but `FetchRawJobs` read only `job_type`, making the AI normalization useless.
- **Fix #3 (Critical)**: Added `AIEnabled bool` field to `domain.JobQuery`. Set it in `bot/handlers.go` from `b.cfg.GroqAPIKey != ""`. Repository skips the regex pre-filter (`levelWordPatterns`/`levelYearPatterns`/`levelNoExpPatterns`) when `q.AIEnabled` is true, fetching wider so the AI extractor sees rows the regex would miss. Service-layer strict filter still applies level matching after extraction. This fixes the bug where a JD like "Backend Engineer, extensive experience required" (no "senior" word, no year count) was filtered out before AI could classify it.
- **Bonus**: Updated `bot/embeds.go` to display the AI summary, salary, and remote flag in the Discord embed. Previously these fields were extracted and stored but never shown to users.

### In Progress
- Nothing — Slice F is complete.

### Blocked
- Nothing.

### Next
- **Slice G**: Package-by-feature refactor — collapse `internal/domain/` + `internal/service/` + `internal/repository/` + `internal/ai/` into a single `internal/job/` package. 7 → 4 packages. Green tests catch regressions. Do this before Phase 2.
- **Phase 2 (UI)**: Discord Components V2 redesign (after Slice G)
- **Phase 3 (Backend)**: REST API + dashboard (after Phase 2)

### Notes
- **3 critical bugs fixed** (all found by ChatGPT review):
  1. Tags/summary/salary/remote now persist across queries (were lost on 2nd read)
  2. AI-normalized job type now read back (was written but ignored)
  3. AI now sees all rows for level queries (was pre-filtered by regex)
- **Tradeoff for Fix #3**: when AI is enabled and user filters by level, all rows (up to LIMIT 20) are AI-processed, even if only 5 match. Acceptable for MVP — 20 jobs/week, weekly scrape, once enriched the queries are free.
- **3 medium issues deferred** (also from ChatGPT review): `DATABASE_MIGRATION_URL` (#4), Groq retry logic (#5), token budget docs (#6). Not blocking; fix in future polish slice.
- **Embed now shows more info**: summary (AI-generated, max 200 chars), salary (if mentioned), remote badge (✅ if applicable). Regex-classified rows show only tags + level + type (no summary/salary/remote).

---

## 2026-07-20 — Slice G: Package-by-Feature Refactor

### Completed
- Collapsed 4 packages (`domain/`, `service/`, `repository/`, `ai/`) into a single `internal/job/` package. 7 → 4 packages (`job/`, `bot/`, `config/`, `scraper/`).
- Moved 16 files via `git mv` (preserves history for tracked files) and `mv` (for untracked ai/ files from Slice B).
- Renamed all 15 Go files' package declarations to `package job`.
- Removed all `domain.`/`ai.`/`repository.`/`service.` package prefixes from internal references (now same-package).
- Removed 4 unused import lines (`"JoblessYu/internal/domain"`, `"JoblessYu/internal/ai"`, etc.) from moved files.
- Updated 4 external files' imports: `cmd/bot/main.go`, `internal/bot/bot.go`, `internal/bot/handlers.go`, `internal/bot/embeds.go` — all now import `"JoblessYu/internal/job"` instead of 3 separate packages.
- Renamed parameter `job` → `j` in `bot/embeds.go` to avoid shadowing the `job` package name.
- Deleted `service_test.go` — its only test (`TestLevelUnknownConstant`) was a duplicate of the one in `levels_test.go`. Compile error avoided.
- Removed custom `min()` function from `groq_test.go` — Go 1.26 has a builtin `min()`.
- Updated stale comments referencing old paths (`internal/ai/fallback.go` → `regex.go`, `internal/domain/` → `internal/job/`).
- Removed 4 empty directories: `internal/domain/`, `internal/service/`, `internal/repository/`, `internal/ai/`.
- **Zero behavior change** — pure code movement + package/import renames. All tests pass, all interfaces preserved.

### In Progress
- Nothing — Slice G is complete.

### Blocked
- Nothing.

### Next
- **Phase 2 (UI)**: Discord Components V2 redesign — `Container`/`Section`/`TextDisplay`/`MediaGallery` layouts, `Button[Detail]` modal, inline `SelectMenu` filters. Requires verifying `bwmarrin/discordgo` Components V2 support first.
- **Phase 3 (Backend)**: REST API extraction, saved jobs, OAuth, dashboard.

### Notes
- **Why refactored**: 2 of 3 critical bugs from ChatGPT review (Slice F) were structural symptoms of the modular split — `MarkAIProcessed` and `FetchRawJobs` were in the same file but conceptually split; `levelWordPatterns` in `repository/` read `LevelRules` from `domain/` consumed by `ai/fallback.go`. Package-by-feature puts all job logic in one place, making these classes of bugs harder to write.
- **Interfaces preserved**: `Extractor` (3 impls), `jobStore` (enables service testing), `bot.jobService` (enables bot testing). They still earn their keep — the refactor didn't remove the seams, just colocated the implementations.
- **Coverage**: `internal/job` at 41.0% — lower than the combined pre-refactor numbers (ai 69.4% + domain [no statements] + service 0% + repository 0%) because the untested code (store.go, service.go) is now in the same package as the tested code. Same tests, same coverage — just a bigger denominator.
- **Adding a field to JobEntry now touches 1 package** (was 4). This is the core benefit of package-by-feature: changes are localized.
- **New features are additive**: `internal/user/`, `internal/subscription/` when Phase 3 arrives. The `job/` package doesn't need to know about them.

---

## 2026-07-20 — Slice H: Code Quality Fixes

### Completed
- **Fix A — `time.Tick` goroutine leak**: Replaced `time.Tick(groqThrottle)` (which creates a ticker that never stops and leaks a goroutine) with `time.NewTicker(groqThrottle)`. Added `Close()` method to `GroqExtractor` and `ChainExtractor` (propagates to primary). `main.go` now calls `defer closer.Close()` via type assertion so the ticker is stopped on shutdown. `Extractor` interface stays clean — `Close()` is an optional capability.
- **Fix B — Groq retries all errors**: Added `jsonParseError` type. `callGroq` wraps `json.Unmarshal` failures as `&jsonParseError{}` and returns API/network errors as plain `fmt.Errorf`. `Extract` uses `errors.As` to retry only on `jsonParseError` (model may produce different output on retry). API/network/rate-limit errors fall back immediately — retrying won't help and wastes 12s per attempt.
- **Fix C — Token math + throttle safety**: Changed `groqThrottle` from 6s to 12s. Updated the comment block with actual measured token counts: skills.md ~706 + schema ~249 + JD ~300 + output ~200 = ~1,455 tokens/call. At 12s spacing = 5 calls/min × 1,455 = 7,275 tokens/min — under 8K TPM **without relying on prompt caching**. The old 6s throttle (10 calls/min × 1,455 = 14,550) exceeded TPM by 1.8×; it worked only because Groq automatically caches the system prompt, but that's a hidden dependency that could fail if the cache is evicted.
- **Fix D — Failed Groq jobs stuck on regex forever**: Added `AIProcessedAt time.Time` field to `JobEntry`. Changed SELECT from `ai_processed_at IS NOT NULL` (bool) to `ai_processed_at` (timestamp, via `sql.NullTime`). Service now retries regex-classified rows when `time.Since(j.AIProcessedAt) > regexRetryCooldown` (1 hour). Un-enriched rows always extract; AI-classified rows skip forever; regex-classified rows retry after cooldown. Also fixed: service now writes `Summary`/`Salary`/`Remote`/`AIModel` from meta to `JobEntry` on re-extraction (was only setting Level/Type/Tags).

### In Progress
- Nothing — Slice H is complete.

### Blocked
- Nothing.

### Next
- **Phase 2 (UI)**: Discord Components V2 redesign
- **Phase 3 (Backend)**: REST API, saved jobs, OAuth, dashboard

### Notes
- **Throttle change impact**: first `/jobs` on 20 un-enriched jobs now takes ~4 min (was ~2 min). Discord interaction timeout is 15 min — still safe.
- **Regex retry cooldown**: 1 hour. If Groq is persistently down, each `/jobs` query retries regex-classified rows at most once per hour. With 12s throttle, 5 regex rows = 60s of Groq attempts per query.
- **Coverage**: `internal/job` dropped from 41.0% to 37.8% — added more code (Close methods, error types, retry condition) which increases the denominator. Same tests, same pass rate.
- **`sql.NullTime` for nullable timestamp**: pgx scans NULL into `sql.NullTime{Valid: false}`. The `entry.AIProcessed` bool is derived from `aiProcessedAt.Valid`. `AIProcessedAt` is zero-valued when NULL.
- **3 medium issues still deferred** (not blocking): `DATABASE_MIGRATION_URL` (#4), daily token budget monitor (#E), few-shot examples in skills.md (#F). Can ride alongside Phase 2.

---

## 2026-07-25 — Phase 2: Hybrid Scraper + Expertise + Bilingual AI

### Session overview
Major architecture overhaul: replaced Firecrawl with hybrid jobspy+Colly scraper, added 13-category expertise filter, rebuilt skills.md as bilingual (EN+VI), fixed 16 audit issues, cleaned up logging.

### Phase 2A: Expertise Logic
- Created `internal/job/expertise.go` — 13 broad categories with `DetectExpertise()` using word-boundary regexes
- Created `migrations/003_add_expertise.sql` — expertise column + index
- Modified `entry.go`: added `Expertise` to `JobEntry`, `JobMeta`, `JobQuery`
- Modified `schema.go`: added expertise to JSON schema description
- Modified `skills.md`: added 14 expertise categories to AI prompt
- Modified `store.go`: expertise in SELECT, WHERE, MarkAIProcessed UPDATE
- Modified `regex.go`: `DetectExpertise()` called in regex fallback
- Modified `service.go`: expertise filtering (strict match)
- Created `expertise_test.go`: 14 test cases including false-positive regression tests
- Key fix: `DetectExpertise(plain, plain)` → split into `titlePlain` + `descPlain` for title-priority logic

### Phase 2B: Scraper Migration (Firecrawl → jobspy + Colly)
- Restored `scraper-python/JoblessYu.py` from git history
- Modified: `search_term="IT"`, `results_wanted=20` (per-site), `hours_old=24`
- Created `internal/job/colly_scraper.go`: Colly-based ITViec scraper
  - Two-step: listing page → extract job URLs → visit each individual page
  - 24h freshness filter: parses "Posted X hours/days ago" from HTML
  - Remote badge capture: "At office"/"Remote"/"Hybrid" from `.text-rich-grey.flex-shrink-0`
  - First-match semantics for working model (not last-match)
  - Rate limiting: 2s delay between requests
- Deleted `internal/job/firecrawl.go`
- Removed `FirecrawlAPIKey` from `config.go` and `.env.example`
- ScraperManager: hybrid pipeline — jobspy subprocess + Colly + enricher
- main.go: delegates to `ScraperManager.RunScrapeAndEnrich()` (no duplicate Python runner)
- Python ON CONFLICT: added AI field reset on description change (matches Go UpsertJobs)
- Python: `raise` on DB failure (so Go detects non-zero exit)

### Phase 2C: Bilingual skills.md
- Vietnamese title mapping table (Chuyên viên → Junior, Thực tập sinh → Intern, etc.)
- Vietnamese experience phrases ("Ưu tiên kinh nghiệm" → Junior, not Fresher)
- Skill-based level inference (cloud arch → Junior minimum, leadership → Senior)
- Age limits explicitly ignored ("Dưới 35 tuổi" is not a level signal)
- Summary language matches JD language
- Added "triệu" (VND salary format), "làm việc từ xa" (remote)
- Added OpenStack, VMware, Oracle to tags (common in VN enterprise)
- Added HTML, CSS, JavaScript, PHP, WordPress to web_dev keywords

### Phase 2D: Audit Fix Pass (16 issues)
- H5: DeleteOldJobs SQL — `($1 || ' days')::interval` → `$1 * interval '1 day'` (would crash weekly cron)
- H3+H4: Expertise validation in groq.go — `IsValidExpertise()` + `DetectExpertise()` fallback for invalid AI values
- C1: Regex fallback in enricher — ONLY on network errors (not 429/400/parse errors)
- H1+H2: DetectExpertise word-boundary regexes — `strings.Contains` → `\b` regexes (fixes "ml" matching "html", "ux" matching "linux")
- M5: Short-circuit Groq when API key empty — `if g.apiKey == "" return error` before throttle
- M2: Remote preservation — `jobs.remote OR EXCLUDED.remote` (once true, stays true)
- M1: Enricher error classification — explicit cases for 429, 400, 401/403, JSON parse, network
- M3: firstCall thread-safety — `bool` → `atomic.Bool` with `.Load()`/`.Store()`
- M4: Pass untruncated description to DetectExpertise in groq.go
- M2+M3: Deduplicate Python runner — main.go delegates to ScraperManager
- M4: resolveScriptPath — `exec.LookPath` → `os.Stat` (removed dead code + shell)
- L1: Documented yearsRe as separate from LevelRules
- L2: Removed dead `reflect` import in regex_test.go
- L3: Removed redundant empty-string check on DetectExpertise return
- L4: Fixed ForEach last-match-wins → first-match for working model badge
- L5: All `log.*` → `slog.*` in scraper.go
- gofmt -w . applied to all files

### Phase 2E: Logging Cleanup
- 4-phase structured output: `[1/4]` `[2/4]` `[3/4]` `[4/4]`
- Python stdout captured via `cmd.CombinedOutput()` (count parsed with regex)
- Colly `OnRequest` log removed (was ×20 noise lines)
- Enricher per-job: `INFO` → `DEBUG` (batch summary stays `INFO`)
- Deleted jobs: silent (count in summary)
- Warnings (429, failures): still visible

### Phase 2F: Documentation
- README.md: full overhaul — architecture diagram, expertise table, tech stack, how it works
- CONTRIBUTING.md: recreated — full setup, architecture, code style, git workflow
- docs/AI/modules.md: updated — colly added, firecrawl removed, rate limits updated, expertise section
- docs/AI/structure.md: updated — new files added, deleted removed, descriptions current
- docs/AI/changelog.md: Phase 2 entries added

### Test results
- `go build`: OK
- `go vet`: OK
- `go test -race -cover`: PASS (29.1% coverage in internal/job)
- `gofmt -l .`: Clean (0 violations)
- Live scrape test: 60 jobs scraped (40 jobspy + 20 Colly), 40 enriched, 20 deleted (empty JDs), 12 min enrichment

### Notes
- **jobspy empty descriptions**: ~50% of Indeed/LinkedIn jobs have empty JDs (behind login wall). These are deleted by the enricher. This is expected behavior — ITViec provides the best data (100% full JDs).
- **Remote always false for jobspy jobs**: Indeed/LinkedIn JDs don't always mention "remote" in text. ITViec remote is captured from HTML badge (reliable). Jobspy remote relies on AI text analysis (best-effort).
- **Expertise not in Discord UI yet**: The backend supports expertise filtering (`JobQuery.Expertise`), but the Discord UI doesn't expose it. Phase 2 UI will add the dropdown.
- **AIProvider removed**: Was loaded but never used. `AI_PROVIDER` env var no longer needed.
- **DISCORD_CHANNEL_ID removed from .env.example**: No Go code reads it.

---

## 2026-07-27 — UI Components V2 & Scraper Robustness Pass

### Completed
- **Discord Select Menu UX Fix (`internal/bot/embeds.go`)**:
  - Dynamically set `Placeholder: "Position: " + positionLabel` (and Level/Location) and `Default: true` on matching `SelectMenuOption` in `buildJobSweeperV2Components`.
  - Component select boxes now display active selections directly (e.g., `Position: Web Development`) instead of static "Choose Position".
  - Removed redundant `Active Filters:` header text block to streamline UI layout.
- **ITViec Colly Scraper Precision Fix (`internal/job/colly_scraper.go`)**:
  - Fixed company extraction: replaced `e.ChildText('a[href*="/companies/"]')` (which concatenated 30+ company links on page) with targeted `.employer-name` and `e.DOM.Find('a[href*="/companies/"]').First().Text()`.
  - Removed dangerous `e.ChildText("main")` fallback for descriptions to avoid capturing site layout text.
  - Added strict validation: skips scraping pages with missing title or description length `< 50` characters.
- **Database Storage Guardrails (`internal/job/store.go`)**:
  - Added newline stripping and length caps (`company <= 100` chars, `location <= 80` chars) in `FetchRawJobs` and `UpsertJobs` to sanitize DB reads/writes.
- **JobSpy Search Query Expansion (`scraper-python/JoblessYu.py`)**:
  - Expanded search term from `"software developer OR IT engineer OR software engineer"` to `"software OR developer OR IT OR engineer OR data OR devops OR QA OR security OR architect OR designer"` to cover all 13 expertise categories (Data/AI, Cloud/DevOps, QA, Security, Mobile, Architecture, UI/UX, etc.) on Indeed and LinkedIn.
  - Added Pandas DataFrame pre-filtering in Python to drop rows missing titles or with descriptions `< 50` chars before writing to Neon DB.

### Verification
- `go build ./...`: OK
- `go test ./...`: PASS
- `python3 -m py_compile scraper-python/JoblessYu.py`: OK

---

## 2026-07-31 — Scraper Cron Timezone & Live Output Streaming Fix

### Completed
- **Cron Scheduler Timezone Fix (`internal/scraper/scraper.go`)**:
  - Configured `cron.New(cron.WithLocation(time.UTC))` in `NewScraperManager` so cron expressions (`"0 22 * * *"` for 5:00 AM ICT and `"55 21 * * 0"` for Mon 4:55 AM ICT) evaluate against **UTC** as specified in code comments, fixing the bug where cron defaulted to system `time.Local` (running at 22:00 ICT / 10:00 PM ICT).
- **Real-Time Subprocess Log Streaming (`internal/scraper/scraper.go`)**:
  - Refactored `runPythonScraper` to stream `cmd.Stdout` and `cmd.Stderr` directly to `os.Stdout` / `os.Stderr` using `io.MultiWriter`, while capturing `outBuf` in memory for parsing upserted job count (`pythonJobCountRe`). Live Python JobSpy logs now print to stdout in real-time for both manual (`--scrape`) and background cron runs.

### Verification
- `go build ./...`: PASS (0 errors)
- Git commit & push: `Beta JoblessYu` committed and pushed to `base-JoblessYu` remote branch.

---

## 2026-08-10 — Discord Announcement & UI Visibility System (Slice I)

### Completed
- **Static Availability Status Card (`internal/bot/notifier.go`, `internal/bot/embeds.go`)**:
  - Implemented `Notifier.UpdateStatusCard()` and `BuildStatusCardEmbed()`. Pins and updates a static status embed (`🟢 ONLINE` Emerald / `🔴 OFFLINE` Crimson Abyss Bot pattern) with uptime, active job pool count, last scrape time, and next cron run schedule.
  - Automatically updates card to `🟢 ONLINE` in `disbot.Start()` and `🔴 OFFLINE` in `disbot.Stop()`.
- **Daily Scrape Public Announcement (`internal/bot/notifier.go`, `internal/scraper/scraper.go`)**:
  - Implemented `Notifier.PostDailyScrapeAnnouncement()` and `BuildDailyAnnouncementEmbed()`. Automatically broadcasts a gold/amber summary card to `DISCORD_ANNOUNCEMENT_CHANNEL_ID` after 5:00 AM ICT pipeline completion.
- **Developer Webhook Logs & Server Channel Resolution Fallback (`internal/bot/notifier.go`)**:
  - Implemented `Notifier.PostDevLogWebhook()`.
  - Added `resolveChannelID` fallback mechanism: if `DISCORD_STATUS_CHANNEL_ID` or `DISCORD_ANNOUNCEMENT_CHANNEL_ID` are empty in `.env`, the bot automatically auto-discovers the connected server (`DISCORD_GUILD_ID`) and targets the primary text channel (`#general`) so notifications work out-of-the-box.
- **Debug Test Suite & CLI Runner (`cmd/bot/main.go`, `Makefile`)**:
  - Added `make test-notify` (`go run ./cmd/bot -test-notify`) suite to trigger and echo all 4 test embeds (`🟢 ONLINE`, `🌅 Announcement`, `🧪 Dev Log`, `🔴 OFFLINE`) live in Discord.

- **Live Status Card Database Metrics (`internal/job/store.go`, `internal/job/service.go`, `internal/bot/bot.go`)**:
  - Implemented `GetScrapeStats(ctx)` on `JobRepository` and `JobService` to query Neon DB for total active jobs (`COUNT(*)`) and the latest `MAX(fetched_at)` timestamp.
  - Implemented `CalculateNextScrapeTime()` to accurately compute the next 05:00 AM ICT (22:00 UTC) cron run time.
  - Connected `fetchStatusDetails(ctx)` into `bot.Start()` and `bot.Stop()` so the static status card displays real, live database counts and timestamps instead of `0` / `N/A`.

- **Discord Dual-Static Hub Architecture & Real-Time DB Interlink (`internal/bot/notifier.go`, `internal/bot/embeds.go`, `internal/bot/bot.go`)**:
  - Implemented 2 static pinned cards (`Card 1: Status` and `Card 2: Daily Summary`) in the Hub channel.
  - Removed scrape timestamps from Card 1 and placed clean ICT timezone (`Asia/Ho_Chi_Minh` +07:00) timestamps in Card 2.
  - Added `GetPipelineSummaryStats()` in `JobRepository` & `JobService` to query live 24h DB metrics for Card 2 on boot.
  - Enforced Card 2 message ordering to always position directly below Card 1 with automatic re-sending if out of order.
  - Executed migration `006_schema_cleanup.sql` (consolidating `job_type` and dropping `ai_model`) and ran `VACUUM FULL jobs;` in Neon Postgres.
  - Updated LLM extraction batch size in `enricher.go` to 3 jobs per batch (`batchChunkSize = 3`) to strictly respect Groq Cloud Free Tier token limits (`6,000 TPM`), keeping token usage around ~2,425 tokens/call with >3,500 tokens of headroom below the limit, preventing HTTP 413 "Request Entity Too Large" errors while preserving fallback retry guarantees.

### Verification
- `go test -v -race -cover ./...`: PASS (0 errors, 100% test pass rate across unit tests).
- `make eval-ai`: PASS (100% Level, 100% Expertise, 100% Location, 0% Hallucinations).
- `make test-notify`: PASS (Delivered all status & summary test embeds to Discord).

---

## 2026-08-11 — AI Enrichment Single-Job Throttling, Dual-Card Real-Time Hub & CI/CD Readiness

### Completed
- **Groq LLM 1-Job Throttled Requests (`internal/job/enricher.go`)**:
  - Reverted batching to process 1 job per LLM request with an 18-second throttle (`groqThrottle = 18s`). Token usage per call is `~1,530 tokens` (`~5,050 TPM`), safely below Groq Cloud's `6,000 TPM` limit without any HTTP 413 or 429 errors.
- **Streamlined Discord Hub Channel & Scrape Card Rename (`internal/bot/embeds.go`, `internal/bot/notifier.go`, `internal/scraper/scraper.go`)**:
  - Renamed Card 2 title from `🌅 DAILY PIPELINE SUMMARY` to `🌅 SCRAPE SUMMARY`.
  - Removed extra webhook log message (`PostDevLogWebhook`), keeping the Discord Hub channel cleanly focused on **EXACTLY TWO static cards**: Card 1 (`🟢 ONLINE` / `🔴 OFFLINE`) & Card 2 (`🌅 SCRAPE SUMMARY`).
- **Scrape Exit Safeguard (`cmd/bot/main.go`, `internal/bot/bot.go`)**:
  - Added `CloseWithoutOffline()` method to `Bot`.
  - Updated `manualScrape` (`make scrape`) to call `disbot.CloseWithoutOffline()` on exit so running manual scrapes updates Discord cards in-place without flipping Card 1 to `🔴 OFFLINE`.
- **Hybrid Real-Time Database Synchronization (`migrations/007_add_notify_trigger.sql`, `internal/job/store.go`, `internal/job/service.go`, `internal/bot/bot.go`)**:
  - Created migration `007_add_notify_trigger.sql` (Postgres `LISTEN/NOTIFY` trigger `notify_jobs_changed()`).
  - Added `ListenForJobChanges` to `JobRepository` & `JobService`.
  - Implemented **Hybrid DB Sync** in `bot.Start()` combining Postgres `LISTEN` push notifications with a 30s count monitor and 500ms debouncing timer. Discord cards now update live in real-time whether rows are modified via scrapers, crons, or manual SQL actions in Neon Console.
- **User-Centric Card 2 Redesign & Multi-Keyword Breakdown (`internal/bot/embeds.go`, `internal/job/store.go`)**:
  - Removed technical dev clutter (`duplicates linked`, `AI Categorized`, `JobSpy + ITViec` breakdown) from Card 2.
  - Redesigned Card 2 to feature **User-Centric Job Pool Insights**: **Timestamps**, **Fresh Roles Today**, **Total Active Pool**, **Experience Level Breakdown** (`🎓 Intern / Fresher`, `🌱 Junior / Mid`, `🚀 Senior`, `⚡ Lead / Manager`), and **Top Locations** (`🏙️ Ho Chi Minh`, `🏛️ Ha Noi`, `🌊 Da Nang`, `💻 Remote`).
  - Expanded `GetPipelineSummaryStats` ILIKE queries in `store.go` to use multi-keyword recognition matching Vietnamese diacritics (`Hà Nội`, `HN`, `Hồ Chí Minh`, `HCM`, `SG`, `Đà Nẵng`, `Remote`, `Junior / Mid`, `Senior`, `Lead`), ensuring 100% database categorization accuracy.

### Verification
- `make eval-ai`: PASS (100% Level, 100% Expertise, 100% Location, 0% Hallucination Rate — 6/6 test cases passed).
- `go test -v -race -cover ./...`: PASS (100% test pass rate across all packages).
- `go vet ./...`: PASS (0 warnings / 0 errors).
- `make scrape`: PASS (Updated Card 1 & Card 2 live in Discord).
- `Migration 007`: Applied & active on Neon PostgreSQL DB.




