import hashlib
import json
import os
import re

import pandas as pd
from dotenv import load_dotenv
from jobspy import scrape_jobs

load_dotenv()

try:
    import psycopg
except ImportError:
    psycopg = None


def _to_nullable(value):
    return None if pd.isna(value) else value


def compute_dedup_hash(company, title, job_type):
    norm_comp = re.sub(r'[^\w]+', '', (company or '').lower()).strip()
    norm_title = re.sub(r'[^\w]+', '', (title or '').lower()).strip()
    norm_type = re.sub(r'[^\w]+', '', (job_type or '').lower()).strip()
    raw = f"{norm_comp}:{norm_title}:{norm_type}"
    return hashlib.md5(raw.encode('utf-8')).hexdigest()


def normalize_job_type(raw_type):
    if not raw_type:
        return None
    val = str(raw_type).lower().replace("-", "").replace(" ", "").replace("_", "")
    if "fulltime" in val:
        return "Full-time"
    elif "parttime" in val:
        return "Part-time"
    elif "contract" in val:
        return "Contract"
    elif "intern" in val:
        return "Internship"
    return str(raw_type)


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
        dedup_hash TEXT,
        alternate_urls JSONB DEFAULT '[]'::jsonb,
        UNIQUE (site, job_url)
    );
    """

    check_dedup_sql = """
    SELECT id, site, job_url, COALESCE(alternate_urls, '[]'::jsonb)
    FROM jobs
    WHERE dedup_hash = %s AND fetched_at >= NOW() - INTERVAL '7 days'
    LIMIT 1;
    """

    update_alt_sql = """
    UPDATE jobs SET alternate_urls = %s, fetched_at = NOW() WHERE id = %s;
    """

    insert_sql = """
    INSERT INTO jobs (job_id, site, job_url, title, company, location, job_type, description, dedup_hash)
    VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)
    ON CONFLICT (site, job_url)
    DO UPDATE SET
        job_id = EXCLUDED.job_id,
        title = EXCLUDED.title,
        company = EXCLUDED.company,
        location = EXCLUDED.location,
        job_type = EXCLUDED.job_type,
        description = EXCLUDED.description,
        fetched_at = NOW(),
        dedup_hash = EXCLUDED.dedup_hash,
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

    print("Connecting to NeonDB...", flush=True)
    saved_count = 0
    try:
        with psycopg.connect(database_url, connect_timeout=10) as conn:
            print("Connected to NeonDB. Creating table...", flush=True)
            with conn.cursor() as cur:
                cur.execute(create_table_sql)
                print("Table ready. Saving data...", flush=True)
                for _, row in jobs_df.iterrows():
                    raw_id = _to_nullable(row["id"])
                    site = row["site"]
                    job_url = row["job_url"]
                    title = _to_nullable(row["title"])
                    company = _to_nullable(row["company"])
                    location = _to_nullable(row["location"])
                    job_type = normalize_job_type(_to_nullable(row["job_type"]))
                    description = _to_nullable(row["description"])

                    dedup_hash = compute_dedup_hash(company, title, job_type)

                    cur.execute(check_dedup_sql, (dedup_hash,))
                    existing = cur.fetchone()
                    if existing:
                        existing_id, _existing_site, existing_url, alt_json = existing
                        alt_list = alt_json if isinstance(alt_json, list) else json.loads(alt_json)
                        already_present = (job_url == existing_url) or any(s.get("url") == job_url for s in alt_list)
                        if not already_present:
                            alt_list.append({"site": site, "url": job_url})
                            cur.execute(update_alt_sql, (json.dumps(alt_list), existing_id))
                        saved_count += 1
                        continue

                    cur.execute(insert_sql, (raw_id, site, job_url, title, company, location, job_type, description, dedup_hash))
                    saved_count += 1
            conn.commit()
        print(f"{saved_count} jobs upserted to Neon.", flush=True)
    except Exception as e:
        print("Error connecting to NeonDB:", e, flush=True)
        raise


def JobScan():
    print("JoblessYu is looking for jobs =w=")

    # Scrape jobs using JobSpy — daily, 20 per site (20 Indeed + 20 LinkedIn).
    # Uses ITViec canonical tech role search query to prevent non-IT noise.
    search_term = (
        '"software engineer" OR "software developer" OR "backend" OR "frontend" OR '
        '"fullstack" OR "data engineer" OR "data analyst" OR "AI engineer" OR '
        '"devops" OR "cloud engineer" OR "QA engineer" OR "tester" OR "SDET" OR '
        '"security engineer" OR "solution architect" OR "product owner" OR '
        '"scrum master" OR "mobile developer" OR "embedded engineer"'
    )
    jobs = scrape_jobs(
        site_name=["indeed", "linkedin"],
        search_term=search_term,
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

    # Filter out empty or invalid jobs (missing title or description < 50 chars)
    jobs = jobs.dropna(subset=["title", "description"])
    jobs["title"] = jobs["title"].astype(str).str.strip()
    jobs["description"] = jobs["description"].astype(str).str.strip()
    jobs = jobs[(jobs["title"].str.len() > 0) & (jobs["description"].str.len() >= 50)]

    # Filter out non-IT jobs (property management, civil/building engineers, sales/leasing, etc.)
    non_it_keywords = [
        "property manager", "leasing", "building engineer", "civil engineer",
        "formwork", "customer service", "sales director", "real estate", "accountant"
    ]
    exclusion_pattern = "|".join(non_it_keywords)
    jobs = jobs[~jobs["title"].str.lower().str.contains(exclusion_pattern, regex=True, na=False)]

    if jobs.empty:
        print("There are no valid IT jobs at the moment :c")
        return

    save_jobs_to_neon(jobs)


if __name__ == "__main__":
    JobScan()
