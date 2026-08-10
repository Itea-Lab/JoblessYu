-- Migration 007: Add Postgres LISTEN/NOTIFY trigger for real-time DB change detection
CREATE OR REPLACE FUNCTION notify_jobs_changed() RETURNS trigger AS $$
BEGIN
  PERFORM pg_notify('jobs_changed', TG_OP);
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_jobs_changed ON jobs;
CREATE TRIGGER trg_jobs_changed
AFTER INSERT OR UPDATE OR DELETE ON jobs
FOR EACH STATEMENT EXECUTE FUNCTION notify_jobs_changed();
