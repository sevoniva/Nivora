CREATE TABLE IF NOT EXISTS runtime_runner_groups (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL DEFAULT '',
  environment_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
  name TEXT NOT NULL,
  labels JSONB NOT NULL DEFAULT '{}'::jsonb,
  max_concurrency INTEGER NOT NULL DEFAULT 0,
  executors JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  version INTEGER NOT NULL DEFAULT 1,
  CONSTRAINT runtime_runner_groups_max_concurrency_nonnegative CHECK (max_concurrency >= 0)
);

CREATE INDEX IF NOT EXISTS idx_runtime_runner_groups_project_id ON runtime_runner_groups(project_id);
CREATE INDEX IF NOT EXISTS idx_runtime_runner_groups_updated_at ON runtime_runner_groups(updated_at);
