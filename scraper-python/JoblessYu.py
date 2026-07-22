import os
from jobspy import scrape_jobs
from dotenv import load_dotenv
import pandas as pd

load_dotenv()

try:
    import psycopg
except ImportError:
    psycopg = None


def _to_nullable(value):
    return None if pd.isna(value) else value


def save_jobs_to_neon(jobs_df):
    database_url = os.getenv("DATABASE_URL")
    if not database_url:
        print("DATABASE_URL not set. Skipping Neon save.")
        return

    if psycopg is None:
        print("psycopg is not installed. Run: pip install psycopg[binary]")
        return

    create_table_sql = """
    CREATE TABLE IF NOT EXISTS jobs (
        id BIGSERIAL PRIMARY KEY,
        job_id TEXT,
        site TEXT NOT NULL,
        job_url TEXT NOT NULL,
        title TEXT,
        company TEXT,
        location TEXT,
        job_type TEXT,
        description TEXT,
        fetched_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
        UNIQUE (site, job_url)
    );
    """

    alter_table_sql = "ALTER TABLE jobs ADD COLUMN IF NOT EXISTS description TEXT;"

    insert_sql = """
    INSERT INTO jobs (job_id, site, job_url, title, company, location, job_type, description)
    VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
    ON CONFLICT (site, job_url)
    DO UPDATE SET
        job_id = EXCLUDED.job_id,
        title = EXCLUDED.title,
        company = EXCLUDED.company,
        location = EXCLUDED.location,
        job_type = EXCLUDED.job_type,
        description = EXCLUDED.description,
        fetched_at = NOW();
    """

    rows = [
        (
            _to_nullable(row["id"]),
            row["site"],
            row["job_url"],
            _to_nullable(row["title"]),
            _to_nullable(row["company"]),
            _to_nullable(row["location"]),
            _to_nullable(row["job_type"]),
            _to_nullable(row["description"]),
        )
        for _, row in jobs_df.iterrows()
    ]

    print("Connecting to NeonDB...", flush=True)
    try:
        with psycopg.connect(database_url, connect_timeout=10) as conn:
            print("Connected to NeonDB. Creating table...", flush=True)
            with conn.cursor() as cur:
                cur.execute(create_table_sql)
                cur.execute(alter_table_sql)
                print("Table created. Saving data...", flush=True)
                cur.executemany(insert_sql, rows)
            conn.commit()
        print(f"{len(rows)} jobs upserted to Neon.", flush=True)
    except Exception as e:
        print("Error connecting to NeonDB:", e, flush=True)


def JobScan():
    print("JoblessYu is looking for jobs =w=")

    # Env-driven scraper params (defaults match pre-Slice-C hardcoded values).
    search_term = os.getenv("SEARCH_TERM", "IT Support")
    search_location = os.getenv("SEARCH_LOCATION", "vietnam")
    results_wanted = int(os.getenv("RESULTS_WANTED", "20"))
    hours_old = int(os.getenv("HOURS_OLD", str(24 * 7)))
    sites = [s.strip() for s in os.getenv("SITES", "indeed,linkedin").split(",")]
    country_indeed = os.getenv("COUNTRY_INDEED", "vietnam")

    print(f"  search_term={search_term!r} location={search_location!r} sites={sites} results={results_wanted}", flush=True)

    # Scrape jobs using JobSpy.
    jobs = scrape_jobs(
        site_name=sites,
        search_term=search_term,
        location=search_location,
        results_wanted=results_wanted,
        hours_old=hours_old,
        country_indeed=country_indeed,
    )

    if jobs.empty:
        print("There are no jobs at the moment :c")
        return

    # Pandas DataFrame to DB columns
    available_filters = ["id", "site", "job_url", "title",
                         "company", "location", "job_type", "description"]
    jobs = jobs[available_filters]
    print(f"{len(jobs)} jobs ready to upsert.")
    save_jobs_to_neon(jobs)


if __name__ == "__main__":
    JobScan()
