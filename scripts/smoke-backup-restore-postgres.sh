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
REQUIRE_ACTUAL_RESTORE="${NIVORA_REQUIRE_ACTUAL_RESTORE:-${CI:-0}}"
ISOLATED_SOURCE="${NIVORA_BACKUP_ISOLATED_SOURCE:-${CI:-0}}"

PASS=0
FAIL=0
SERVER_PID=""
SOURCE_DB_CREATED=0
RESTORE_DB_CREATED=0
ADMIN_DATABASE_URL=""
SOURCE_DATABASE_NAME=""
RESTORE_DATABASE_URL=""
RESTORE_DATABASE_NAME=""

pass() { echo "PASS: $*"; PASS=$((PASS + 1)); }
fail() { echo "FAIL: $*" >&2; FAIL=$((FAIL + 1)); }
warn() { echo "WARN: $*" >&2; }

redact_database_url() {
  printf '%s\n' "$1" | sed -E 's#(postgres(ql)?://)[^/@]*@#\1***@#'
}

redact_file() {
  sed -E 's#(postgres(ql)?://)[^/@]*@#\1***@#g'
}

sql_identifier_safe() {
  case "$1" in
    ""|*[!A-Za-z0-9_]*)
      return 1
      ;;
  esac
  return 0
}

sql_literal() {
  printf '%s' "$1" | sed "s/'/''/g"
}

derive_database_url() {
  source_url="$1"
  current_db="$2"
  target_db="$3"
  if ! sql_identifier_safe "$current_db" || ! sql_identifier_safe "$target_db"; then
    return 1
  fi
  derived=$(printf '%s\n' "$source_url" | sed "s#/${current_db}\\([?]\\|$\\)#/${target_db}\\1#")
  if [ "$derived" = "$source_url" ] && [ "$current_db" != "$target_db" ]; then
    return 1
  fi
  printf '%s\n' "$derived"
}

psql_value() {
  db_url="$1"
  sql="$2"
  psql "$db_url" -At -v ON_ERROR_STOP=1 -c "$sql" 2>/dev/null | tr -d '[:space:]'
}

stop_server() {
  if [ "${SERVER_PID:-}" ]; then
    kill "$SERVER_PID" >/dev/null 2>&1 || true
    wait "$SERVER_PID" >/dev/null 2>&1 || true
    SERVER_PID=""
  fi
}

drop_restore_database() {
  if [ "$RESTORE_DB_CREATED" = "1" ] && [ -n "$ADMIN_DATABASE_URL" ] && [ -n "$RESTORE_DATABASE_NAME" ]; then
    psql "$ADMIN_DATABASE_URL" -v ON_ERROR_STOP=1 -c "DROP DATABASE IF EXISTS \"${RESTORE_DATABASE_NAME}\"" >/dev/null 2>&1 || true
    RESTORE_DB_CREATED=0
  fi
}

drop_source_database() {
  if [ "$SOURCE_DB_CREATED" = "1" ] && [ -n "$ADMIN_DATABASE_URL" ] && [ -n "$SOURCE_DATABASE_NAME" ]; then
    psql "$ADMIN_DATABASE_URL" -v ON_ERROR_STOP=1 -c "DROP DATABASE IF EXISTS \"${SOURCE_DATABASE_NAME}\"" >/dev/null 2>&1 || true
    SOURCE_DB_CREATED=0
  fi
}

cleanup() {
  stop_server
  drop_restore_database
  drop_source_database
  if [ -n "${CONFIG_DIR:-}" ]; then
    rm -rf "$CONFIG_DIR"
  fi
}
trap cleanup EXIT INT TERM

write_server_config() {
  app_name="$1"
  db_url="$2"
  object_path="$3"
  cat > "$CONFIG_FILE" <<YAML
app:
  name: ${app_name}
environment: development
http:
  bind_address: ":${SERVER_PORT}"
database:
  runtime_store: postgres
  url: "${db_url}"
event_bus:
  type: memory
object_store:
  type: local
  path: "${object_path}"
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
}

start_server() {
  app_name="$1"
  db_url="$2"
  object_path="$3"
  write_server_config "$app_name" "$db_url" "$object_path"
  echo "==> Starting test server on ${BASE_URL} using $(redact_database_url "$db_url")"
  go run ./cmd/nivora server --config "$CONFIG_FILE" >"$LOG_FILE" 2>&1 &
  SERVER_PID="$!"
  for _ in $(seq 1 20); do
    if curl -fsS "${BASE_URL}/healthz" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  return 1
}

json_id_with_prefix() {
  body="$1"
  prefix="$2"
  printf '%s\n' "$body" | sed -n 's/.*"id":"\('"$prefix"'[^"]*\)".*/\1/p' | head -1
}

expect_sql_count_at_least() {
  label="$1"
  db_url="$2"
  minimum="$3"
  sql="$4"
  count=$(psql_value "$db_url" "$sql" || printf '')
  case "$count" in
    ''|*[!0-9]*)
      fail "${label}: could not read count"
      return
      ;;
  esac
  if [ "$count" -ge "$minimum" ]; then
    pass "${label}: ${count} rows"
  else
    fail "${label}: got ${count}, want at least ${minimum}"
  fi
}

echo "=== Backup/Restore Drill ==="
echo "Database: $(redact_database_url "$DATABASE_URL")"

# Test Postgres connectivity.
if ! command -v psql >/dev/null 2>&1; then
  echo "SKIP: psql not found; cannot verify Postgres connectivity"
  exit 0
fi
if ! psql "$DATABASE_URL" -c 'SELECT 1' >/dev/null 2>&1; then
  echo "SKIP: cannot connect to Postgres at $(redact_database_url "$DATABASE_URL")"
  exit 0
fi

# Keep the restore drill isolated from other integration-test records in CI.
# Otherwise a broken or deliberately tampered audit chain from a previous test can make
# this backup smoke look like a restore failure even when the dump/import path is healthy.
case "$ISOLATED_SOURCE" in
  1|true|TRUE|yes|YES)
    base_db=$(psql_value "$DATABASE_URL" 'SELECT current_database()' || printf '')
    SOURCE_DATABASE_NAME="nivora_backup_src_$$"
    if sql_identifier_safe "$base_db" && sql_identifier_safe "$SOURCE_DATABASE_NAME"; then
      ADMIN_DATABASE_URL=$(derive_database_url "$DATABASE_URL" "$base_db" "postgres" || printf '')
      SOURCE_DATABASE_URL=$(derive_database_url "$DATABASE_URL" "$base_db" "$SOURCE_DATABASE_NAME" || printf '')
    else
      SOURCE_DATABASE_URL=""
    fi
    if [ -n "$ADMIN_DATABASE_URL" ] && [ -n "$SOURCE_DATABASE_URL" ] &&
       psql "$ADMIN_DATABASE_URL" -v ON_ERROR_STOP=1 -c "DROP DATABASE IF EXISTS \"${SOURCE_DATABASE_NAME}\"" >/dev/null 2>&1 &&
       psql "$ADMIN_DATABASE_URL" -v ON_ERROR_STOP=1 -c "CREATE DATABASE \"${SOURCE_DATABASE_NAME}\"" >/dev/null 2>&1; then
      SOURCE_DB_CREATED=1
      DATABASE_URL="$SOURCE_DATABASE_URL"
      pass "isolated source database created for backup smoke"
      echo "Source database: $(redact_database_url "$DATABASE_URL")"
    else
      warn "could not create isolated source database"
      case "$REQUIRE_ACTUAL_RESTORE" in
        1|true|TRUE|yes|YES)
          fail "isolated source database is required in this environment"
          ;;
      esac
    fi
    ;;
esac

# --- Phase 1: Run migrations ---
echo ""
echo "--- Phase 1: Migrations ---"
echo "Running goose migrations..."
MIGRATION_DIR="internal/infra/migration"

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
ORIGINAL_OBJECTSTORE="${CONFIG_DIR}/objectstore-original"
RESTORE_OBJECTSTORE="${CONFIG_DIR}/objectstore-restored"

if start_server "nivora-backup-test" "$DATABASE_URL" "$ORIGINAL_OBJECTSTORE"; then
  pass "test server started for record insertion"
else
  fail "test server did not start"
  cat "$LOG_FILE" | redact_file >&2
fi

echo "  Creating test PipelineRun..."
pipeline_response=$(curl -fsS -X POST "${BASE_URL}/api/v1/pipeline-runs" \
  -H 'Content-Type: application/json' \
  -d '{"apiVersion":"nivora.io/v1alpha1","kind":"Pipeline","metadata":{"name":"backup-test"},"spec":{"stages":[{"name":"test","jobs":[{"name":"echo","executor":"shell","steps":[{"name":"say","run":"printf backup-test"}]}]}]}}' 2>/dev/null || echo '{"status":"Failed"}')

run_id=$(json_id_with_prefix "$pipeline_response" "prun-")

if [ -n "$run_id" ]; then
  pass "test PipelineRun created: ${run_id}"
  if curl -fsS "${BASE_URL}/api/v1/pipeline-runs/${run_id}" | grep -q '"id":"'"${run_id}"'"'; then
    pass "PipelineRun retrievable via API"
  else
    fail "PipelineRun not retrievable"
  fi
else
  fail "could not create test PipelineRun"
fi

echo "  Creating credential metadata record..."
credential_response=$(curl -fsS -X POST "${BASE_URL}/api/v1/credentials" \
  -H 'Content-Type: application/json' \
  -d '{"name":"backup-registry-credential","type":"registry","scopeType":"project","scopeId":"backup-smoke","secretRef":{"id":"secret-ref-placeholder","name":"placeholder","provider":"builtin","key":"registry/token-placeholder"}}' 2>/dev/null || echo '{"status":"Failed"}')

credential_id=$(json_id_with_prefix "$credential_response" "cred-")
if [ -n "$credential_id" ]; then
  pass "credential metadata created: ${credential_id}"
else
  fail "could not create credential metadata"
fi

bundle_id=""
if [ -n "$run_id" ]; then
  echo "  Generating persisted evidence bundle..."
  evidence_response=$(curl -fsS -X POST "${BASE_URL}/api/v1/evidence/bundles" \
    -H 'Content-Type: application/json' \
    -d '{"subjectType":"pipelineRun","subjectId":"'"${run_id}"'"}' 2>/dev/null || echo '{"status":"Failed"}')
  bundle_id=$(json_id_with_prefix "$evidence_response" "evb-")
  if [ -n "$bundle_id" ]; then
    pass "evidence bundle persisted: ${bundle_id}"
  else
    fail "could not generate evidence bundle"
  fi
fi

stop_server

# --- Phase 3: Backup ---
echo ""
echo "--- Phase 3: Backup ---"

BACKUP_FILE="${CONFIG_DIR}/nivora-backup.sql"
BACKUP_READY=0

if command -v pg_dump >/dev/null 2>&1; then
  echo "  Running pg_dump..."
  PG_DUMP_ERR="${CONFIG_DIR}/pg_dump.err"
  if pg_dump "$DATABASE_URL" --no-owner --no-privileges > "$BACKUP_FILE" 2>"$PG_DUMP_ERR"; then
    backup_size=$(wc -c < "$BACKUP_FILE" | tr -d ' ')
    if [ "$backup_size" -gt 100 ]; then
      BACKUP_READY=1
      pass "pg_dump created backup ($backup_size bytes)"
    else
      fail "pg_dump backup too small ($backup_size bytes)"
    fi
  elif grep -qi "version mismatch\\|server version" "$PG_DUMP_ERR"; then
    warn "pg_dump client is incompatible with the PostgreSQL server version"
    redact_file < "$PG_DUMP_ERR" >&2
  else
    fail "pg_dump failed"
    redact_file < "$PG_DUMP_ERR" >&2
  fi
else
  warn "pg_dump not found; install PostgreSQL client tools for full backup/restore drill"
fi

if [ "$BACKUP_READY" = "1" ]; then
  for table in runtime_pipeline_runs runtime_deployment_runs compliance_audit_records compliance_evidence_bundles credential_records; do
    if grep -q "$table" "$BACKUP_FILE" 2>/dev/null; then
      pass "backup contains table $table"
    else
      fail "backup missing expected table $table"
    fi
  done
fi

# --- Phase 4: Restore into a temporary database ---
echo ""
echo "--- Phase 4: Actual restore into temporary database ---"

ACTUAL_RESTORE_DONE=0
if [ "$BACKUP_READY" = "1" ]; then
  current_db=$(psql_value "$DATABASE_URL" 'SELECT current_database()' || printf '')
  RESTORE_DATABASE_NAME="nivora_restore_$$"
  if sql_identifier_safe "$current_db" && sql_identifier_safe "$RESTORE_DATABASE_NAME"; then
    ADMIN_DATABASE_URL=$(derive_database_url "$DATABASE_URL" "$current_db" "postgres" || printf '')
    RESTORE_DATABASE_URL=$(derive_database_url "$DATABASE_URL" "$current_db" "$RESTORE_DATABASE_NAME" || printf '')
  fi

  if [ -n "$ADMIN_DATABASE_URL" ] && [ -n "$RESTORE_DATABASE_URL" ]; then
    echo "  Creating temporary restore database ${RESTORE_DATABASE_NAME}..."
    if psql "$ADMIN_DATABASE_URL" -v ON_ERROR_STOP=1 -c "DROP DATABASE IF EXISTS \"${RESTORE_DATABASE_NAME}\"" >/dev/null 2>&1 &&
       psql "$ADMIN_DATABASE_URL" -v ON_ERROR_STOP=1 -c "CREATE DATABASE \"${RESTORE_DATABASE_NAME}\"" >/dev/null 2>&1; then
      RESTORE_DB_CREATED=1
      pass "temporary restore database created"
    else
      fail "could not create temporary restore database"
    fi

    if [ "$RESTORE_DB_CREATED" = "1" ]; then
      echo "  Restoring pg_dump into ${RESTORE_DATABASE_NAME}..."
      RESTORE_ERR="${CONFIG_DIR}/restore.err"
      if psql "$RESTORE_DATABASE_URL" -v ON_ERROR_STOP=1 -f "$BACKUP_FILE" >/dev/null 2>"$RESTORE_ERR"; then
        ACTUAL_RESTORE_DONE=1
        pass "backup restored into temporary database"
      else
        fail "restore into temporary database failed"
        redact_file < "$RESTORE_ERR" >&2
      fi
    fi
  else
    warn "could not derive admin/restore database URL from $(redact_database_url "$DATABASE_URL")"
  fi
fi

if [ "$ACTUAL_RESTORE_DONE" = "1" ]; then
  if [ -n "$run_id" ]; then
    run_id_sql=$(sql_literal "$run_id")
    expect_sql_count_at_least "restored PipelineRun" "$RESTORE_DATABASE_URL" 1 "SELECT count(*) FROM runtime_pipeline_runs WHERE id='${run_id_sql}'"
  fi
  if [ -n "$bundle_id" ]; then
    bundle_id_sql=$(sql_literal "$bundle_id")
    expect_sql_count_at_least "restored evidence bundle" "$RESTORE_DATABASE_URL" 1 "SELECT count(*) FROM compliance_evidence_bundles WHERE id='${bundle_id_sql}'"
  fi
  if [ -n "$credential_id" ]; then
    credential_id_sql=$(sql_literal "$credential_id")
    expect_sql_count_at_least "restored credential metadata" "$RESTORE_DATABASE_URL" 1 "SELECT count(*) FROM credential_records WHERE id='${credential_id_sql}'"
  fi
  expect_sql_count_at_least "restored hash-chained audit records" "$RESTORE_DATABASE_URL" 1 "SELECT count(*) FROM compliance_audit_records WHERE record_hash <> ''"

  if start_server "nivora-restore-test" "$RESTORE_DATABASE_URL" "$RESTORE_OBJECTSTORE"; then
    pass "server started against restored database"
  else
    fail "server did not start against restored database"
    cat "$LOG_FILE" | redact_file >&2
  fi

  if [ -n "$run_id" ]; then
    if curl -fsS "${BASE_URL}/api/v1/pipeline-runs/${run_id}" | grep -q '"id":"'"${run_id}"'"'; then
      pass "PipelineRun ${run_id} retrievable from restored database"
    else
      fail "PipelineRun not found from restored database"
    fi
  fi
  if [ -n "$bundle_id" ]; then
    if curl -fsS "${BASE_URL}/api/v1/evidence/bundles/${bundle_id}" | grep -q '"id":"'"${bundle_id}"'"'; then
      pass "evidence bundle ${bundle_id} retrievable from restored database"
    else
      fail "evidence bundle not found from restored database"
    fi
  fi
  if [ -n "$credential_id" ]; then
    if curl -fsS "${BASE_URL}/api/v1/credentials/${credential_id}?scopeType=project&scopeId=backup-smoke" | grep -q '"id":"'"${credential_id}"'"'; then
      pass "credential metadata ${credential_id} retrievable from restored database"
    else
      fail "credential metadata not found from restored database"
    fi
  fi
  if curl -fsS "${BASE_URL}/api/v1/audit/verify?scopeType=pipeline" 2>/dev/null | grep -q '"valid":true'; then
    pass "pipeline audit hash chain verifies from restored database"
  else
    fail "pipeline audit hash chain did not verify from restored database"
  fi
  stop_server
else
  warn "actual temporary-database restore did not run"
  case "$REQUIRE_ACTUAL_RESTORE" in
    1|true|TRUE|yes|YES)
      fail "actual restore is required in this environment"
      ;;
    *)
      echo ""
      echo "--- Phase 4 fallback: same-database restart only ---"
      if start_server "nivora-restore-fallback-test" "$DATABASE_URL" "$RESTORE_OBJECTSTORE"; then
        pass "server restarted against original database"
      else
        fail "server did not restart against original database"
        cat "$LOG_FILE" | redact_file >&2
      fi
      if [ -n "$run_id" ] && curl -fsS "${BASE_URL}/api/v1/pipeline-runs/${run_id}" | grep -q '"id":"'"${run_id}"'"'; then
        pass "PipelineRun ${run_id} survived stop/restart fallback"
      else
        fail "PipelineRun not found after same-database fallback restart"
      fi
      stop_server
      ;;
  esac
fi

echo ""
echo "=== Backup/restore drill: $PASS passed, $FAIL failed ==="
if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
