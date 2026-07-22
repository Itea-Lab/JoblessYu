# JoblessYu

JoblessYu is a Discord bot built in Go that scrapes job listings using a Python script and stores the data in a PostgreSQL database.

## Prerequisites

Before running the project, make sure you have the following installed:

- **Go 1.26.3** or later
- **Python 3.13**
- **PostgreSQL Database** ([NeonDB](https://neon.tech/))

## Setup instructions

### 1. Environment variables

Create a `.env` file in the root directory of the project and add the following configuration variables:

```env
DISCORD_BOT_TOKEN=your_discord_bot_token
DISCORD_GUILD_ID=your_discord_guild_id
DATABASE_URL=your_postgresql_connection_string
GROQ_API_KEY=your_groq_api_key
```

### 2. Install Go dependencies

From the root of the project, download the required Go modules:

```bash
go mod download
```

### 3. Install Python dependencies

The python scraper requires a few packages. It is recommended to use a virtual environment:

**Mac / Linux:**

```bash
python3 -m venv scraper-python/.venv
source scraper-python/.venv/bin/activate
pip install -r scraper-python/requirements.txt
```

**Windows (PowerShell):**

```powershell
python -m venv scraper-python\.venv
.\scraper-python\.venv\Scripts\Activate.ps1
pip install -r scraper-python\requirements.txt
```

### 4. Running the application

To start the bot and the scheduled scraper, run the following command from the root directory:

```bash
go run cmd/bot/main.go
```

### 5. How it works

- **Initialization:** The Go application initializes the Discord bot and connects to your PostgreSQL database.
- **Scraping schedule:** A background job scheduler is started, which automatically runs the Python scraper (`scraper-python/JoblessYu.py`) every 6 hours.
- **Python scraper:** The script scrapes "IT Support" jobs (from Indeed and LinkedIn) and upserts the records into your database.
- **Graceful shutdown:** You can stop the application safely at any time by pressing `CTRL+C` in your terminal.
