CREATE TABLE IF NOT EXISTS compliance_evidence_bundles (
  id TEXT PRIMARY KEY,
  subject_type TEXT NOT NULL,
  subject_id TEXT NOT NULL,
  scope_type TEXT NOT NULL DEFAULT '',
  scope_id TEXT NOT NULL DEFAULT '',
  summary TEXT NOT NULL DEFAULT '',
  payload JSONB NOT NULL,
  generated_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS compliance_retention_policies (
  id TEXT PRIMARY KEY,
  scope_type TEXT NOT NULL DEFAULT '',
  scope_id TEXT NOT NULL DEFAULT '',
  log_days INTEGER NOT NULL DEFAULT 30,
  audit_days INTEGER NOT NULL DEFAULT 365,
  event_days INTEGER NOT NULL DEFAULT 180,
  evidence_days INTEGER NOT NULL DEFAULT 365,
  immutable_audit BOOLEAN NOT NULL DEFAULT true,
  payload JSONB NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS compliance_audit_records (
  id TEXT PRIMARY KEY,
  actor_id TEXT NOT NULL DEFAULT '',
  action TEXT NOT NULL,
  subject_type TEXT NOT NULL DEFAULT '',
  subject_id TEXT NOT NULL DEFAULT '',
  subject TEXT NOT NULL DEFAULT '',
  scope_type TEXT NOT NULL DEFAULT '',
  scope_id TEXT NOT NULL DEFAULT '',
  correlation_id TEXT NOT NULL DEFAULT '',
  request_id TEXT NOT NULL DEFAULT '',
  previous_hash TEXT NOT NULL DEFAULT '',
  record_hash TEXT NOT NULL DEFAULT '',
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_compliance_evidence_subject ON compliance_evidence_bundles(subject_type, subject_id, generated_at);
CREATE INDEX IF NOT EXISTS idx_compliance_evidence_scope ON compliance_evidence_bundles(scope_type, scope_id, generated_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_compliance_retention_scope ON compliance_retention_policies(scope_type, scope_id);
CREATE INDEX IF NOT EXISTS idx_compliance_audit_subject ON compliance_audit_records(subject_type, subject_id, created_at);
CREATE INDEX IF NOT EXISTS idx_compliance_audit_scope ON compliance_audit_records(scope_type, scope_id, created_at);
