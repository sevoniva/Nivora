CREATE TABLE IF NOT EXISTS catalog_artifact_registries (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL DEFAULT '',
  name TEXT NOT NULL,
  registry_type TEXT NOT NULL DEFAULT 'oci',
  registry_url TEXT NOT NULL DEFAULT '',
  endpoint TEXT NOT NULL DEFAULT '',
  insecure BOOLEAN NOT NULL DEFAULT false,
  credential_ref TEXT NOT NULL DEFAULT '',
  capabilities JSONB NOT NULL DEFAULT '[]'::jsonb,
  labels JSONB NOT NULL DEFAULT '{}'::jsonb,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  enabled BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_catalog_artifact_registries_project ON catalog_artifact_registries (project_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_catalog_artifact_registries_project_name ON catalog_artifact_registries (project_id, lower(name));

CREATE TABLE IF NOT EXISTS catalog_policies (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL DEFAULT '',
  environment_id TEXT NOT NULL DEFAULT '',
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  policy_type TEXT NOT NULL DEFAULT 'security',
  mode TEXT NOT NULL DEFAULT 'warn',
  critical_deny INTEGER NOT NULL DEFAULT 0,
  high_warn INTEGER NOT NULL DEFAULT 0,
  require_digest BOOLEAN NOT NULL DEFAULT false,
  approval_on_critical BOOLEAN NOT NULL DEFAULT false,
  labels JSONB NOT NULL DEFAULT '{}'::jsonb,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  enabled BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_catalog_policies_project ON catalog_policies (project_id);
CREATE INDEX IF NOT EXISTS idx_catalog_policies_environment ON catalog_policies (environment_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_catalog_policies_scope_name ON catalog_policies (project_id, environment_id, lower(name));

CREATE TABLE IF NOT EXISTS catalog_policy_attachments (
  id TEXT PRIMARY KEY,
  policy_id TEXT NOT NULL,
  scope_type TEXT NOT NULL,
  scope_id TEXT NOT NULL DEFAULT '',
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  enabled BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_catalog_policy_attachments_policy ON catalog_policy_attachments (policy_id);
CREATE INDEX IF NOT EXISTS idx_catalog_policy_attachments_scope ON catalog_policy_attachments (scope_type, scope_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_catalog_policy_attachments_unique_scope ON catalog_policy_attachments (policy_id, scope_type, scope_id);
