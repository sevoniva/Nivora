CREATE TABLE IF NOT EXISTS runtime_pipeline_artifacts (
  id TEXT PRIMARY KEY,
  pipeline_run_id TEXT NOT NULL REFERENCES runtime_pipeline_runs(id) ON DELETE CASCADE,
  stage_run_id TEXT NOT NULL DEFAULT '',
  job_run_id TEXT NOT NULL DEFAULT '',
  step_run_id TEXT NOT NULL DEFAULT '',
  name TEXT NOT NULL,
  artifact_type TEXT NOT NULL DEFAULT '',
  size_bytes BIGINT NOT NULL DEFAULT 0,
  content_hash TEXT NOT NULL DEFAULT '',
  storage_ref TEXT NOT NULL DEFAULT '',
  retention_days INTEGER NOT NULL DEFAULT 0,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  payload JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS runtime_pipeline_cache_entries (
  id TEXT PRIMARY KEY,
  pipeline_run_id TEXT NOT NULL REFERENCES runtime_pipeline_runs(id) ON DELETE CASCADE,
  job_run_id TEXT NOT NULL DEFAULT '',
  step_run_id TEXT NOT NULL DEFAULT '',
  cache_key TEXT NOT NULL,
  restore_keys JSONB NOT NULL DEFAULT '[]'::jsonb,
  scope TEXT NOT NULL DEFAULT '',
  hit BOOLEAN NOT NULL DEFAULT false,
  size_bytes BIGINT NOT NULL DEFAULT 0,
  storage_ref TEXT NOT NULL DEFAULT '',
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  payload JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  expires_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS runtime_pipeline_annotations (
  id TEXT PRIMARY KEY,
  pipeline_run_id TEXT NOT NULL REFERENCES runtime_pipeline_runs(id) ON DELETE CASCADE,
  stage_run_id TEXT NOT NULL DEFAULT '',
  job_run_id TEXT NOT NULL DEFAULT '',
  step_run_id TEXT NOT NULL DEFAULT '',
  level TEXT NOT NULL,
  file_path TEXT NOT NULL DEFAULT '',
  line_number INTEGER NOT NULL DEFAULT 0,
  column_number INTEGER NOT NULL DEFAULT 0,
  title TEXT NOT NULL DEFAULT '',
  message TEXT NOT NULL,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  payload JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS runtime_pipeline_step_summaries (
  id TEXT PRIMARY KEY,
  pipeline_run_id TEXT NOT NULL REFERENCES runtime_pipeline_runs(id) ON DELETE CASCADE,
  stage_run_id TEXT NOT NULL DEFAULT '',
  job_run_id TEXT NOT NULL DEFAULT '',
  step_run_id TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL DEFAULT '',
  content TEXT NOT NULL DEFAULT '',
  storage_ref TEXT NOT NULL DEFAULT '',
  format TEXT NOT NULL DEFAULT '',
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  payload JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_runtime_pipeline_artifacts_run_created_at ON runtime_pipeline_artifacts(pipeline_run_id, created_at);
CREATE INDEX IF NOT EXISTS idx_runtime_pipeline_artifacts_job ON runtime_pipeline_artifacts(job_run_id, created_at);
CREATE INDEX IF NOT EXISTS idx_runtime_pipeline_cache_run_created_at ON runtime_pipeline_cache_entries(pipeline_run_id, created_at);
CREATE INDEX IF NOT EXISTS idx_runtime_pipeline_annotations_run_created_at ON runtime_pipeline_annotations(pipeline_run_id, created_at);
CREATE INDEX IF NOT EXISTS idx_runtime_pipeline_annotations_level ON runtime_pipeline_annotations(level, created_at);
CREATE INDEX IF NOT EXISTS idx_runtime_pipeline_summaries_run_created_at ON runtime_pipeline_step_summaries(pipeline_run_id, created_at);
