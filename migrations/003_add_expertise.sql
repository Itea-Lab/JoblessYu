-- 003_add_expertise.sql
-- Adds expertise column for broad job category classification.
-- Populated by the BatchEnricher via Groq AI (or regex keyword fallback).
-- Values: management, web_dev, mobile_game, enterprise, architecture,
--         data_ai, cloud_devops, systems_network, support_security,
--         embedded_iot, testing_qa, design_ux, consulting_sales, unknown

ALTER TABLE jobs ADD COLUMN IF NOT EXISTS expertise TEXT;

CREATE INDEX IF NOT EXISTS idx_jobs_expertise ON jobs(expertise);
