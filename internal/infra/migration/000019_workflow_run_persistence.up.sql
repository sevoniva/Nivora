CREATE TABLE IF NOT EXISTS workflow_run_records (
  id TEXT PRIMARY KEY,
  workflow_id TEXT NOT NULL,
  workflow_plan_id TEXT NOT NULL,
  repository_id TEXT NOT NULL DEFAULT '',
  pipeline_run_id TEXT NOT NULL,
  pipeline_id TEXT NOT NULL,
  project_id TEXT NOT NULL DEFAULT '',
  environment_id TEXT NOT NULL DEFAULT '',
  ref TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  warnings JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_workflow_run_records_workflow_created
  ON workflow_run_records (workflow_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_workflow_run_records_plan_created
  ON workflow_run_records (workflow_plan_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_workflow_run_records_repository_created
  ON workflow_run_records (repository_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_workflow_run_records_project_status
  ON workflow_run_records (project_id, status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_workflow_run_records_pipeline_run
  ON workflow_run_records (pipeline_run_id);
