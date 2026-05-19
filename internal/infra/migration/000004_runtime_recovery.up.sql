ALTER TABLE runtime_pipeline_runs ADD COLUMN IF NOT EXISTS owner_id TEXT NOT NULL DEFAULT '';
ALTER TABLE runtime_pipeline_runs ADD COLUMN IF NOT EXISTS lease_expires_at TIMESTAMPTZ;
ALTER TABLE runtime_pipeline_runs ADD COLUMN IF NOT EXISTS attempt INTEGER NOT NULL DEFAULT 1;
ALTER TABLE runtime_pipeline_runs ADD COLUMN IF NOT EXISTS heartbeat_at TIMESTAMPTZ;

ALTER TABLE deployment_runs ADD COLUMN IF NOT EXISTS owner_id TEXT NOT NULL DEFAULT '';
ALTER TABLE deployment_runs ADD COLUMN IF NOT EXISTS lease_expires_at TIMESTAMPTZ;
ALTER TABLE deployment_runs ADD COLUMN IF NOT EXISTS attempt INTEGER NOT NULL DEFAULT 1;
ALTER TABLE deployment_runs ADD COLUMN IF NOT EXISTS heartbeat_at TIMESTAMPTZ;

ALTER TABLE runtime_event_outbox ADD COLUMN IF NOT EXISTS retry_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE runtime_event_outbox ADD COLUMN IF NOT EXISTS next_attempt_at TIMESTAMPTZ;
ALTER TABLE runtime_event_outbox ADD COLUMN IF NOT EXISTS last_error TEXT NOT NULL DEFAULT '';

ALTER TABLE event_outbox ADD COLUMN IF NOT EXISTS retry_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE event_outbox ADD COLUMN IF NOT EXISTS next_attempt_at TIMESTAMPTZ;
ALTER TABLE event_outbox ADD COLUMN IF NOT EXISTS last_error TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_runtime_pipeline_runs_lease ON runtime_pipeline_runs(status, lease_expires_at);
CREATE INDEX IF NOT EXISTS idx_runtime_pipeline_runs_cancel_requested ON runtime_pipeline_runs(cancel_requested, updated_at);
CREATE INDEX IF NOT EXISTS idx_deployment_runs_lease ON deployment_runs(status, lease_expires_at);
CREATE INDEX IF NOT EXISTS idx_runtime_outbox_retry ON runtime_event_outbox(status, next_attempt_at, created_at);
CREATE INDEX IF NOT EXISTS idx_event_outbox_retry ON event_outbox(status, next_attempt_at, created_at);
