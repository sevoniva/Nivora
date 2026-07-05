CREATE TABLE IF NOT EXISTS repository_records (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    provider TEXT NOT NULL,
    url TEXT NOT NULL,
    web_url TEXT NOT NULL DEFAULT '',
    default_branch TEXT NOT NULL DEFAULT '',
    credential_ref TEXT NOT NULL DEFAULT '',
    project_id TEXT NOT NULL DEFAULT '',
    environment_id TEXT NOT NULL DEFAULT '',
    labels JSONB NOT NULL DEFAULT '{}'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_repository_records_project_id ON repository_records(project_id);
CREATE INDEX IF NOT EXISTS idx_repository_records_status ON repository_records(status);
CREATE INDEX IF NOT EXISTS idx_repository_records_updated_at ON repository_records(updated_at);

CREATE TABLE IF NOT EXISTS repository_snapshots (
    id TEXT PRIMARY KEY,
    repository_id TEXT NOT NULL,
    ref TEXT NOT NULL DEFAULT '',
    commit_sha TEXT NOT NULL DEFAULT '',
    branch TEXT NOT NULL DEFAULT '',
    tag TEXT NOT NULL DEFAULT '',
    tree_hash TEXT NOT NULL,
    files JSONB NOT NULL DEFAULT '[]'::jsonb,
    detected_languages JSONB NOT NULL DEFAULT '[]'::jsonb,
    detected_frameworks JSONB NOT NULL DEFAULT '[]'::jsonb,
    detected_build_tools JSONB NOT NULL DEFAULT '[]'::jsonb,
    detected_package_managers JSONB NOT NULL DEFAULT '[]'::jsonb,
    detected_deployment_files JSONB NOT NULL DEFAULT '[]'::jsonb,
    detected_workflow_files JSONB NOT NULL DEFAULT '[]'::jsonb,
    detected_security_files JSONB NOT NULL DEFAULT '[]'::jsonb,
    warnings JSONB NOT NULL DEFAULT '[]'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_repository_snapshots_repository_created ON repository_snapshots(repository_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_repository_snapshots_tree_hash ON repository_snapshots(tree_hash);

CREATE TABLE IF NOT EXISTS repository_intelligence (
    repository_id TEXT NOT NULL,
    snapshot_id TEXT NOT NULL,
    language_summary JSONB NOT NULL DEFAULT '[]'::jsonb,
    framework_summary JSONB NOT NULL DEFAULT '[]'::jsonb,
    build_command_candidates JSONB NOT NULL DEFAULT '[]'::jsonb,
    test_command_candidates JSONB NOT NULL DEFAULT '[]'::jsonb,
    package_command_candidates JSONB NOT NULL DEFAULT '[]'::jsonb,
    deployment_target_candidates JSONB NOT NULL DEFAULT '[]'::jsonb,
    security_scan_candidates JSONB NOT NULL DEFAULT '[]'::jsonb,
    recommended_nivora_workflow_draft TEXT NOT NULL DEFAULT '',
    warnings JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (repository_id, snapshot_id)
);

CREATE INDEX IF NOT EXISTS idx_repository_intelligence_snapshot_id ON repository_intelligence(snapshot_id);
CREATE INDEX IF NOT EXISTS idx_repository_intelligence_created_at ON repository_intelligence(created_at);
