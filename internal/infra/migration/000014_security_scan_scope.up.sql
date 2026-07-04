ALTER TABLE IF EXISTS security_scans ADD COLUMN IF NOT EXISTS project_id TEXT NOT NULL DEFAULT '';
ALTER TABLE IF EXISTS security_scans ADD COLUMN IF NOT EXISTS environment_id TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_security_scans_project_created ON security_scans(project_id, created_at);
CREATE INDEX IF NOT EXISTS idx_security_scans_environment_created ON security_scans(environment_id, created_at);
