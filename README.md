# JoblessYu

JoblessYu is a Discord bot built in Go that scrapes IT job listings from Vietnamese and global job boards (ITViec, Indeed, LinkedIn), classifies them with Groq AI, and serves them via an interactive, 100% ephemeral (`"Only you can see this"`) slash command with rich filtering and pagination.

## Architecture

```
DAILY 5:00 AM ICT (automated cron)
    │
    ├── 1. SCRAPE
    │      ├── Python jobspy  → Indeed (40 jobs) + LinkedIn (40 jobs)
    │      ├── Go Colly        → ITViec (40 jobs, 24h freshness filter)
    │      └── Cross-Site Deduplication (7-day window → merges alternate URLs)
    │
    ├── 2. AI ENRICHMENT (Groq — llama-3.1-8b-instant)
    │      ├── Classifies: level, type, expertise, tags, salary, remote, summary
    │      ├── Regex fallback only when Groq is unreachable (network errors)
    │      ├── Jobs with empty descriptions (<50 chars) are deleted
    │      └── Bilingual: handles Vietnamese + English + mixed JDs
    │
    ├── 3. SERVING (/jobs command — 100% Ephemeral & Private)
    │      ├── Filter menu & job result cards are tagged "Only you can see this"
    │      ├── Interactive filter panel: Level, Location, Position, Job Type
    │      ├── Self-healing pagination cache with automatic DB recovery
    │      └── Multi-platform apply links (Indeed, LinkedIn, ITViec)
    │
    └── 4. WEEKLY CLEANUP (Monday 4:55 AM)
           └── Auto-delete jobs older than 30 days
```

## Expertise Categories (24 Categories)

The AI classifies each job into one of 24 canonical IT expertise categories based on ITViec's master taxonomy:

| Category | Value | Covers |
|---|---|---|
| IT Executive & Management | `management` | Project/product manager, CTO, CIO, CISO, CDO, VP, Director |
| Web Application Development | `web_dev` | Backend, frontend, fullstack, Node.js, React, Vue, Go |
| Mobile Application Development | `mobile_dev` | iOS, Android, Flutter, React Native, Swift, Kotlin |
| Core / Enterprise Systems | `enterprise` | ERP, CRM, SAP, Oracle, banking systems, Salesforce |
| Low-Code / No-Code Dev | `lowcode_nocode` | RPA, UiPath, Power Apps, Mendix, OutSystems |
| Technical Architecture | `architecture` | Solution, enterprise, and technical software architects |
| Blockchain Development | `blockchain` | Blockchain, smart contracts, Solidity, Web3, Ethereum |
| Game Development | `game_dev` | Unity, Unreal, Godot, game designer, game producer |
| Software Testing & QA | `testing_qa` | QA, automation tester, manual tester, SDET, PQA |
| Data Analytics & BI | `data_analytics` | Data analyst, BI analyst, BI developer, Power BI, Tableau |
| Data Engineering | `data_engineering` | Data engineer, Big Data, DataOps, MLOps, ETL, Spark |
| Data Science & AI / ML | `data_ai` | Machine learning engineer, AI researcher, data scientist |
| Data Management & Governance | `data_governance` | Data architect, data steward, database administrator (DBA) |
| Cloud Computing | `cloud` | Cloud engineer, AWS, Azure, GCP, cloud architect |
| Systems & Network Admin | `systems_network` | Network engineer, sysadmin, sysops, Linux, Windows server |
| DevOps & Site Reliability (SRE) | `devops_sre` | DevOps engineer, Kubernetes, Terraform, SRE, CI/CD |
| IT Support & Helpdesk | `support_helpdesk` | IT support, helpdesk, IT administrator, field support |
| Cybersecurity | `cybersecurity` | Security engineer, penetration tester, DevSecOps, SOC analyst |
| IT Compliance & Risk | `compliance_risk` | Compliance officer, GRC specialist, IT auditor, risk manager |
| Embedded, IoT & Robotics | `embedded_iot` | Embedded engineer, firmware, IoT, robotics, RTOS, microcontrollers |
| Product Management | `product_mgmt` | Product manager, product owner, product analyst |
| Project Management & Tech Comm | `project_mgmt` | Scrum master, agile coach, BrSE, business analyst, tech writer |
| Design & User Experience | `design_ux` | UX/UI designer, product designer, Figma, motion designer |
| IT Consulting & Sales | `consulting_sales` | IT consultant, pre-sales, technical account manager |

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
# On Linux/macOS:
python3 -m venv scraper-python/.venv
source scraper-python/.venv/bin/activate
pip install -r scraper-python/requirements.txt

# On Windows (PowerShell):
py -3.13 -m venv scraper-python\.venv
.\scraper-python\.venv\Scripts\Activate.ps1
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

1. **Scrape (JobSpy + Colly)**
   - Python JobSpy scrapes Indeed + LinkedIn (40 jobs each, posts from last 24 hours) using ITViec's canonical IT search query and non-IT title filters
   - Go Colly scrapes ITViec (40 jobs, filtered by "Posted X ago" ≤ 24h)
   - Cross-site deduplication computes `dedup_hash` (Company + Title + JobType) over a 7-day window, merging duplicate multi-platform URLs into `alternate_urls` JSONB

2. **AI Enrichment (Groq)**
   - Batch enrichment processes all un-enriched jobs (2-hour window)
   - Groq (`llama-3.1-8b-instant`) classifies: level (`Intern`, `Fresher`, `Junior`, `Senior`), type (`Full-time`, `Part-time`, `Contract`), expertise (24 categories), tags, salary, remote, summary
   - JDs truncated to 1,500 chars to stay within Groq's free-tier TPM limit (8,000 TPM)
   - 18s throttle between calls (~3.3 calls/min, ~5,050 TPM — 63% of limit)
   - Regex fallback only when Groq is unreachable (network errors only)
   - Jobs with empty descriptions (<50 chars) are deleted
   - Bilingual: handles Vietnamese titles ("Chuyên viên" → Junior), experience phrases, skill inference

3. **Serving (/jobs command — 100% Ephemeral & Private)**
   - Pure DB read — instant response, no AI calls during user interaction
   - Responses are tagged `"Only you can see this"` to preserve user privacy and keep channels clean
   - Filter dropdowns: Position (24 categories), Experience Level (`Intern`, `Fresher`, `Junior`, `Senior`, `All`), Location (`Ho Chi Minh`, `Ha Noi`, `All`), Job Type (`Full-time`, `Part-time`, `Contract`, `All`)
   - Interactive pagination with Prev/Next buttons and self-healing DB fallback recovery

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
