-- Migration 005: Add Trigram and Composite Index for Fast Discord Filtering
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Fast fuzzy title search index
CREATE INDEX IF NOT EXISTS idx_jobs_title_trgm ON jobs USING gin (title gin_trgm_ops);

-- Composite index for multi-column filtering (/jobs level, expertise, location)
CREATE INDEX IF NOT EXISTS idx_jobs_filter_perf ON jobs (level, expertise, fetched_at DESC);
