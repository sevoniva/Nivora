CREATE TABLE IF NOT EXISTS runtime_artifacts (
  id TEXT PRIMARY KEY,
  artifact_type TEXT NOT NULL DEFAULT '',
  name TEXT NOT NULL DEFAULT '',
  version_name TEXT NOT NULL DEFAULT '',
  reference TEXT NOT NULL,
  digest TEXT NOT NULL DEFAULT '',
  registry TEXT NOT NULL DEFAULT '',
  repository TEXT NOT NULL DEFAULT '',
  media_type TEXT NOT NULL DEFAULT '',
  size_bytes BIGINT NOT NULL DEFAULT 0,
  manifest_schema TEXT NOT NULL DEFAULT '',
  payload JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_runtime_artifacts_type_created_at ON runtime_artifacts (artifact_type, created_at);
CREATE INDEX IF NOT EXISTS idx_runtime_artifacts_reference ON runtime_artifacts (reference);
CREATE INDEX IF NOT EXISTS idx_runtime_artifacts_digest ON runtime_artifacts (digest);
CREATE INDEX IF NOT EXISTS idx_runtime_artifacts_registry_repository ON runtime_artifacts (registry, repository);
