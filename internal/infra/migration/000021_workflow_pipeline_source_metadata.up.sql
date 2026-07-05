ALTER TABLE workflow_plan_records
  ADD COLUMN IF NOT EXISTS repository_snapshot_id TEXT NOT NULL DEFAULT '';

ALTER TABLE workflow_run_records
  ADD COLUMN IF NOT EXISTS repository_snapshot_id TEXT NOT NULL DEFAULT '';

ALTER TABLE runtime_pipeline_runs
  ADD COLUMN IF NOT EXISTS workflow_id TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS workflow_plan_id TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS workflow_run_id TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS repository_id TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS repository_snapshot_id TEXT NOT NULL DEFAULT '';

ALTER TABLE runtime_job_runs
  ADD COLUMN IF NOT EXISTS workflow_job_id TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_workflow_plan_records_snapshot_created
  ON workflow_plan_records(repository_snapshot_id, created_at DESC, id);

CREATE INDEX IF NOT EXISTS idx_workflow_run_records_snapshot_created
  ON workflow_run_records(repository_snapshot_id, created_at DESC, id);

CREATE INDEX IF NOT EXISTS idx_runtime_pipeline_runs_workflow_created
  ON runtime_pipeline_runs(workflow_id, created_at DESC, id);

CREATE INDEX IF NOT EXISTS idx_runtime_pipeline_runs_workflow_run
  ON runtime_pipeline_runs(workflow_run_id);

CREATE INDEX IF NOT EXISTS idx_runtime_pipeline_runs_repository_snapshot
  ON runtime_pipeline_runs(repository_id, repository_snapshot_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_runtime_job_runs_workflow_job
  ON runtime_job_runs(workflow_job_id, pipeline_run_id);
