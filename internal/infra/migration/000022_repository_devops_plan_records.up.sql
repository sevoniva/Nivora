CREATE TABLE IF NOT EXISTS repository_devops_plan_records (
    id TEXT PRIMARY KEY,
    repository_id TEXT NOT NULL,
    snapshot_id TEXT NOT NULL DEFAULT '',
    project_id TEXT NOT NULL DEFAULT '',
    content_hash TEXT NOT NULL,
    plan JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_repository_devops_plan_records_repository_created
  ON repository_devops_plan_records(repository_id, created_at DESC, id);

CREATE INDEX IF NOT EXISTS idx_repository_devops_plan_records_snapshot_created
  ON repository_devops_plan_records(snapshot_id, created_at DESC, id);

CREATE INDEX IF NOT EXISTS idx_repository_devops_plan_records_project_created
  ON repository_devops_plan_records(project_id, created_at DESC, id);

CREATE INDEX IF NOT EXISTS idx_repository_devops_plan_records_content_hash
  ON repository_devops_plan_records(content_hash);
