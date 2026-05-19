DROP INDEX IF EXISTS idx_event_outbox_retry;
DROP INDEX IF EXISTS idx_runtime_outbox_retry;
DROP INDEX IF EXISTS idx_deployment_runs_lease;
DROP INDEX IF EXISTS idx_runtime_pipeline_runs_cancel_requested;
DROP INDEX IF EXISTS idx_runtime_pipeline_runs_lease;

ALTER TABLE event_outbox DROP COLUMN IF EXISTS last_error;
ALTER TABLE event_outbox DROP COLUMN IF EXISTS next_attempt_at;
ALTER TABLE event_outbox DROP COLUMN IF EXISTS retry_count;

ALTER TABLE runtime_event_outbox DROP COLUMN IF EXISTS last_error;
ALTER TABLE runtime_event_outbox DROP COLUMN IF EXISTS next_attempt_at;
ALTER TABLE runtime_event_outbox DROP COLUMN IF EXISTS retry_count;

ALTER TABLE deployment_runs DROP COLUMN IF EXISTS heartbeat_at;
ALTER TABLE deployment_runs DROP COLUMN IF EXISTS attempt;
ALTER TABLE deployment_runs DROP COLUMN IF EXISTS lease_expires_at;
ALTER TABLE deployment_runs DROP COLUMN IF EXISTS owner_id;

ALTER TABLE runtime_pipeline_runs DROP COLUMN IF EXISTS heartbeat_at;
ALTER TABLE runtime_pipeline_runs DROP COLUMN IF EXISTS attempt;
ALTER TABLE runtime_pipeline_runs DROP COLUMN IF EXISTS lease_expires_at;
ALTER TABLE runtime_pipeline_runs DROP COLUMN IF EXISTS owner_id;
