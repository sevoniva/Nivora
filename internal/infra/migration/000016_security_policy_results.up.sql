CREATE TABLE IF NOT EXISTS security_policy_results (
  id TEXT PRIMARY KEY,
  policy_id TEXT NOT NULL DEFAULT '',
  subject_type TEXT NOT NULL,
  subject_id TEXT NOT NULL,
  project_id TEXT NOT NULL DEFAULT '',
  environment_id TEXT NOT NULL DEFAULT '',
  decision TEXT NOT NULL,
  reason TEXT NOT NULL DEFAULT '',
  findings JSONB NOT NULL DEFAULT '[]'::jsonb,
  evaluated_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_security_policy_results_policy ON security_policy_results(policy_id, evaluated_at);
CREATE INDEX IF NOT EXISTS idx_security_policy_results_subject ON security_policy_results(subject_type, subject_id, evaluated_at);
CREATE INDEX IF NOT EXISTS idx_security_policy_results_project ON security_policy_results(project_id, evaluated_at);
CREATE INDEX IF NOT EXISTS idx_security_policy_results_environment ON security_policy_results(environment_id, evaluated_at);
