CREATE TABLE IF NOT EXISTS runtime_deployment_runs (
  id TEXT PRIMARY KEY,
  release_id TEXT NOT NULL DEFAULT '',
  application_id TEXT NOT NULL DEFAULT '',
  environment_id TEXT NOT NULL DEFAULT '',
  release_target_id TEXT NOT NULL DEFAULT '',
  target_type TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  reason TEXT NOT NULL DEFAULT '',
  correlation_id TEXT NOT NULL DEFAULT '',
  owner_id TEXT NOT NULL DEFAULT '',
  lease_expires_at TIMESTAMPTZ,
  attempt INTEGER NOT NULL DEFAULT 1,
  heartbeat_at TIMESTAMPTZ,
  manifest_snapshot_ref TEXT NOT NULL DEFAULT '',
  record JSONB NOT NULL,
  version INTEGER NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  started_at TIMESTAMPTZ,
  finished_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS runtime_deployment_host_groups (
  id TEXT PRIMARY KEY,
  environment_id TEXT NOT NULL DEFAULT '',
  name TEXT NOT NULL DEFAULT '',
  payload JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS runtime_deployment_logs (
  id TEXT PRIMARY KEY,
  deployment_run_id TEXT NOT NULL REFERENCES runtime_deployment_runs(id) ON DELETE CASCADE,
  stream TEXT NOT NULL,
  sequence BIGINT NOT NULL,
  content TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS runtime_deployment_events (
  id TEXT PRIMARY KEY,
  deployment_run_id TEXT NOT NULL REFERENCES runtime_deployment_runs(id) ON DELETE CASCADE,
  event_type TEXT NOT NULL,
  source TEXT NOT NULL,
  subject TEXT NOT NULL DEFAULT '',
  payload JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS runtime_deployment_audit_logs (
  id TEXT PRIMARY KEY,
  deployment_run_id TEXT NOT NULL REFERENCES runtime_deployment_runs(id) ON DELETE CASCADE,
  org_id TEXT NOT NULL DEFAULT '',
  actor_id TEXT NOT NULL DEFAULT '',
  action TEXT NOT NULL,
  subject TEXT NOT NULL,
  correlation_id TEXT NOT NULL DEFAULT '',
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS runtime_deployment_resources (
  deployment_run_id TEXT NOT NULL REFERENCES runtime_deployment_runs(id) ON DELETE CASCADE,
  inventory_type TEXT NOT NULL,
  resource_key TEXT NOT NULL,
  api_version TEXT NOT NULL DEFAULT '',
  kind TEXT NOT NULL DEFAULT '',
  namespace TEXT NOT NULL DEFAULT '',
  name TEXT NOT NULL DEFAULT '',
  desired_hash TEXT NOT NULL DEFAULT '',
  health TEXT NOT NULL DEFAULT '',
  payload JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (deployment_run_id, inventory_type, resource_key)
);

CREATE TABLE IF NOT EXISTS runtime_manifest_snapshots (
  id TEXT PRIMARY KEY,
  deployment_run_id TEXT NOT NULL REFERENCES runtime_deployment_runs(id) ON DELETE CASCADE,
  content_hash TEXT NOT NULL,
  document_count INTEGER NOT NULL DEFAULT 0,
  resource_count INTEGER NOT NULL DEFAULT 0,
  storage_ref TEXT NOT NULL DEFAULT '',
  payload JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS runtime_rollback_plans (
  deployment_run_id TEXT PRIMARY KEY REFERENCES runtime_deployment_runs(id) ON DELETE CASCADE,
  current_snapshot_id TEXT NOT NULL DEFAULT '',
  previous_snapshot_id TEXT NOT NULL DEFAULT '',
  target_type TEXT NOT NULL DEFAULT '',
  target_name TEXT NOT NULL DEFAULT '',
  strategy TEXT NOT NULL DEFAULT '',
  executable BOOLEAN NOT NULL DEFAULT false,
  payload JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS runtime_releases (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  version_name TEXT NOT NULL,
  application_id TEXT NOT NULL DEFAULT '',
  environment_id TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT '',
  record JSONB NOT NULL,
  version INTEGER NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS runtime_release_artifacts (
  id TEXT PRIMARY KEY,
  release_id TEXT NOT NULL REFERENCES runtime_releases(id) ON DELETE CASCADE,
  artifact_id TEXT NOT NULL DEFAULT '',
  name TEXT NOT NULL DEFAULT '',
  artifact_type TEXT NOT NULL DEFAULT '',
  reference TEXT NOT NULL DEFAULT '',
  digest TEXT NOT NULL DEFAULT '',
  digest_reference TEXT NOT NULL DEFAULT '',
  payload JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS runtime_release_events (
  id TEXT PRIMARY KEY,
  release_id TEXT NOT NULL REFERENCES runtime_releases(id) ON DELETE CASCADE,
  event_type TEXT NOT NULL,
  source TEXT NOT NULL,
  subject TEXT NOT NULL DEFAULT '',
  payload JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS runtime_release_audit_logs (
  id TEXT PRIMARY KEY,
  release_id TEXT NOT NULL REFERENCES runtime_releases(id) ON DELETE CASCADE,
  org_id TEXT NOT NULL DEFAULT '',
  actor_id TEXT NOT NULL DEFAULT '',
  action TEXT NOT NULL,
  subject TEXT NOT NULL,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS runtime_release_plans (
  id TEXT PRIMARY KEY,
  release_id TEXT NOT NULL DEFAULT '',
  environment_id TEXT NOT NULL DEFAULT '',
  strategy TEXT NOT NULL DEFAULT '',
  record JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS runtime_release_executions (
  id TEXT PRIMARY KEY,
  release_id TEXT NOT NULL DEFAULT '',
  environment_id TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  reason TEXT NOT NULL DEFAULT '',
  correlation_id TEXT NOT NULL DEFAULT '',
  owner_id TEXT NOT NULL DEFAULT '',
  lease_expires_at TIMESTAMPTZ,
  attempt INTEGER NOT NULL DEFAULT 1,
  heartbeat_at TIMESTAMPTZ,
  record JSONB NOT NULL,
  version INTEGER NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  started_at TIMESTAMPTZ,
  finished_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS runtime_release_execution_targets (
  execution_id TEXT NOT NULL REFERENCES runtime_release_executions(id) ON DELETE CASCADE,
  target_id TEXT NOT NULL,
  target_name TEXT NOT NULL DEFAULT '',
  target_type TEXT NOT NULL DEFAULT '',
  deployment_run_id TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  target_order INTEGER NOT NULL DEFAULT 0,
  payload JSONB NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (execution_id, target_id)
);

CREATE TABLE IF NOT EXISTS runtime_release_execution_events (
  id TEXT PRIMARY KEY,
  execution_id TEXT NOT NULL REFERENCES runtime_release_executions(id) ON DELETE CASCADE,
  event_type TEXT NOT NULL,
  source TEXT NOT NULL,
  subject TEXT NOT NULL DEFAULT '',
  payload JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS runtime_release_execution_audit_logs (
  id TEXT PRIMARY KEY,
  execution_id TEXT NOT NULL REFERENCES runtime_release_executions(id) ON DELETE CASCADE,
  org_id TEXT NOT NULL DEFAULT '',
  actor_id TEXT NOT NULL DEFAULT '',
  action TEXT NOT NULL,
  subject TEXT NOT NULL,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_runtime_deployment_runs_status_created_at ON runtime_deployment_runs(status, created_at);
CREATE INDEX IF NOT EXISTS idx_runtime_deployment_host_groups_environment_id ON runtime_deployment_host_groups(environment_id);
CREATE INDEX IF NOT EXISTS idx_runtime_deployment_runs_release_id ON runtime_deployment_runs(release_id);
CREATE INDEX IF NOT EXISTS idx_runtime_deployment_runs_environment_id ON runtime_deployment_runs(environment_id);
CREATE INDEX IF NOT EXISTS idx_runtime_deployment_runs_correlation_id ON runtime_deployment_runs(correlation_id);
CREATE INDEX IF NOT EXISTS idx_runtime_deployment_runs_lease ON runtime_deployment_runs(status, lease_expires_at);
CREATE INDEX IF NOT EXISTS idx_runtime_deployment_logs_run_sequence ON runtime_deployment_logs(deployment_run_id, sequence);
CREATE INDEX IF NOT EXISTS idx_runtime_deployment_events_run_created_at ON runtime_deployment_events(deployment_run_id, created_at);
CREATE INDEX IF NOT EXISTS idx_runtime_deployment_audit_subject_created_at ON runtime_deployment_audit_logs(subject, created_at);
CREATE INDEX IF NOT EXISTS idx_runtime_deployment_resources_run_type ON runtime_deployment_resources(deployment_run_id, inventory_type);
CREATE INDEX IF NOT EXISTS idx_runtime_releases_application_id ON runtime_releases(application_id);
CREATE INDEX IF NOT EXISTS idx_runtime_releases_environment_id ON runtime_releases(environment_id);
CREATE INDEX IF NOT EXISTS idx_runtime_release_artifacts_release_id ON runtime_release_artifacts(release_id);
CREATE INDEX IF NOT EXISTS idx_runtime_release_plans_release_created_at ON runtime_release_plans(release_id, created_at);
CREATE INDEX IF NOT EXISTS idx_runtime_release_executions_release_id ON runtime_release_executions(release_id);
CREATE INDEX IF NOT EXISTS idx_runtime_release_executions_status_created_at ON runtime_release_executions(status, created_at);
CREATE INDEX IF NOT EXISTS idx_runtime_release_executions_lease ON runtime_release_executions(status, lease_expires_at);
CREATE INDEX IF NOT EXISTS idx_runtime_release_execution_events_created_at ON runtime_release_execution_events(execution_id, created_at);
