# JoblessYu

JoblessYu is a Discord bot built in Go that scrapes IT job listings from Vietnamese and global job boards, classifies them with AI, and serves them via an interactive slash command with pagination and filtering.

## Architecture

```
DAILY 5:00 AM ICT (automated cron)
    │
    ├── 1. SCRAPE
    │      ├── Python jobspy  → Indeed (20 jobs) + LinkedIn (20 jobs)
    │      ├── Go Colly        → ITViec (20 jobs, 24h freshness filter)
    │      └── Upsert to Neon Postgres
    │
    ├── 2. AI ENRICHMENT (Groq — llama-3.1-8b-instant)
    │      ├── Classifies: level, type, expertise, tags, salary, remote, summary
    │      ├── Regex fallback only when Groq is unreachable (network errors)
    │      ├── Jobs with empty descriptions (<50 chars) are deleted
    │      └── Bilingual: handles Vietnamese + English + mixed JDs
    │
    ├── 3. SERVING (/jobs command — pure DB read, instant)
    │      ├── Only AI-enriched jobs are shown
    │      ├── Interactive filter panel: level, location, position title
    │      └── Pagination with Prev/Next buttons
    │
    └── 4. WEEKLY CLEANUP (Monday 4:55 AM)
           └── Auto-delete jobs older than 30 days
```

## Expertise Filter (backend)

The AI classifies each job into one of 13 broad expertise categories:

| Category | Covers |
|---|---|
| Management & Executive | Project/product manager, CTO, CIO, director |
| Web Development | Backend, frontend, fullstack, HTML/CSS/JS |
| Mobile & Game Development | iOS, Android, Flutter, Unity |
| Enterprise Systems | ERP, CRM, SAP, RPA, banking systems |
| Architecture | Solution/enterprise/technical architect |
| Data & AI | Data analyst, ML engineer, data scientist |
| Cloud & DevOps | DevOps, cloud engineer, AWS, Kubernetes |
| Systems & Network | Network engineer, sysadmin, infrastructure |
| IT Support & Security | Helpdesk, security engineer, cybersecurity |
| Embedded & IoT | Firmware, microcontroller, robotics |
| Testing & QA | QA, tester, SDET, automation |
| Design & UX | UX/UI designer, product designer |
| Consulting & Sales | IT consultant, pre-sales, technical account |

## Prerequisites

- **Go 1.26** or later
- **Python 3.13** (for jobspy scraper)
- **PostgreSQL Database** ([Neon](https://neon.tech/) — free tier)
- **Groq API Key** ([GroqCloud](https://console.groq.com/) — free tier)
- **Discord Bot Token** ([Developer Portal](https://discord.com/developers/applications))

## Setup

### 1. Clone and install dependencies

```bash
git clone https://github.com/Itea-Lab/JoblessYu.git
cd JoblessYu
go mod download
```

### 2. Python scraper setup

```bash
python3 -m venv scraper-python/.venv
source scraper-python/.venv/bin/activate
pip install -r scraper-python/requirements.txt
```

### 3. Environment variables

Copy `.env.example` to `.env` and fill in your credentials:

```env
DISCORD_BOT_TOKEN=your_discord_bot_token
DISCORD_GUILD_ID=your_guild_id
DATABASE_URL=your_neon_postgres_connection_string
GROQ_API_KEY=your_groq_api_key
AI_MODEL=llama-3.1-8b-instant
JOB_RETENTION_DAYS=30
```

### 4. Database migrations

```bash
make migrate
```

### 5. Run the bot

```bash
make bot
```

## How it works

### Daily pipeline (5:00 AM ICT)

1. **Scrape (jobspy + Colly)**
   - Python jobspy scrapes Indeed + LinkedIn (20 jobs each, posts from last 24 hours)
   - Go Colly scrapes ITViec (20 jobs, filtered by "Posted X ago" ≤ 24h)
   - Total: ~60 jobs/day, upserted to Neon DB (deduplication via `ON CONFLICT`)

2. **AI Enrichment (Groq)**
   - Batch enrichment processes all un-enriched jobs (2-hour window)
   - Groq (`llama-3.1-8b-instant`) classifies: level, type, expertise, tags, salary, remote, summary
   - JDs truncated to 1,500 chars to stay within Groq's free-tier TPM limit (8,000 TPM)
   - 18s throttle between calls (~3.3 calls/min, ~5,050 TPM — 63% of limit)
   - Regex fallback only when Groq is unreachable (network errors only — not 429 or 400)
   - Jobs with empty descriptions (<50 chars) are deleted (useless for classification)
   - Bilingual: handles Vietnamese titles ("Chuyên viên" → Junior), experience phrases, skill inference

3. **Serving (/jobs command)**
   - Pure DB read — instant response, no AI calls during user interaction
   - Only AI-enriched jobs shown (`WHERE ai_processed_at IS NOT NULL`)
   - Interactive filter panel: level, location, position title
   - Pagination with Prev/Next buttons

4. **Weekly cleanup (Monday 4:55 AM)**
   - Jobs older than 30 days auto-deleted to keep DB lean (Neon free tier)

## Make commands

```bash
make bot       # Start Discord bot + cron scheduler
make scrape    # Full pipeline: jobspy (Indeed+LinkedIn) + Colly (ITViec) + AI enrichment
make enrich    # AI enrichment only (processes un-enriched DB jobs)
make test      # Run Go tests with race detector + coverage
make lint      # Run Go vet + Python ruff
make migrate   # Apply all SQL migrations to Neon DB
make clean     # Remove build artifacts (bot binary, jobs.json)
```

## Tech stack

| Component | Technology | Free tier |
|---|---|---|
| Bot | Go + discordgo | — |
| Scraper (Indeed + LinkedIn) | Python + jobspy | — |
| Scraper (ITViec) | Go + Colly | — |
| AI enrichment | Groq (llama-3.1-8b-instant) | 30 RPM, 8K TPM, 1K RPD |
| Database | Neon Postgres (serverless) | 0.5 GB storage |
| Scheduler | robfig/cron (Go) | — |
| CI | GitHub Actions (Go test + Python lint) | 2,000 min/month |
