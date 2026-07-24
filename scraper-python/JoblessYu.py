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
        fetched_at = NOW(),
        ai_processed_at = CASE WHEN jobs.description IS DISTINCT FROM EXCLUDED.description
                                THEN NULL ELSE jobs.ai_processed_at END,
        level = CASE WHEN jobs.description IS DISTINCT FROM EXCLUDED.description
                     THEN NULL ELSE jobs.level END,
        job_type_normalized = CASE WHEN jobs.description IS DISTINCT FROM EXCLUDED.description
                                   THEN NULL ELSE jobs.job_type_normalized END,
        tags = CASE WHEN jobs.description IS DISTINCT FROM EXCLUDED.description
                    THEN '{}'::jsonb ELSE jobs.tags END,
        summary = CASE WHEN jobs.description IS DISTINCT FROM EXCLUDED.description
                       THEN NULL ELSE jobs.summary END,
        salary = CASE WHEN jobs.description IS DISTINCT FROM EXCLUDED.description
                      THEN NULL ELSE jobs.salary END,
        remote = CASE WHEN jobs.description IS DISTINCT FROM EXCLUDED.description
                      THEN NULL ELSE jobs.remote END,
        ai_model = CASE WHEN jobs.description IS DISTINCT FROM EXCLUDED.description
                        THEN NULL ELSE jobs.ai_model END,
        expertise = CASE WHEN jobs.description IS DISTINCT FROM EXCLUDED.description
                         THEN NULL ELSE jobs.expertise END;
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
                print("Table ready. Saving data...", flush=True)
                cur.executemany(insert_sql, rows)
            conn.commit()
        print(f"{len(rows)} jobs upserted to Neon.", flush=True)
    except Exception as e:
        print("Error connecting to NeonDB:", e, flush=True)
        raise


def JobScan():
    print("JoblessYu is looking for jobs =w=")

    # Scrape jobs using JobSpy — daily, 20 per site (20 Indeed + 20 LinkedIn).
    jobs = scrape_jobs(
        site_name=["indeed", "linkedin"],
        search_term="IT",
        location="vietnam",
        results_wanted=20,
        hours_old=24,
        country_indeed='vietnam',
    )

    if jobs.empty:
        print("There are no jobs at the moment :c")
        return

    # Pandas DataFrame to JSON
    available_filters = ["id", "site", "job_url", "title",
                         "company", "location", "job_type", "description"]
    jobs = jobs[available_filters]
    save_jobs_to_neon(jobs)


if __name__ == "__main__":
    JobScan()
