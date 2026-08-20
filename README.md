# JoblessYu

JoblessYu is a high-performance Discord bot built in Go that scrapes IT job listings from Vietnamese and global job boards (ITViec, Indeed, LinkedIn), classifies them with Groq AI (`openai/gpt-oss-20b`), and serves them via an interactive, 100% ephemeral (`"Only you can see this"`) slash command with multi-keyword search, modal page jumps, and dual real-time Discord Hub status cards.

## Architecture

```
DAILY 5:00 AM ICT (automated cron) / Manual Trigger (`make scrape`)
    │
    ├── 1. SCRAPE (Target: 30 ITViec + 30 Indeed + 30 LinkedIn)
    │      ├── Python jobspy  → Indeed (30 jobs) + LinkedIn (30 jobs)
    │      ├── Go Colly        → ITViec (30 jobs, 24h freshness filter)
    │      └── Cross-Site Deduplication (7-day window → merges alternate URLs)
    │
    ├── 2. AI ENRICHMENT (Groq — openai/gpt-oss-20b)
    │      ├── Classifies: level, type, expertise, tags, salary, remote, summary
    │      ├── 1-job request loop with 18s throttle (~5,050 TPM safely under 6,000 TPM limit)
    │      ├── Regex fallback only when Groq is unreachable (network errors)
    │      └── Bilingual: handles Vietnamese + English + mixed JDs
    │
    ├── 3. REAL-TIME DISCORD HUB (Dual Static Pinned Cards)
    │      ├── Card 1: System Status (🟢 ONLINE / 🔴 OFFLINE, version, active job pool)
    │      ├── Card 2: Scrape Summary (Timestamps, fresh roles, levels & location breakdown)
    │      └── Hybrid DB Sync: Postgres LISTEN jobs_changed + 30s monitor + 500ms debounce
    │
    ├── 4. SERVING (/jobs command — 100% Ephemeral & Private)
    │      ├── Filter menu & job result cards are tagged "Only you can see this"
    │      ├── Interactive filter panel: Level, Location, Position, Job Type
    │      ├── Modal Page Jump ("Go to Page") with self-healing DB fallback recovery
    │      └── Multi-platform apply links (Indeed, LinkedIn, ITViec)
    │
    └── 5. WEEKLY CLEANUP (Monday 4:55 AM)
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

- **Go 1.24** or later
- **Python 3.11+** (for jobspy scraper)
- **PostgreSQL Database** ([Neon](https://neon.tech/) — free tier serverless Postgres)
- **Groq API Key** ([GroqCloud](https://console.groq.com/) — free tier)
- **Discord Bot Token** ([Developer Portal](https://discord.com/developers/applications))

## Setup & Deployment

### 1. Local Environment Setup

```bash
git clone https://github.com/Itea-Lab/JoblessYu.git
cd JoblessYu
go mod download

# Set up Python virtual environment:
python3 -m venv scraper-python/.venv
source scraper-python/.venv/bin/activate
pip install -r scraper-python/requirements.txt
```

### 2. Environment Variables (`.env`)

```env
DISCORD_BOT_TOKEN=your_discord_bot_token
DISCORD_GUILD_ID=your_guild_id
DISCORD_CHANNEL_ID=your_hub_channel_id
DATABASE_URL=your_neon_postgres_connection_string
GROQ_API_KEY=your_groq_api_key
AI_MODEL=openai/gpt-oss-20b
JOB_RETENTION_DAYS=30
```

### 3. Database Migrations

```bash
make migrate
```

### 4. Running Docker Container (Cloud-Ready)

```bash
# Build multi-stage hybrid container:
docker build -t joblessyu .

# Run container with HTTP /healthz probe on port 8080:
docker run -d --env-file .env -p 8080:8080 joblessyu
```

## How It Works

### 1. Dual-Card Discord Hub Architecture
- **Card 1 (`🟢 ONLINE` / `🔴 OFFLINE`)**: Displays real-time bot lifecycle status, current version (`v1.2.0`), retention policy, and active job pool size. Automatically switches to `🔴 OFFLINE` when gracefully stopped.
- **Card 2 (`🌅 SCRAPE SUMMARY`)**: Displays user-centric job insights:
  - **Timestamps**: Last scrape time & next scheduled 05:00 AM ICT scrape.
  - **Fresh Roles Today**: Count of new job listings ingested today.
  - **Experience Level Breakdown**: `🎓 Intern / Fresher`, `🌱 Junior`, `🚀 Senior`, `⚡ Lead / Manager`.
  - **Top Locations**: `🏙️ Ho Chi Minh`, `🏛️ Ha Noi`, `🌊 Da Nang`, `💻 Remote`.

### 2. Real-Time Hybrid Database Synchronization
- Uses Postgres trigger `notify_jobs_changed()` (`LISTEN jobs_changed`) combined with a 30s fail-safe ticker and 500ms debouncer.
- Automatically updates Discord status cards whenever job records are added, updated, or deleted.

### 3. Interactive Ephemeral Job Search (`/jobs`)
- Pure DB query — 100% ephemeral (`"Only you can see this"`).
- Multi-select checkbox filtering by Level, Location, Position, and Job Type (compact fixed-height layout).
- Ephemeral pagination with `◀️ Prev`, `Page X/Y`, `Next ▶️`, and `🔢 Go to Page` modal jump.

## Make Commands

```bash
make bot          # Start Discord bot + cron scheduler + real-time DB listener
make dev          # Alias for `make bot`
make scrape       # Full pipeline: jobspy (Indeed+LinkedIn) + Colly (ITViec) + AI enrichment
make enrich       # AI enrichment only (processes un-enriched DB jobs)
make test         # Run Go tests with race detector + coverage
make test-notify  # Test Discord notifications & UI cards (Online/Offline status, Scrape Summary)
make eval-ai      # Run AI Evaluation & Hallucination Benchmark Suite
make lint         # Run Go vet + Python ruff static analysis
make lint-go      # Run Go vet static analysis
make lint-python  # Run ruff check on Python scraper scripts
make migrate      # Apply SQL migrations to Neon DB
make clean        # Remove build artifacts
```

## Tech Stack

| Component | Technology | Free Tier Capabilities |
|---|---|---|
| Bot Gateway | Go + discordgo | Dual static pinned cards + ephemeral components |
| Scrapers | Python JobSpy (Indeed, LinkedIn) + Go Colly (ITViec) | 30/30/30 target scrape distribution |
| AI Enrichment | Groq (`openai/gpt-oss-20b`) | 1-job request loop @ 18s delay (~5,050 TPM) |
| Database | Neon PostgreSQL (Serverless) | GIN Trigram indexes (`pg_trgm`) + LISTEN/NOTIFY |
| Container | Docker Multi-stage (Go 1.24 static + Python 3.11) | HTTP `/healthz` probe on port 8080 |
| CI | GitHub Actions | Automated Go test + static analysis |
