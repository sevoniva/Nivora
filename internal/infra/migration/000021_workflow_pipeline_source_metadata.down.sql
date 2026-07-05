DROP INDEX IF EXISTS idx_runtime_job_runs_workflow_job;
DROP INDEX IF EXISTS idx_runtime_pipeline_runs_repository_snapshot;
DROP INDEX IF EXISTS idx_runtime_pipeline_runs_workflow_run;
DROP INDEX IF EXISTS idx_runtime_pipeline_runs_workflow_created;
DROP INDEX IF EXISTS idx_workflow_run_records_snapshot_created;
DROP INDEX IF EXISTS idx_workflow_plan_records_snapshot_created;

ALTER TABLE runtime_job_runs
  DROP COLUMN IF EXISTS workflow_job_id;

ALTER TABLE runtime_pipeline_runs
  DROP COLUMN IF EXISTS repository_snapshot_id,
  DROP COLUMN IF EXISTS repository_id,
  DROP COLUMN IF EXISTS workflow_run_id,
  DROP COLUMN IF EXISTS workflow_plan_id,
  DROP COLUMN IF EXISTS workflow_id;

ALTER TABLE workflow_run_records
  DROP COLUMN IF EXISTS repository_snapshot_id;

ALTER TABLE workflow_plan_records
  DROP COLUMN IF EXISTS repository_snapshot_id;
