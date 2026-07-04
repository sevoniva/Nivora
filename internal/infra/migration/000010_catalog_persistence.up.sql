CREATE TABLE IF NOT EXISTS catalog_orgs (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  slug TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  labels JSONB NOT NULL DEFAULT '{}'::jsonb,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  enabled BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_catalog_orgs_slug ON catalog_orgs (lower(slug)) WHERE slug <> '';

CREATE TABLE IF NOT EXISTS catalog_projects (
  id TEXT PRIMARY KEY,
  org_id TEXT NOT NULL,
  name TEXT NOT NULL,
  slug TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  labels JSONB NOT NULL DEFAULT '{}'::jsonb,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  enabled BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_catalog_projects_org ON catalog_projects (org_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_catalog_projects_org_slug ON catalog_projects (org_id, lower(slug)) WHERE slug <> '';

CREATE TABLE IF NOT EXISTS catalog_applications (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL,
  name TEXT NOT NULL,
  slug TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  labels JSONB NOT NULL DEFAULT '{}'::jsonb,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  enabled BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_catalog_applications_project ON catalog_applications (project_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_catalog_applications_project_slug ON catalog_applications (project_id, lower(slug)) WHERE slug <> '';

CREATE TABLE IF NOT EXISTS catalog_environments (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL,
  name TEXT NOT NULL,
  slug TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  labels JSONB NOT NULL DEFAULT '{}'::jsonb,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  enabled BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_catalog_environments_project ON catalog_environments (project_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_catalog_environments_project_slug ON catalog_environments (project_id, lower(slug)) WHERE slug <> '';

CREATE TABLE IF NOT EXISTS catalog_repositories (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL,
  name TEXT NOT NULL,
  url TEXT NOT NULL,
  provider TEXT NOT NULL DEFAULT '',
  default_branch TEXT NOT NULL DEFAULT '',
  credential_ref TEXT NOT NULL DEFAULT '',
  labels JSONB NOT NULL DEFAULT '{}'::jsonb,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  enabled BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_catalog_repositories_project ON catalog_repositories (project_id);

CREATE TABLE IF NOT EXISTS catalog_release_targets (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL DEFAULT '',
  environment_id TEXT NOT NULL,
  name TEXT NOT NULL,
  target_type TEXT NOT NULL,
  context TEXT NOT NULL DEFAULT '',
  namespace TEXT NOT NULL DEFAULT '',
  config_ref TEXT NOT NULL DEFAULT '',
  credential_ref TEXT NOT NULL DEFAULT '',
  labels JSONB NOT NULL DEFAULT '{}'::jsonb,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  allow_apply BOOLEAN NOT NULL DEFAULT false,
  allow_sync BOOLEAN NOT NULL DEFAULT false,
  allow_remote_host_deploy BOOLEAN NOT NULL DEFAULT false,
  enabled BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_catalog_release_targets_project ON catalog_release_targets (project_id);
CREATE INDEX IF NOT EXISTS idx_catalog_release_targets_environment ON catalog_release_targets (environment_id);

CREATE TABLE IF NOT EXISTS pipeline_definitions (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL DEFAULT '',
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  labels JSONB NOT NULL DEFAULT '{}'::jsonb,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  enabled BOOLEAN NOT NULL DEFAULT true,
  version_id TEXT NOT NULL,
  version INTEGER NOT NULL,
  definition_hash TEXT NOT NULL,
  definition JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  version_created_at TIMESTAMPTZ NOT NULL,
  version_updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_pipeline_definitions_project ON pipeline_definitions (project_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_pipeline_definitions_project_name ON pipeline_definitions (project_id, lower(name));
