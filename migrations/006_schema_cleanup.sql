-- Migration 006: Schema Cleanup & Data Storage Optimization
-- 1. Consolidate job_type_normalized into job_type
-- 2. Drop redundant columns (job_type_normalized, ai_model)
-- 3. Clean up EMPTY_STRING sentinel values in job_type and location

-- Copy normalized job_type values over raw job_type where available
UPDATE jobs
SET job_type = job_type_normalized
WHERE job_type_normalized IS NOT NULL
  AND job_type_normalized != ''
  AND job_type_normalized != 'EMPTY_STRING';

-- Replace 'EMPTY_STRING' sentinels with clean empty strings
UPDATE jobs SET job_type = '' WHERE job_type = 'EMPTY_STRING';
UPDATE jobs SET location = '' WHERE location = 'EMPTY_STRING';

-- Drop redundant columns to reclaim storage
ALTER TABLE jobs DROP COLUMN IF EXISTS job_type_normalized;
ALTER TABLE jobs DROP COLUMN IF EXISTS ai_model;
