-- 002_add_ai_columns.sql
-- Adds columns for AI-enriched job metadata (Slice C).
-- Slice D will populate these via repository.MarkAIProcessed.
-- Until then, all columns are NULL/empty — the regex-based
-- detection in ai/fallback.go continues to work as before.

ALTER TABLE jobs
    ADD COLUMN IF NOT EXISTS level              TEXT,
    ADD COLUMN IF NOT EXISTS job_type_normalized TEXT,
    ADD COLUMN IF NOT EXISTS tags               JSONB DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS summary            TEXT,
    ADD COLUMN IF NOT EXISTS salary             TEXT,
    ADD COLUMN IF NOT EXISTS remote             BOOLEAN,
    ADD COLUMN IF NOT EXISTS ai_processed_at    TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS ai_model           TEXT;

-- Indexes for the most common filter paths.
CREATE INDEX IF NOT EXISTS idx_jobs_level       ON jobs(level);
CREATE INDEX IF NOT EXISTS idx_jobs_fetched_at  ON jobs(fetched_at DESC);

-- Partial index for finding un-enriched rows (Slice D will use this
-- to find jobs that need AI processing).
CREATE INDEX IF NOT EXISTS idx_jobs_ai_pending  ON jobs(ai_processed_at) WHERE ai_processed_at IS NULL;
