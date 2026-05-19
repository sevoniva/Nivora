CREATE TABLE IF NOT EXISTS auth_users (
  id TEXT PRIMARY KEY,
  username TEXT NOT NULL,
  email TEXT NOT NULL DEFAULT '',
  display_name TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'active',
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS auth_service_accounts (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  scope_type TEXT NOT NULL DEFAULT '',
  scope_id TEXT NOT NULL DEFAULT '',
  role TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'active',
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS auth_api_tokens (
  id TEXT PRIMARY KEY,
  subject_id TEXT NOT NULL,
  subject_type TEXT NOT NULL DEFAULT '',
  name TEXT NOT NULL DEFAULT '',
  scope_type TEXT NOT NULL DEFAULT '',
  scope_id TEXT NOT NULL DEFAULT '',
  role TEXT NOT NULL DEFAULT '',
  token_hash TEXT NOT NULL,
  issued_at TIMESTAMPTZ NOT NULL,
  expires_at TIMESTAMPTZ,
  revoked_at TIMESTAMPTZ,
  last_used_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS auth_memberships (
  id TEXT PRIMARY KEY,
  scope_type TEXT NOT NULL,
  scope_id TEXT NOT NULL DEFAULT '',
  user_id TEXT NOT NULL,
  role TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS credential_records (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  credential_type TEXT NOT NULL,
  scope_type TEXT NOT NULL DEFAULT '',
  scope_id TEXT NOT NULL DEFAULT '',
  provider TEXT NOT NULL DEFAULT '',
  secret_key TEXT NOT NULL DEFAULT '',
  secret_ref_id TEXT NOT NULL DEFAULT '',
  secret_version TEXT NOT NULL DEFAULT '',
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS credential_secret_usages (
  id TEXT PRIMARY KEY,
  secret_ref_id TEXT NOT NULL,
  credential_id TEXT NOT NULL DEFAULT '',
  used_by TEXT NOT NULL DEFAULT '',
  purpose TEXT NOT NULL DEFAULT '',
  subject_type TEXT NOT NULL DEFAULT '',
  subject_id TEXT NOT NULL DEFAULT '',
  environment TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS security_scans (
  id TEXT PRIMARY KEY,
  subject_type TEXT NOT NULL,
  subject_id TEXT NOT NULL,
  scanner TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  summary_total INTEGER NOT NULL DEFAULT 0,
  summary_low INTEGER NOT NULL DEFAULT 0,
  summary_medium INTEGER NOT NULL DEFAULT 0,
  summary_high INTEGER NOT NULL DEFAULT 0,
  summary_critical INTEGER NOT NULL DEFAULT 0,
  findings JSONB NOT NULL DEFAULT '[]'::jsonb,
  policy_decision TEXT NOT NULL DEFAULT '',
  policy_reason TEXT NOT NULL DEFAULT '',
  policy_findings JSONB NOT NULL DEFAULT '[]'::jsonb,
  signature_subject TEXT NOT NULL DEFAULT '',
  signature_status TEXT NOT NULL DEFAULT '',
  signature_result TEXT NOT NULL DEFAULT '',
  signature_key_ref TEXT NOT NULL DEFAULT '',
  signature_identity TEXT NOT NULL DEFAULT '',
  signature_issuer TEXT NOT NULL DEFAULT '',
  sbom_format TEXT NOT NULL DEFAULT '',
  sbom_storage_ref TEXT NOT NULL DEFAULT '',
  sbom_digest TEXT NOT NULL DEFAULT '',
  warnings JSONB NOT NULL DEFAULT '[]'::jsonb,
  started_at TIMESTAMPTZ,
  finished_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS approval_requests (
  id TEXT PRIMARY KEY,
  subject_type TEXT NOT NULL,
  subject_id TEXT NOT NULL,
  environment_id TEXT NOT NULL DEFAULT '',
  target_type TEXT NOT NULL DEFAULT '',
  target_id TEXT NOT NULL DEFAULT '',
  severity TEXT NOT NULL DEFAULT '',
  policy_result_id TEXT NOT NULL DEFAULT '',
  required_by_policy BOOLEAN NOT NULL DEFAULT false,
  status TEXT NOT NULL,
  requested_by TEXT NOT NULL DEFAULT '',
  requested_at TIMESTAMPTZ NOT NULL,
  expires_at TIMESTAMPTZ,
  reason TEXT NOT NULL DEFAULT '',
  participants JSONB NOT NULL DEFAULT '[]'::jsonb,
  decisions JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS approval_change_windows (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  environment_id TEXT NOT NULL,
  timezone TEXT NOT NULL DEFAULT '',
  start_time TEXT NOT NULL,
  end_time TEXT NOT NULL,
  days_of_week JSONB NOT NULL DEFAULT '[]'::jsonb,
  allowed BOOLEAN NOT NULL DEFAULT true,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS approval_notifications (
  id TEXT PRIMARY KEY,
  notification_type TEXT NOT NULL,
  channel TEXT NOT NULL DEFAULT '',
  subject TEXT NOT NULL DEFAULT '',
  body_text TEXT NOT NULL DEFAULT '',
  recipients JSONB NOT NULL DEFAULT '[]'::jsonb,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS cloud_accounts (
  id TEXT PRIMARY KEY,
  provider TEXT NOT NULL,
  name TEXT NOT NULL,
  credential_ref TEXT NOT NULL DEFAULT '',
  config JSONB NOT NULL DEFAULT '{}'::jsonb,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS cloud_inventory_snapshots (
  id TEXT PRIMARY KEY,
  account_id TEXT NOT NULL,
  regions JSONB NOT NULL DEFAULT '[]'::jsonb,
  clusters JSONB NOT NULL DEFAULT '[]'::jsonb,
  hosts JSONB NOT NULL DEFAULT '[]'::jsonb,
  registries JSONB NOT NULL DEFAULT '[]'::jsonb,
  scanned_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS tenancy_quotas (
  id TEXT PRIMARY KEY,
  scope_type TEXT NOT NULL,
  scope_id TEXT NOT NULL DEFAULT '',
  max_pipelines_per_hour INTEGER NOT NULL DEFAULT 0,
  max_deployments_per_hour INTEGER NOT NULL DEFAULT 0,
  max_runners INTEGER NOT NULL DEFAULT 0,
  max_parallel_jobs INTEGER NOT NULL DEFAULT 0,
  max_secrets INTEGER NOT NULL DEFAULT 0,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS tenancy_usage_records (
  id TEXT PRIMARY KEY,
  scope_type TEXT NOT NULL,
  scope_id TEXT NOT NULL DEFAULT '',
  resource_type TEXT NOT NULL,
  resource_count INTEGER NOT NULL DEFAULT 0,
  window_start TIMESTAMPTZ NOT NULL,
  window_end TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS governance_audit_logs (
  id TEXT PRIMARY KEY,
  source TEXT NOT NULL DEFAULT '',
  actor_id TEXT NOT NULL DEFAULT '',
  action TEXT NOT NULL,
  subject TEXT NOT NULL DEFAULT '',
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS governance_event_logs (
  id TEXT PRIMARY KEY,
  source TEXT NOT NULL DEFAULT '',
  event_type TEXT NOT NULL,
  subject TEXT NOT NULL DEFAULT '',
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_auth_users_username ON auth_users(username);
CREATE INDEX IF NOT EXISTS idx_auth_users_status ON auth_users(status);
CREATE INDEX IF NOT EXISTS idx_auth_service_accounts_scope ON auth_service_accounts(scope_type, scope_id);
CREATE INDEX IF NOT EXISTS idx_auth_api_tokens_hash ON auth_api_tokens(token_hash);
CREATE INDEX IF NOT EXISTS idx_auth_api_tokens_subject ON auth_api_tokens(subject_id);
CREATE INDEX IF NOT EXISTS idx_auth_memberships_scope ON auth_memberships(scope_type, scope_id);
CREATE INDEX IF NOT EXISTS idx_auth_memberships_user ON auth_memberships(user_id);
CREATE INDEX IF NOT EXISTS idx_credential_records_scope ON credential_records(scope_type, scope_id);
CREATE INDEX IF NOT EXISTS idx_credential_records_type ON credential_records(credential_type);
CREATE INDEX IF NOT EXISTS idx_credential_secret_usages_ref ON credential_secret_usages(secret_ref_id);
CREATE INDEX IF NOT EXISTS idx_security_scans_subject ON security_scans(subject_type, subject_id);
CREATE INDEX IF NOT EXISTS idx_security_scans_status_created ON security_scans(status, created_at);
CREATE INDEX IF NOT EXISTS idx_approval_requests_subject ON approval_requests(subject_type, subject_id);
CREATE INDEX IF NOT EXISTS idx_approval_requests_status ON approval_requests(status);
CREATE INDEX IF NOT EXISTS idx_approval_change_windows_env ON approval_change_windows(environment_id);
CREATE INDEX IF NOT EXISTS idx_cloud_accounts_provider ON cloud_accounts(provider);
CREATE INDEX IF NOT EXISTS idx_cloud_inventory_snapshots_account ON cloud_inventory_snapshots(account_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_tenancy_quotas_scope ON tenancy_quotas(scope_type, scope_id);
CREATE INDEX IF NOT EXISTS idx_tenancy_usage_scope ON tenancy_usage_records(scope_type, scope_id, window_start);
CREATE INDEX IF NOT EXISTS idx_governance_audit_source ON governance_audit_logs(source, created_at);
CREATE INDEX IF NOT EXISTS idx_governance_audit_subject ON governance_audit_logs(subject, created_at);
CREATE INDEX IF NOT EXISTS idx_governance_event_source ON governance_event_logs(source, created_at);
