DROP INDEX IF EXISTS idx_job_runs_lease_expires_at;
DROP INDEX IF EXISTS idx_event_outbox_status_created_at;
DROP TABLE IF EXISTS event_outbox;

ALTER TABLE job_runs DROP COLUMN IF EXISTS max_retries;
ALTER TABLE job_runs DROP COLUMN IF EXISTS attempt;
ALTER TABLE job_runs DROP COLUMN IF EXISTS lease_expires_at;
ALTER TABLE pipeline_runs DROP COLUMN IF EXISTS cancel_requested;
