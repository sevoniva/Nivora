CREATE TABLE IF NOT EXISTS workflow_plan_records (
    id TEXT PRIMARY KEY,
    workflow_id TEXT NOT NULL,
    repository_id TEXT NOT NULL DEFAULT '',
    source_path TEXT NOT NULL DEFAULT '',
    ref TEXT NOT NULL DEFAULT '',
    name TEXT NOT NULL,
    content_hash TEXT NOT NULL,
    plan JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_workflow_plan_records_workflow_created ON workflow_plan_records(workflow_id, created_at DESC, id);
CREATE INDEX IF NOT EXISTS idx_workflow_plan_records_repository_created ON workflow_plan_records(repository_id, created_at DESC, id);
CREATE INDEX IF NOT EXISTS idx_workflow_plan_records_content_hash ON workflow_plan_records(content_hash);
