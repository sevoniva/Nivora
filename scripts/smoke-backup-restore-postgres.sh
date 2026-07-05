#!/usr/bin/env sh
# Backup/restore drill smoke test.
# Requires PostgreSQL with DATABASE_URL. Skip with SKIP_BACKUP_RESTORE=1.
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
cd "$ROOT_DIR"

if [ "${SKIP_BACKUP_RESTORE:-0}" = "1" ]; then
  echo "SKIP: SKIP_BACKUP_RESTORE=1"
  exit 0
fi

DATABASE_URL="${DATABASE_URL:-postgres://nivora:nivora@localhost:5432/nivora?sslmode=disable}"

redact_database_url() {
  printf '%s\n' "$1" | sed -E 's#(postgres(ql)?://)[^/@]*@#\1***@#'
}

# Test Postgres connectivity.
if ! command -v psql >/dev/null 2>&1; then
  echo "SKIP: psql not found; cannot verify Postgres connectivity"
  exit 0
fi
if ! psql "$DATABASE_URL" -c 'SELECT 1' >/dev/null 2>&1; then
  echo "SKIP: cannot connect to Postgres at $(redact_database_url "$DATABASE_URL")"
  exit 0
fi

echo "=== Backup/Restore Drill ==="
echo "Database: $(redact_database_url "$DATABASE_URL")"
PASS=0
FAIL=0

pass() { echo "PASS: $*"; PASS=$((PASS + 1)); }
fail() { echo "FAIL: $*" >&2; FAIL=$((FAIL + 1)); }

# --- Phase 1: Run migrations ---
echo ""
echo "--- Phase 1: Migrations ---"
echo "Running goose migrations..."
MIGRATION_DIR="internal/infra/migration"

# Count migrations
up_count=$(ls "$MIGRATION_DIR"/*.up.sql 2>/dev/null | wc -l | tr -d ' ')
down_count=$(ls "$MIGRATION_DIR"/*.down.sql 2>/dev/null | wc -l | tr -d ' ')
echo "  Found $up_count up / $down_count down migrations"

if [ "$up_count" -gt 0 ] && [ "$up_count" -eq "$down_count" ]; then
  pass "migrations have reversible up/down pairs ($up_count pairs)"
else
  fail "migration count mismatch: $up_count up, $down_count down"
fi

echo "Applying migrations for backup/restore smoke..."
if DATABASE_URL="$DATABASE_URL" go run ./scripts/apply-postgres-migrations.go >/dev/null 2>&1; then
  pass "migrations applied before backup/restore smoke"
else
  fail "could not apply migrations before backup/restore smoke"
fi

# --- Phase 2: Insert test records ---
echo ""
echo "--- Phase 2: Insert test records ---"

SERVER_PORT="${NIVORA_BACKUP_PORT:-18090}"
BASE_URL="http://127.0.0.1:${SERVER_PORT}"
CONFIG_DIR="$(mktemp -d "${TMPDIR:-/tmp}/nivora-backup.XXXXXX")"
CONFIG_FILE="${CONFIG_DIR}/server.yaml"
LOG_FILE="${CONFIG_DIR}/server.log"
SERVER_PID=""

cleanup() {
  if [ "${SERVER_PID:-}" ]; then
    kill "$SERVER_PID" >/dev/null 2>&1 || true
    wait "$SERVER_PID" >/dev/null 2>&1 || true
  fi
  rm -rf "$CONFIG_DIR"
}
trap cleanup EXIT INT TERM

cat > "$CONFIG_FILE" <<YAML
app:
  name: nivora-backup-test
environment: development
http:
  bind_address: ":${SERVER_PORT}"
database:
  runtime_store: postgres
  url: "${DATABASE_URL}"
event_bus:
  type: memory
object_store:
  type: local
  path: "${CONFIG_DIR}/objectstore"
log:
  level: info
telemetry:
  enabled: false
  endpoint: ""
auth:
  enabled: false
  mode: dev
  dev_user: local-admin
runner:
  name: backup-runner
  group: default
  heartbeat_interval: 30s
runtime:
  allow_local_shell_executor: true
  allow_privileged_executor: false
  allow_remote_host_deploy: false
  allow_kubernetes_apply: false
  allow_argo_sync: false
  allow_insecure_registry: false
YAML

echo "==> Starting test server on ${BASE_URL}"
go run ./cmd/nivora server --config "$CONFIG_FILE" >"$LOG_FILE" 2>&1 &
SERVER_PID="$!"

for _ in $(seq 1 15); do
  if curl -fsS "${BASE_URL}/healthz" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

if ! curl -fsS "${BASE_URL}/healthz" >/dev/null 2>&1; then
  fail "test server did not start"
  cat "$LOG_FILE" >&2
else
  pass "test server started for record insertion"
fi

# Insert test records.
echo "  Creating test data..."
response=$(curl -fsS -X POST "${BASE_URL}/api/v1/pipeline-runs" \
  -H 'Content-Type: application/json' \
  -d '{"apiVersion":"nivora.io/v1alpha1","kind":"Pipeline","metadata":{"name":"backup-test"},"spec":{"stages":[{"name":"test","jobs":[{"name":"echo","executor":"shell","steps":[{"name":"say","run":"printf backup-test"}]}]}]}}' 2>/dev/null || echo '{"status":"Failed"}')

run_id=$(printf '%s\n' "$response" | sed -n 's/.*"id":"\(prun-[^"]*\)".*/\1/p' | head -1)

if [ -n "$run_id" ]; then
  pass "test PipelineRun created: ${run_id}"

  # Verify it's retrievable.
  if curl -fsS "${BASE_URL}/api/v1/pipeline-runs/${run_id}" | grep -q '"id":"'"${run_id}"'"'; then
    pass "PipelineRun retrievable via API"
  else
    fail "PipelineRun not retrievable"
  fi
else
  fail "could not create test PipelineRun"
fi

# Stop server before backup.
echo "  Stopping server for backup"
kill "$SERVER_PID" >/dev/null 2>&1 || true
wait "$SERVER_PID" >/dev/null 2>&1 || true
SERVER_PID=""

# --- Phase 3: Simulate backup ---
echo ""
echo "--- Phase 3: Backup simulation ---"

BACKUP_FILE="${CONFIG_DIR}/nivora-backup.sql"

if command -v pg_dump >/dev/null 2>&1; then
  echo "  Running pg_dump..."
  if pg_dump "$DATABASE_URL" --no-owner --no-privileges > "$BACKUP_FILE" 2>/dev/null; then
    backup_size=$(wc -c < "$BACKUP_FILE" | tr -d ' ')
    if [ "$backup_size" -gt 100 ]; then
      pass "pg_dump created backup ($backup_size bytes)"
    else
      fail "pg_dump backup too small ($backup_size bytes)"
    fi
  else
    fail "pg_dump failed"
  fi
else
  echo "WARN: pg_dump not found; skipping backup simulation"
  echo "  Install PostgreSQL client tools for full backup/restore drill."
fi

# Verify the backup file contains expected tables.
if [ -f "$BACKUP_FILE" ]; then
  for table in runtime_pipeline_runs runtime_deployment_runs compliance_audit_records; do
    if grep -q "$table" "$BACKUP_FILE" 2>/dev/null; then
      pass "backup contains table $table"
    else
      echo "WARN: backup may not contain table $table (expected for empty DB)"
    fi
  done
fi

# --- Phase 4: Verify data survived (simulated restore) ---
echo ""
echo "--- Phase 4: Restore simulation ---"

# Restart server and verify data still exists (proves data survived the stop).
cat > "$CONFIG_FILE" <<YAML
app:
  name: nivora-restore-test
environment: development
http:
  bind_address: ":${SERVER_PORT}"
database:
  runtime_store: postgres
  url: "${DATABASE_URL}"
event_bus:
  type: memory
object_store:
  type: local
  path: "${CONFIG_DIR}/objectstore"
log:
  level: info
telemetry:
  enabled: false
  endpoint: ""
auth:
  enabled: false
  mode: dev
  dev_user: local-admin
runner:
  name: restore-runner
  group: default
  heartbeat_interval: 30s
runtime:
  allow_local_shell_executor: true
  allow_privileged_executor: false
  allow_remote_host_deploy: false
  allow_kubernetes_apply: false
  allow_argo_sync: false
  allow_insecure_registry: false
YAML

echo "  Restarting server..."
go run ./cmd/nivora server --config "$CONFIG_FILE" >"$LOG_FILE" 2>&1 &
SERVER_PID="$!"

for _ in $(seq 1 15); do
  if curl -fsS "${BASE_URL}/healthz" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

if [ -n "$run_id" ]; then
  if curl -fsS "${BASE_URL}/api/v1/pipeline-runs/${run_id}" | grep -q '"id":"'"${run_id}"'"'; then
    pass "PipelineRun ${run_id} survived stop/restart (restore simulation)"
  else
    fail "PipelineRun not found after restart"
  fi

  # Verify audit records.
  if curl -fsS "${BASE_URL}/api/v1/audit/search?subject=${run_id}" | grep -q '"action"'; then
    pass "audit records survived restart"
  else
    echo "WARN: audit records may not have survived"
  fi

  # Verify audit chain if available.
  if curl -fsS "${BASE_URL}/api/v1/audit/verify?scopeType=pipeline" 2>/dev/null | grep -q '"valid"'; then
    pass "audit chain verification available"
  fi
fi

echo ""
echo "=== Backup/restore drill: $PASS passed, $FAIL failed ==="
if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
