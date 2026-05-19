DROP INDEX IF EXISTS idx_runtime_runners_status_seen;
DROP INDEX IF EXISTS idx_runtime_runners_token_id;
DROP INDEX IF EXISTS idx_runtime_runners_group_id;

ALTER TABLE runtime_runners DROP CONSTRAINT IF EXISTS runtime_runners_max_concurrency_positive;

ALTER TABLE runtime_runners DROP COLUMN IF EXISTS last_seen_at;
ALTER TABLE runtime_runners DROP COLUMN IF EXISTS token_revoked_at;
ALTER TABLE runtime_runners DROP COLUMN IF EXISTS token_rotated_at;
ALTER TABLE runtime_runners DROP COLUMN IF EXISTS token_created_at;
ALTER TABLE runtime_runners DROP COLUMN IF EXISTS token_hash;
ALTER TABLE runtime_runners DROP COLUMN IF EXISTS token_id;
ALTER TABLE runtime_runners DROP COLUMN IF EXISTS max_concurrency;
ALTER TABLE runtime_runners DROP COLUMN IF EXISTS capabilities;
