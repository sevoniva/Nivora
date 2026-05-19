CREATE TABLE IF NOT EXISTS runtime_pipeline_runs (
  id TEXT PRIMARY KEY,
  pipeline_id TEXT NOT NULL,
  status TEXT NOT NULL,
  correlation_id TEXT NOT NULL DEFAULT '',
  cancel_requested BOOLEAN NOT NULL DEFAULT false,
  version INTEGER NOT NULL DEFAULT 1,
  record JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  started_at TIMESTAMPTZ,
  finished_at TIMESTAMPTZ,
  failure_reason TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS runtime_job_runs (
  id TEXT PRIMARY KEY,
  pipeline_run_id TEXT NOT NULL REFERENCES runtime_pipeline_runs(id) ON DELETE CASCADE,
  stage_run_id TEXT NOT NULL,
  runner_id TEXT NOT NULL DEFAULT '',
  name TEXT NOT NULL,
  status TEXT NOT NULL,
  attempt INTEGER NOT NULL DEFAULT 1,
  max_retries INTEGER NOT NULL DEFAULT 0,
  lease_expires_at TIMESTAMPTZ,
  version INTEGER NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  started_at TIMESTAMPTZ,
  finished_at TIMESTAMPTZ,
  failure_reason TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS runtime_log_chunks (
  id TEXT PRIMARY KEY,
  pipeline_run_id TEXT NOT NULL REFERENCES runtime_pipeline_runs(id) ON DELETE CASCADE,
  stage_run_id TEXT NOT NULL DEFAULT '',
  job_run_id TEXT NOT NULL DEFAULT '',
  step_run_id TEXT NOT NULL DEFAULT '',
  stream TEXT NOT NULL,
  sequence BIGINT NOT NULL,
  content TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS runtime_events (
  id TEXT PRIMARY KEY,
  pipeline_run_id TEXT NOT NULL REFERENCES runtime_pipeline_runs(id) ON DELETE CASCADE,
  event_type TEXT NOT NULL,
  source TEXT NOT NULL,
  subject TEXT NOT NULL DEFAULT '',
  payload JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS runtime_audit_logs (
  id TEXT PRIMARY KEY,
  pipeline_run_id TEXT NOT NULL REFERENCES runtime_pipeline_runs(id) ON DELETE CASCADE,
  org_id TEXT NOT NULL DEFAULT '',
  actor_id TEXT NOT NULL DEFAULT '',
  action TEXT NOT NULL,
  subject TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS runtime_runners (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  group_id TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  labels JSONB NOT NULL DEFAULT '{}'::jsonb,
  executors JSONB NOT NULL DEFAULT '[]'::jsonb,
  last_heartbeat_at TIMESTAMPTZ,
  version INTEGER NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS runtime_event_outbox (
  id TEXT PRIMARY KEY,
  event_type TEXT NOT NULL,
  subject TEXT NOT NULL DEFAULT '',
  payload JSONB NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending',
  created_at TIMESTAMPTZ NOT NULL,
  published_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS idempotency_keys (
  scope TEXT NOT NULL,
  key TEXT NOT NULL,
  resource_type TEXT NOT NULL,
  resource_id TEXT NOT NULL,
  request_hash TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (scope, key)
);

CREATE INDEX IF NOT EXISTS idx_runtime_pipeline_runs_status_created_at ON runtime_pipeline_runs(status, created_at);
CREATE INDEX IF NOT EXISTS idx_runtime_pipeline_runs_correlation_id ON runtime_pipeline_runs(correlation_id);
CREATE INDEX IF NOT EXISTS idx_runtime_pipeline_runs_updated_at ON runtime_pipeline_runs(updated_at);
CREATE INDEX IF NOT EXISTS idx_runtime_job_runs_pipeline_run_id ON runtime_job_runs(pipeline_run_id);
CREATE INDEX IF NOT EXISTS idx_runtime_job_runs_status ON runtime_job_runs(status);
CREATE INDEX IF NOT EXISTS idx_runtime_job_runs_lease ON runtime_job_runs(status, lease_expires_at);
CREATE INDEX IF NOT EXISTS idx_runtime_log_chunks_run_sequence ON runtime_log_chunks(pipeline_run_id, sequence);
CREATE INDEX IF NOT EXISTS idx_runtime_log_chunks_job_sequence ON runtime_log_chunks(job_run_id, sequence);
CREATE INDEX IF NOT EXISTS idx_runtime_events_run_created_at ON runtime_events(pipeline_run_id, created_at);
CREATE INDEX IF NOT EXISTS idx_runtime_events_type ON runtime_events(event_type);
CREATE INDEX IF NOT EXISTS idx_runtime_audit_subject_created_at ON runtime_audit_logs(subject, created_at);
CREATE INDEX IF NOT EXISTS idx_runtime_runners_status ON runtime_runners(status);
CREATE INDEX IF NOT EXISTS idx_runtime_runners_last_heartbeat ON runtime_runners(last_heartbeat_at);
CREATE INDEX IF NOT EXISTS idx_runtime_outbox_status_created_at ON runtime_event_outbox(status, created_at);
