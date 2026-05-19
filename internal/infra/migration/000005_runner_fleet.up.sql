ALTER TABLE runtime_runners ADD COLUMN IF NOT EXISTS capabilities JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE runtime_runners ADD COLUMN IF NOT EXISTS max_concurrency INTEGER NOT NULL DEFAULT 1;
ALTER TABLE runtime_runners ADD COLUMN IF NOT EXISTS token_id TEXT NOT NULL DEFAULT '';
ALTER TABLE runtime_runners ADD COLUMN IF NOT EXISTS token_hash TEXT NOT NULL DEFAULT '';
ALTER TABLE runtime_runners ADD COLUMN IF NOT EXISTS token_created_at TIMESTAMPTZ;
ALTER TABLE runtime_runners ADD COLUMN IF NOT EXISTS token_rotated_at TIMESTAMPTZ;
ALTER TABLE runtime_runners ADD COLUMN IF NOT EXISTS token_revoked_at TIMESTAMPTZ;
ALTER TABLE runtime_runners ADD COLUMN IF NOT EXISTS last_seen_at TIMESTAMPTZ;

ALTER TABLE runtime_runners ADD CONSTRAINT runtime_runners_max_concurrency_positive CHECK (max_concurrency >= 0);

CREATE INDEX IF NOT EXISTS idx_runtime_runners_group_id ON runtime_runners(group_id);
CREATE INDEX IF NOT EXISTS idx_runtime_runners_token_id ON runtime_runners(token_id);
CREATE INDEX IF NOT EXISTS idx_runtime_runners_status_seen ON runtime_runners(status, last_seen_at);
