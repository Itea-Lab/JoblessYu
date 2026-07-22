-- 001_init.sql
-- Extracted from scraper-python/JoblessYu.py (Slice C).
-- This is the canonical schema. The Python scraper retains its
-- CREATE TABLE IF NOT EXISTS as an idempotent safety net, but
-- this file is the source of truth applied via `make migrate`.

CREATE TABLE IF NOT EXISTS jobs (
    id          BIGSERIAL PRIMARY KEY,
    job_id      TEXT,
    site        TEXT NOT NULL,
    job_url     TEXT NOT NULL,
    title       TEXT,
    company     TEXT,
    location    TEXT,
    job_type    TEXT,
    description TEXT,
    fetched_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (site, job_url)
);
