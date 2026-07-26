-- 004_add_dedup.sql
-- Add dedup_hash for cross-site job deduplication (LinkedIn, Indeed, ITViec)
-- Add alternate_urls for storing cross-posted links

ALTER TABLE jobs ADD COLUMN IF NOT EXISTS dedup_hash TEXT;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS alternate_urls JSONB DEFAULT '[]'::jsonb;

CREATE INDEX IF NOT EXISTS idx_jobs_dedup ON jobs (dedup_hash, fetched_at DESC);
