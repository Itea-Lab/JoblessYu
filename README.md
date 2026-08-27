# JoblessYu 🚀

<div align="center">

![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![Python Version](https://img.shields.io/badge/Python-3.11+-3776AB?style=for-the-badge&logo=python&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-Neon%20Serverless-4169E1?style=for-the-badge&logo=postgresql&logoColor=white)
![Discord API](https://img.shields.io/badge/Discord-Components%20V2-5865F2?style=for-the-badge&logo=discord&logoColor=white)
![Groq Cloud](https://img.shields.io/badge/AI%20Enrichment-Groq%20Llama%203.1-F55036?style=for-the-badge&logo=groq&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-Multi--Stage-2496ED?style=for-the-badge&logo=docker&logoColor=white)
![Terraform](https://img.shields.io/badge/IaC-Terraform-7B42BC?style=for-the-badge&logo=terraform&logoColor=white)
![License](https://img.shields.io/badge/License-MIT-green.svg?style=for-the-badge)

**High-performance, AI-enriched IT job aggregation engine and Discord bot engineered for Vietnam's tech talent.**

[Features](#-key-features) • [Architecture](#-architecture) • [Expertise Taxonomy](#-canonical-expertise-taxonomy-24-categories) • [Prerequisites](#-prerequisites) • [Quick Start](#-quick-start--local-setup) • [Deployment](#-deployment-options) • [Commands](#-make-commands-reference) • [Contributing](#-contributing)

</div>

---

## Overview

**JoblessYu** is a modern, enterprise-grade IT job aggregation platform and Discord bot written in **Go 1.24** and **Python 3.11+**. It automatically monitors major Vietnamese and international recruitment platforms (**ITViec**, **Indeed**, **LinkedIn**), extracts fresh job postings within a strict 24-hour freshness window, executes cross-site deduplication, enriches listing metadata with **Groq AI (`llama-3.1-8b-instant`)**, and delivers an interactive, **100% ephemeral (`"Only you can see this"`)** search experience inside Discord.

### Why JoblessYu?
- **100% Private & Ephemeral**: Filter panels and search results are only visible to the user who triggered the command, keeping Discord channels clutter-free and searches discreet.
- **Tailored to Vietnam Tech**: Canonical 24-category IT taxonomy derived from ITViec, filtering out non-IT noise and classifying tech stacks accurately.
- **Real-Time Dual-Card Status Hub**: Static pinned Discord status cards reflect live bot lifecycle states and daily scrape analytics driven by PostgreSQL `LISTEN / NOTIFY` events.
- **Hybrid Deployment Architecture**: Run as a containerized long-running daemon (with `/healthz` probes) or deploy serverlessly to **AWS Lambda (ARM64)** using **Terraform**.

---

## Architecture

```
DAILY 05:00 AM ICT (Automated Cron / Lambda EventBridge) / Manual Trigger (`make scrape`)
    │
    ├── 1. HYBRID SCRAPING PIPELINE (Target: ITViec + Indeed + LinkedIn)
    │      ├── Python JobSpy   ──► Indeed & LinkedIn (24h freshness filter)
    │      ├── Go Colly        ──► ITViec (Native fast crawling, 24h filter)
    │      └── Deduplication   ──► MD5(Company:Title:Type) over 7-day window (merges multi-platform URLs)
    │
    ├── 2. AI ENRICHMENT (Groq Cloud — llama-3.1-8b-instant)
    │      ├── Extracts: Level, Type, Expertise, Categorized Tech Tags, Salary, Remote, 1-Sentence Summary
    │      ├── Rate-limit safe: Single-job request loop with 18s throttling (~5,050 TPM < 6,000 TPM limit)
    │      ├── Multilingual: Native handling of Vietnamese, English, and mixed IT job descriptions
    │      └── Network Resilience: Fast regex extractor fallback on AI service outage
    │
    ├── 3. STORAGE & REAL-TIME EVENT BUS (Neon Serverless PostgreSQL)
    │      ├── Optimized search indexes: GIN Trigram indexes (`pg_trgm`) & compound B-tree indexes
    │      └── Database Triggers: `notify_jobs_changed()` broadcasts `jobs_changed` payload via PostgreSQL LISTEN/NOTIFY
    │
    ├── 4. DISCORD HUB & REAL-TIME SYNCHRONIZATION
    │      ├── Card 1 (System Status): Pinned card (🟢 ONLINE / 🔴 OFFLINE, version, active job pool)
    │      ├── Card 2 (Scrape Summary): Pinned card (Last/next scrape, fresh roles today, level & location breakdown)
    │      └── Hybrid Sync Engine: Postgres LISTEN + 30s fail-safe ticker + 500ms debouncer
    │
    ├── 5. SERVING (/jobs slash command — Discord Components V2)
    │      ├── 100% Ephemeral ("Only you can see this") interaction flow
    │      ├── Multi-Choice Dropdowns: Position (24 categories), Level (6 tiers), Location (4 areas), Job Type
    │      ├── Dynamic Pagination: `◀️ Prev`, `Page X/Y`, `Next ▶️`
    │      └── Modal Page Jump: `🔢 Go to Page` modal dialog with self-healing query fallback
    │
    └── 6. AUTOMATED RETENTION CLEANUP (Weekly Mondays 04:55 AM ICT)
           └── Automatically purges job postings older than configured retention period (Default: 30 days)
```

---

## Key Features

### 1. Interactive Ephemeral Job Sweeper (`/jobs`)
- Powered by Discord's **Components V2** interface.
- **Multi-Select Filters**: Select one or multiple criteria simultaneously:
  - **Positions**: Filter by any combination of 24 canonical IT categories.
  - **Levels**: `Intern`, `Fresher`, `Junior`, `Middle`, `Senior`, `Lead / Manager`.
  - **Locations**: `Ho Chi Minh`, `Ha Noi`, `Da Nang`, `Remote`.
  - **Job Types**: `Full-time`, `Part-time`, `Contract`.
- **Fast Pagination & Modal Jump**: Browse through hundreds of listings with real-time pagination and direct `🔢 Go to Page` input modals.
- **Multi-Platform Application Links**: Displays unified buttons for ITViec, LinkedIn, and Indeed when a role is discovered across multiple boards.

### 2. Groq AI Enrichment Engine
- Uses `llama-3.1-8b-instant` via Groq's high-speed inference cloud to parse complex, unstructured job descriptions.
- Groups tech stack tags into 8 distinct technical domains:
  - `Cloud`, `IaC`, `Pipeline / CI/CD`, `Containers`, `Security`, `Languages`, `Data / DB`, `AI / ML`.
- Extracts standardized salary figures (VND / USD) and remote work policies.
- Automatically generates concise, neutral 1-sentence summaries for rapid scanning on mobile and desktop.

### 3. Dual-Card Real-Time Discord Hub
- **Card 1 (`🟢 ONLINE` / `🔴 OFFLINE`)**: Live status monitor displaying service health, version, retention policy, and total active listings in Neon DB. Gracefully flips to `🔴 OFFLINE` on shutdown.
- **Card 2 (`🌅 SCRAPE SUMMARY`)**: Comprehensive job market summary displaying:
  - Last executed scrape and next scheduled 05:00 AM ICT scrape timestamp.
  - Fresh roles ingested today (+X new roles).
  - Categorized breakdown by experience levels and metropolitan hubs.
- **Postgres Event Synchronization**: Uses `LISTEN jobs_changed` to immediately re-render status cards whenever new jobs are inserted, updated, or cleaned up.

### 4. Cross-Site Deduplication
- Calculates a normalized MD5 hash: `hash(normalize(company) : normalize(title) : normalize(job_type))`.
- When the same opening is listed on both ITViec and LinkedIn/Indeed within a 7-day window, JoblessYu merges alternate URLs into the existing record rather than creating duplicates.

---

## Canonical Expertise Taxonomy (24 Categories)

JoblessYu categorizes every ingested job into one of 24 canonical IT domains based on ITViec's master taxonomy:

| Category | Key | Typical Roles & Technologies |
|---|---|---|
| **IT Executive & Management** | `management` | CTO, CIO, VP of Engineering, Director of IT, IT Manager |
| **Web Application Development** | `web_dev` | Fullstack, Frontend, Backend, Go, Node.js, React, Vue, Java, .NET |
| **Mobile Application Development** | `mobile_dev` | iOS, Android, Flutter, React Native, Swift, Kotlin |
| **Core / Enterprise Systems** | `enterprise` | ERP, CRM, SAP, Oracle, Salesforce, Core Banking |
| **Low-Code / No-Code Dev** | `lowcode_nocode` | RPA, UiPath, Power Apps, Mendix, OutSystems |
| **Technical Architecture** | `architecture` | Enterprise Architect, Solution Architect, Software Architect |
| **Blockchain Development** | `blockchain` | Smart Contracts, Solidity, Web3, Ethereum, Rust |
| **Game Development** | `game_dev` | Unity, Unreal Engine, Godot, Game Developer, 3D Graphics |
| **Software Testing & QA** | `testing_qa` | Manual Tester, Automation QA, SDET, Cypress, Playwright, Selenium |
| **Data Analytics & BI** | `data_analytics` | Data Analyst, BI Developer, Power BI, Tableau, Looker |
| **Data Engineering** | `data_engineering` | Data Engineer, Big Data, Spark, Airflow, Kafka, ETL/ELT |
| **Data Science & AI / ML** | `data_ai` | Machine Learning Engineer, AI Engineer, Data Scientist, GenAI, LLM |
| **Data Management & Governance** | `data_governance` | Data Architect, Data Steward, Database Administrator (DBA) |
| **Cloud Computing** | `cloud` | Cloud Engineer, Cloud Architect, AWS, GCP, Microsoft Azure |
| **Systems & Network Admin** | `systems_network` | SysAdmin, Network Engineer, Linux/Windows Server, IT Operations |
| **DevOps & Site Reliability (SRE)** | `devops_sre` | DevOps Engineer, SRE, Kubernetes, Docker, Terraform, CI/CD |
| **IT Support & Helpdesk** | `support_helpdesk` | IT Support, Helpdesk Specialist, Desktop Support, IT Admin |
| **Cybersecurity** | `cybersecurity` | Security Engineer, SOC Analyst, Pen Tester, DevSecOps |
| **IT Compliance & Risk** | `compliance_risk` | Compliance Officer, GRC Specialist, IT Auditor, ISO 27001 |
| **Embedded, IoT & Robotics** | `embedded_iot` | Embedded C/C++, Firmware, IoT, Robotics, RTOS, ARM |
| **Product Management** | `product_mgmt` | Product Manager (PM), Technical Product Manager, Product Owner |
| **Project Management & Tech Comm** | `project_mgmt` | Scrum Master, Agile Coach, Project Manager, BrSE, Technical Writer |
| **Design & User Experience** | `design_ux` | UI/UX Designer, Product Designer, Figma, Design Systems |
| **IT Consulting & Sales** | `consulting_sales` | IT Consultant, Pre-Sales Engineer, Technical Account Manager |

---

## Project Structure

```
JoblessYu/
├── cmd/
│   ├── bot/                # Main standalone daemon entrypoint (Bot + Cron + HTTP Healthz)
│   ├── lambda-bot/         # AWS Lambda handler for Discord interaction webhooks
│   ├── lambda-pipeline/    # AWS Lambda handler for EventBridge-scheduled scraping
│   └── migrate/            # CLI database migration runner
├── internal/
│   ├── bot/                # Discord bot lifecycle, interaction handlers, embeds & notifier
│   ├── config/             # Environment variables parser & fail-fast validation
│   ├── job/                # Repositories, Colly scraper, Groq AI enricher, eval benchmark
│   └── scraper/            # Multi-scraper orchestrator, Python subprocess manager, scheduler
├── migrations/             # SQL migration files (001_init to 007_notify_trigger)
├── scraper-python/         # Python JobSpy scraper module (Indeed & LinkedIn)
│   ├── JoblessYu.py        # Python scraper script with Neon direct upsert
│   └── requirements.txt    # Pinned Python dependencies
├── terraform/              # Infrastructure as Code (AWS Lambda, API Gateway, EventBridge)
│   ├── modules/            # Reusable Terraform modules
│   └── environments/prod/  # Production environment definition
├── .env.example            # Sample environment configuration template
├── Dockerfile              # Multi-stage container build (Static Go + Python 3.11)
├── Makefile                # Unified build, test, scrape, migration & deployment tasks
└── README.md               # Project documentation
---

## Prerequisites

Before setting up JoblessYu, ensure you have:

- **Go**: Version `1.24` or higher installed ([Download Go](https://go.dev/dl/))
- **Python**: Version `3.11+` installed ([Download Python](https://www.python.org/downloads/))
- **PostgreSQL Database**: Free serverless database from [Neon.tech](https://neon.tech/) (or standard PostgreSQL 14+)
- **Groq API Key**: Free tier API key from [GroqCloud Console](https://console.groq.com/)
- **Discord Bot Token & App**: Created via the [Discord Developer Portal](https://discord.com/developers/applications)

---

## 🚀 Quick Start & Local Setup

### 1. Clone the Repository

```bash
git clone https://github.com/Itea-Lab/JoblessYu.git
cd JoblessYu
```

### 2. Install Go & Python Dependencies

```bash
# Download Go modules
go mod download

# Set up Python virtual environment
python3 -m venv scraper-python/.venv

# Activate virtual environment (Linux/macOS)
source scraper-python/.venv/bin/activate

# Or on Windows (PowerShell):
# .\scraper-python\.venv\Scripts\Activate.ps1

# Install Python requirements
pip install -r scraper-python/requirements.txt
```

### 3. Configure Environment Variables (`.env`)

Copy `.env.example` to `.env` and fill in your credentials:

```bash
cp .env.example .env
```

```env
# ==========================================
# DISCORD CONFIGURATION
# ==========================================
DISCORD_BOT_TOKEN=your_discord_bot_token_here
DISCORD_GUILD_ID=your_guild_id_here          # Optional: Faster command propagation in dev
DISCORD_CHANNEL_ID=your_hub_channel_id_here  # Required: Target channel for status cards
DISCORD_LOG_WEBHOOK_URL=                     # Optional: Webhook URL for developer logs

# ==========================================
# DATABASE (NEON POSTGRESQL)
# ==========================================
DATABASE_URL=postgresql://user:pass@ep-xyz.neon.tech/joblessyu?sslmode=require

# ==========================================
# AI ENRICHMENT (GROQ)
# ==========================================
GROQ_API_KEY=gsk_your_groq_api_key_here
AI_MODEL=llama-3.1-8b-instant

# ==========================================
# LIFECYCLE & RETENTION SETTINGS
# ==========================================
JOB_RETENTION_DAYS=30
SCRAPE_TARGET_LIMIT=30
PORT=8080
```

### 4. Run Database Migrations

Apply all SQL migrations to your PostgreSQL database:

```bash
make migrate
```

### 5. Start the Bot

```bash
# Start bot gateway + cron scheduler + real-time status listener
make bot
```

Type `/jobs` in your Discord server to test the interactive Job Sweeper!

---

## Make Commands Reference

The included `Makefile` provides shorthand commands for common development and maintenance tasks:

| Command | Description |
|---|---|
| `make bot` / `make dev` | Start the Discord bot daemon, cron scheduler, and real-time DB listener. |
| `make scrape` | Run the full scraping pipeline once (JobSpy + Colly + AI enrichment) and exit. |
| `make enrich` | Run AI enrichment batch on all un-enriched listings in the database. |
| `make test` | Run complete Go test suite with `-race` detection and code coverage. |
| `make test-notify` | Verify Discord UI (Status Cards, Daily Announcement, Developer Webhooks). |
| `make eval-ai` | Run AI evaluation and hallucination benchmark suite against test dataset. |
| `make lint` | Run both Go static analysis (`go vet`) and Python linter (`ruff`). |
| `make lint-go` | Run Go vet linter on all packages. |
| `make lint-python` | Run Ruff linter on `scraper-python/`. |
| `make migrate` | Execute all sequential `.sql` migration files in `migrations/`. |
| `make build-lambda` | Compile optimized ARM64 Linux binaries for AWS Lambda into `bin/`. |
| `make tf-init` | Initialize Terraform working directory in `terraform/environments/prod`. |
| `make tf-plan` | Generate and inspect Terraform execution plan. |
| `make tf-apply` | Apply Terraform plan to deploy AWS Lambda infrastructure. |
| `make clean` | Clean up generated binaries, zip files, and build artifacts. |

---

## Deployment Options

### Option A: Docker Container (Recommended for Cloud Run / VPS)

JoblessYu includes a multi-stage `Dockerfile` that packages both the compiled static Go binary and the Python environment with JobSpy into a lightweight container.

```bash
# 1. Build the multi-stage image
docker build -t joblessyu:latest .

# 2. Run the container with environment variables and healthcheck probe
docker run -d \
  --name joblessyu \
  --restart unless-stopped \
  --env-file .env \
  -p 8080:8080 \
  joblessyu:latest
```

The container exposes a lightweight HTTP `/healthz` health check on port `8080` for uptime monitoring, Docker Compose, or Kubernetes liveness probes.

---

### Option B: AWS Serverless with Terraform (Zero Idle Cost)

JoblessYu can run 100% serverlessly on AWS using two ARM64 Lambda functions:
1. `lambda-bot`: Handles incoming Discord Interactions via HTTP API Gateway webhook.
2. `lambda-pipeline`: Scheduled via AWS EventBridge Cron (Daily at 22:00 UTC / 05:00 AM ICT).

```bash
# 1. Build Lambda packages
make build-lambda

# 2. Initialize and apply Terraform
make tf-init
make tf-plan
make tf-apply
```

---

## AI Evaluation & Benchmarking

JoblessYu includes a built-in benchmark harness to evaluate Groq LLM accuracy and detect hallucinations in classification, level mapping, and salary extraction:

```bash
make eval-ai
```

This tests the prompt with ground-truth Vietnamese and English job descriptions and outputs a detailed score report with precision metrics across all categories.

---

## Contributing

Contributions are welcome! Please follow these guidelines:

1. Read [`CONTRIBUTING.md`](CONTRIBUTING.md) for commit message conventions and branch workflow.
2. Ensure all tests pass before submitting a pull request:
   ```bash
   make test
   make lint
   ```
3. Open a Pull Request with a clear description of the feature or bug fix.

---

## License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.

<div align="center">
Made with ❤️ for the ITeaLab.
</div>