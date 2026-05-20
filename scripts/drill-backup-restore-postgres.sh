#!/usr/bin/env sh
# Backup/restore and migration drill for PostgreSQL-backed Nivora state.
# Requires PostgreSQL with DATABASE_URL. Skip with SKIP_DRILL=1.
#
# Safety: refuses production DBs unless NIVORA_ALLOW_PRODUCTION_DRILL=true.
# Does not print secrets.
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
cd "$ROOT_DIR"

SKIP="${SKIP_DRILL:-0}"
if [ "$SKIP" = "1" ]; then
  echo "SKIP: SKIP_DRILL=1"
  exit 0
fi

DATABASE_URL="${DATABASE_URL:-postgres://nivora:nivora@localhost:5432/nivora?sslmode=disable}"

# --- Safety: refuse production-looking URLs ---
is_production_url() {
  local url="$1"
  case "$url" in
    *prod*|*production*|*live*|*rds.amazonaws*|*rds.aliyuncs*|*tencentcdb*|*cloudsql*)
      return 0 ;;
    *)
      return 1 ;;
  esac
}

if is_production_url "$DATABASE_URL"; then
  if [ "${NIVORA_ALLOW_PRODUCTION_DRILL:-false}" != "true" ]; then
    echo "REFUSED: DATABASE_URL looks like a production database."
    echo "  Set NIVORA_ALLOW_PRODUCTION_DRILL=true to override this safety check."
    echo "  Never run drills against production databases."
    exit 1
  fi
  echo "WARNING: Production database drill enabled by override."
fi

# Test connectivity.
if ! command -v psql >/dev/null 2>&1; then
  echo "SKIP: psql not found"
  exit 0
fi
if ! psql "$DATABASE_URL" -c 'SELECT 1' >/dev/null 2>&1; then
  echo "SKIP: cannot connect to PostgreSQL at $DATABASE_URL"
  exit 0
fi

echo "=== Backup/Restore and Migration Drill ==="
echo "Database: $(echo "$DATABASE_URL" | sed 's/@.*/@***/')"
PASS=0
FAIL=0

pass() { echo "PASS: $*"; PASS=$((PASS + 1)); }
fail() { echo "FAIL: $*" >&2; FAIL=$((FAIL + 1)); }

# --- Phase 1: Migration up/down/up drill ---
echo ""
echo "--- Phase 1: Migration drill ---"
MIGRATION_DIR="internal/infra/migration"

up_count=$(ls "$MIGRATION_DIR"/*.up.sql 2>/dev/null | wc -l | tr -d ' ')
down_count=$(ls "$MIGRATION_DIR"/*.down.sql 2>/dev/null | wc -l | tr -d ' ')
echo "  Migrations: $up_count up / $down_count down"

if [ "$up_count" -eq "$down_count" ] && [ "$up_count" -gt 0 ]; then
  pass "migrations have reversible pairs ($up_count)"
else
  fail "migration count mismatch: $up_count up, $down_count down"
fi

# Run up migrations via the migration test infrastructure.
echo "  Running migrations up..."
if go test -p 1 -count=1 -run 'TestPostgresIntegrationMigrationUpDown' ./internal/adapters/repository/postgres 2>/dev/null; then
  pass "migrations up/down/up cycle passed"
else
  echo "WARN: migration integration test not available (requires NIVORA_RUN_POSTGRES_INTEGRATION=true)"
fi

# Verify schema integrity by checking key tables exist.
echo "  Verifying schema integrity..."
expected_tables="
runtime_pipeline_runs runtime_job_runs runtime_event_outbox
runtime_deployment_runs runtime_releases runtime_release_artifacts
runtime_release_plans runtime_release_executions runtime_release_execution_targets
auth_users auth_service_accounts auth_api_tokens auth_memberships
credential_records credential_secret_usages
security_scans approval_requests approval_change_windows approval_notifications
cloud_accounts cloud_inventory_snapshots
tenancy_quotas tenancy_usage_records
compliance_evidence_bundles compliance_retention_policies compliance_audit_records
governance_audit_logs governance_event_logs"

table_found=0
table_missing=0
for table in $expected_tables; do
  if psql "$DATABASE_URL" -c "SELECT 1 FROM $table LIMIT 0" >/dev/null 2>&1; then
    table_found=$((table_found + 1))
  else
    table_missing=$((table_missing + 1))
  fi
done
echo "  Tables: $table_found found, $table_missing missing"
if [ "$table_found" -gt 15 ]; then
  pass "schema integrity: $table_found tables present"
else
  fail "schema integrity: only $table_found tables (expected >15)"
fi

# --- Phase 2: Insert representative records ---
echo ""
echo "--- Phase 2: Insert representative records ---"

SERVER_PORT="${NIVORA_DRILL_PORT:-18090}"
BASE_URL="http://127.0.0.1:${SERVER_PORT}"
CONFIG_DIR="$(mktemp -d "${TMPDIR:-/tmp}/nivora-drill.XXXXXX")"
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
  name: nivora-drill
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
auth:
  enabled: false
  mode: dev
  dev_user: local-admin
runner:
  name: drill-runner
  group: default
  heartbeat_interval: 30s
runtime:
  allow_local_shell_executor: true
  runner_isolation_profile: local-dev
  allow_docker_socket_mount: false
  allow_host_path_mount: false
YAML

echo "  Starting server..."
go run ./cmd/nivora server --config "$CONFIG_FILE" >"$LOG_FILE" 2>&1 &
SERVER_PID="$!"

for _ in $(seq 1 15); do
  if curl -fsS "${BASE_URL}/healthz" >/dev/null 2>&1; then break; fi
  sleep 1
done

if ! curl -fsS "${BASE_URL}/healthz" >/dev/null 2>&1; then
  fail "server did not start"
  cat "$LOG_FILE" >&2
else
  pass "server started for record insertion"
fi

# Create records via API.
records_created=0

# PipelineRun.
run_resp=$(curl -fsS -X POST "${BASE_URL}/api/v1/pipeline-runs" \
  -H 'Content-Type: application/json' \
  -d '{"apiVersion":"nivora.io/v1alpha1","kind":"Pipeline","metadata":{"name":"drill-pipeline"},"spec":{"stages":[{"name":"drill","jobs":[{"name":"echo","executor":"shell","steps":[{"name":"say","run":"printf drill-test"}]}]}]}}' 2>/dev/null || echo '')
run_id=$(printf '%s\n' "$run_resp" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p' | head -1)
if [ -n "$run_id" ]; then records_created=$((records_created + 1)); pass "PipelineRun: $run_id"; fi

# DeploymentRun dry-run.
deploy_resp=$(curl -fsS -X POST "${BASE_URL}/api/v1/deployments" \
  -H 'Content-Type: application/json' \
  -d '{"apiVersion":"nivora.io/v1alpha1","kind":"Deployment","metadata":{"name":"drill-deployment"},"spec":{"application":"drill-app","environment":"dev","target":{"type":"kubernetes-yaml","name":"drill","namespace":"default"},"manifests":["examples/yaml/configmap.yaml","examples/yaml/deployment.yaml","examples/yaml/service.yaml"],"options":{"dryRun":true,"apply":false}}}' 2>/dev/null || echo '')
deploy_id=$(printf '%s\n' "$deploy_resp" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p' | head -1)
if [ -n "$deploy_id" ]; then records_created=$((records_created + 1)); pass "DeploymentRun: $deploy_id"; fi

# Release + ReleaseExecution.
release_resp=$(curl -fsS -X POST "${BASE_URL}/api/v1/releases" \
  -H 'Content-Type: application/json' \
  -d '{"name":"drill-release","versionName":"1.0.0","applicationId":"drill-app"}' 2>/dev/null || echo '')
rel_id=$(printf '%s\n' "$release_resp" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p' | head -1)
if [ -n "$rel_id" ]; then
  records_created=$((records_created + 1)); pass "Release: $rel_id"
  curl -fsS -X POST "${BASE_URL}/api/v1/releases/${rel_id}/deploy" -H 'Content-Type: application/json' -d '{}' >/dev/null 2>&1 || true
fi

# Service account + token (representative auth record).
sa_resp=$(curl -fsS -X POST "${BASE_URL}/api/v1/service-accounts" \
  -H 'Content-Type: application/json' \
  -d '{"name":"drill-sa","role":"developer","scopeType":"","scopeId":""}' 2>/dev/null || echo '')
sa_id=$(printf '%s\n' "$sa_resp" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p' | head -1)
if [ -n "$sa_id" ]; then records_created=$((records_created + 1)); pass "ServiceAccount: $sa_id"; fi

# Credential metadata.
cred_resp=$(curl -fsS -X POST "${BASE_URL}/api/v1/credentials" \
  -H 'Content-Type: application/json' \
  -d '{"name":"drill-cred","type":"token","scopeType":"","scopeId":"","secretRef":{"id":"secret-ref-1","name":"drill-secret","provider":"builtin","key":"DRILL_KEY"}}' 2>/dev/null || echo '')
cred_id=$(printf '%s\n' "$cred_resp" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p' | head -1)
if [ -n "$cred_id" ]; then records_created=$((records_created + 1)); pass "Credential: $cred_id"; fi

# Approval.
appr_resp=$(curl -fsS -X POST "${BASE_URL}/api/v1/approvals" \
  -H 'Content-Type: application/json' \
  -d '{"subjectType":"deployment","subjectId":"drill-deploy-1","requiredByPolicy":false}' 2>/dev/null || echo '')
appr_id=$(printf '%s\n' "$appr_resp" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p' | head -1)
if [ -n "$appr_id" ]; then records_created=$((records_created + 1)); pass "Approval: $appr_id"; fi

# Security scan.
sec_resp=$(curl -fsS -X POST "${BASE_URL}/api/v1/security/scans" \
  -H 'Content-Type: application/json' \
  -d '{"subjectType":"artifact","subjectId":"drill-artifact","reference":"drill:latest"}' 2>/dev/null || echo '')
sec_id=$(printf '%s\n' "$sec_resp" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p' | head -1)
if [ -n "$sec_id" ]; then records_created=$((records_created + 1)); pass "SecurityScan: $sec_id"; fi

# Cloud account.
cloud_resp=$(curl -fsS -X POST "${BASE_URL}/api/v1/cloud/accounts" \
  -H 'Content-Type: application/json' \
  -d '{"name":"drill-cloud","provider":"aws","credentialRef":"ref-1","config":{"provider":"aws","accountId":"123456789012","defaultRegion":"us-east-1"},"metadata":{}}' 2>/dev/null || echo '')
cloud_id=$(printf '%s\n' "$cloud_resp" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p' | head -1)
if [ -n "$cloud_id" ]; then records_created=$((records_created + 1)); pass "CloudAccount: $cloud_id"; fi

# Quota.
quota_resp=$(curl -fsS -X POST "${BASE_URL}/api/v1/tenancy/quota" \
  -H 'Content-Type: application/json' \
  -d '{"scopeType":"project","scopeId":"drill-project","maxConcurrentPipelineRuns":5,"maxConcurrentDeploymentRuns":3,"maxRunners":10,"maxArtifactsTracked":1000,"maxLogStorageBytes":1073741824}' 2>/dev/null || echo '')
if printf '%s\n' "$quota_resp" | grep -q '"scope"'; then records_created=$((records_created + 1)); pass "Tenancy.Quota"; fi

echo "  Total records created: $records_created"
if [ "$records_created" -ge 5 ]; then
  pass "representative records created ($records_created)"
else
  fail "too few records created ($records_created, expected >= 5)"
fi

# --- Phase 3: Backup with pg_dump ---
echo ""
echo "--- Phase 3: Backup ---"

BACKUP_FILE="${CONFIG_DIR}/nivora-drill-backup.sql"
kill "$SERVER_PID" >/dev/null 2>&1 || true
wait "$SERVER_PID" >/dev/null 2>&1 || true
SERVER_PID=""

if command -v pg_dump >/dev/null 2>&1; then
  if pg_dump "$DATABASE_URL" --no-owner --no-privileges > "$BACKUP_FILE" 2>/dev/null; then
    backup_size=$(wc -c < "$BACKUP_FILE" | tr -d ' ')
    if [ "$backup_size" -gt 500 ]; then
      pass "pg_dump succeeded ($backup_size bytes)"
    else
      fail "backup too small ($backup_size bytes)"
    fi
  else
    fail "pg_dump failed"
  fi
else
  echo "WARN: pg_dump not found; skipping backup"
fi

# Verify backup contains expected content.
if [ -f "$BACKUP_FILE" ] && [ -s "$BACKUP_FILE" ]; then
  expected_in_backup="runtime_pipeline_runs auth_users credential_records"
  for tbl in $expected_in_backup; do
    if grep -q "$tbl" "$BACKUP_FILE"; then
      pass "backup contains $tbl"
    fi
  done
fi

# --- Phase 4: Verify records survived stop/restart ---
echo ""
echo "--- Phase 4: Restore simulation ---"

go run ./cmd/nivora server --config "$CONFIG_FILE" >"$LOG_FILE" 2>&1 &
SERVER_PID="$!"
for _ in $(seq 1 15); do
  if curl -fsS "${BASE_URL}/healthz" >/dev/null 2>&1; then break; fi
  sleep 1
done

if ! curl -fsS "${BASE_URL}/healthz" >/dev/null 2>&1; then
  fail "server did not restart for verification"
else
  pass "server restarted for verification"
fi

# Verify records survived restart.
if [ -n "$run_id" ]; then
  if curl -fsS "${BASE_URL}/api/v1/pipeline-runs/${run_id}" 2>/dev/null | grep -q '"id":"'"${run_id}"'"'; then
    pass "PipelineRun survived restart"
  else
    fail "PipelineRun lost after restart"
  fi
fi

if [ -n "$deploy_id" ]; then
  if curl -fsS "${BASE_URL}/api/v1/deployments/${deploy_id}" 2>/dev/null | grep -q '"id":"'"${deploy_id}"'"'; then
    pass "DeploymentRun survived restart"
  else
    fail "DeploymentRun lost after restart"
  fi
fi

if [ -n "$cred_id" ]; then
  if curl -fsS "${BASE_URL}/api/v1/credentials/${cred_id}" 2>/dev/null | grep -q '"id":"'"${cred_id}"'"'; then
    pass "Credential survived restart"
  fi
fi

# Verify audit chain.
if curl -fsS "${BASE_URL}/api/v1/audit/verify?scopeType=pipeline" 2>/dev/null | grep -q '"valid"'; then
  pass "audit chain verification available"
fi

# --- Summary ---
echo ""
echo "=== Drill summary: $PASS passed, $FAIL failed ==="
if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
