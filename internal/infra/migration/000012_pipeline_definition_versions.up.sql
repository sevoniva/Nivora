CREATE TABLE IF NOT EXISTS pipeline_definition_versions (
  id TEXT PRIMARY KEY,
  pipeline_id TEXT NOT NULL REFERENCES pipeline_definitions(id) ON DELETE CASCADE,
  version INTEGER NOT NULL,
  definition_hash TEXT NOT NULL,
  definition JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_pipeline_definition_versions_pipeline ON pipeline_definition_versions (pipeline_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_pipeline_definition_versions_unique ON pipeline_definition_versions (pipeline_id, version);

INSERT INTO pipeline_definition_versions (id, pipeline_id, version, definition_hash, definition, created_at, updated_at)
SELECT version_id, id, version, definition_hash, definition, version_created_at, version_updated_at
FROM pipeline_definitions
ON CONFLICT (pipeline_id, version) DO NOTHING;
