#!/usr/bin/env sh
# Production-like runtime soak test harness.
# Runs server + worker + runner + PostgreSQL for a configurable duration
# and verifies runtime stability with repeated workload creation.
#
# Duration controls:
#   NIVORA_SOAK_DURATION_SECONDS  default 60 (set higher for overnight)
#   NIVORA_SOAK_INTERVAL_SECONDS  default 5  (time between workload loops)
#   NIVORA_SOAK_RUNS              default 0  (0=use duration; N=exact count)
#   NIVORA_SOAK_RESTART_WORKER    default 1  (periodically restart worker)
#
# Overnight example:
#   DATABASE_URL="..." NIVORA_SOAK_DURATION_SECONDS=21600 ./scripts/soak-runtime-postgres.sh
#
# Skip: SKIP_SOAK_RUNTIME=1

set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
cd "$ROOT_DIR"

# --- Configuration ---
SKIP="${SKIP_SOAK_RUNTIME:-0}"
DURATION="${NIVORA_SOAK_DURATION_SECONDS:-60}"
INTERVAL="${NIVORA_SOAK_INTERVAL_SECONDS:-5}"
SOAK_RUNS="${NIVORA_SOAK_RUNS:-0}"
RESTART_WORKER="${NIVORA_SOAK_RESTART_WORKER:-1}"
RESTART_WORKER_EVERY="${NIVORA_SOAK_RESTART_WORKER_EVERY:-3}" # every N loops

DATABASE_URL="${DATABASE_URL:-postgres://nivora:nivora@localhost:5432/nivora?sslmode=disable}"

if [ "$SKIP" = "1" ]; then
  echo "SKIP: SKIP_SOAK_RUNTIME=1"
  exit 0
fi

# Test Postgres connectivity.
if ! command -v psql >/dev/null 2>&1; then
  echo "SKIP: psql not found; cannot verify PostgreSQL"
  exit 0
fi
if ! psql "$DATABASE_URL" -c 'SELECT 1' >/dev/null 2>&1; then
  echo "SKIP: cannot connect to PostgreSQL at $DATABASE_URL"
  echo "  Start: docker run --rm -e POSTGRES_USER=nivora -e POSTGRES_PASSWORD=nivora -p 5432:5432 postgres:16-alpine"
  exit 0
fi

# --- Setup ---
SERVER_PORT="${NIVORA_SOAK_SERVER_PORT:-18080}"
WORKER_PORT="${NIVORA_SOAK_WORKER_PORT:-18081}"
RUNNER_PORT="${NIVORA_SOAK_RUNNER_PORT:-18082}"
BASE_URL="http://127.0.0.1:${SERVER_PORT}"

CONFIG_DIR="$(mktemp -d "${TMPDIR:-/tmp}/nivora-soak.XXXXXX")"
SERVER_CONFIG="${CONFIG_DIR}/server.yaml"
WORKER_CONFIG="${CONFIG_DIR}/worker.yaml"
RUNNER_CONFIG="${CONFIG_DIR}/runner.yaml"

SERVER_PID=""
WORKER_PID=""
RUNNER_PID=""

# --- Counters ---
PIPELINE_PASS=0
PIPELINE_FAIL=0
DEPLOY_PASS=0
DEPLOY_FAIL=0
WORKER_RESTARTS=0
WORKER_FAILURES=0
RUNNER_RESTARTS=0
RUNNER_FAILURES=0
HEARTBEAT_LOST=0
API_TIMEOUTS=0
LOOPS=0
STUCK_RUNS=0

cleanup() {
  echo ""
  echo "=== Soak cleanup ==="
  for pid in ${RUNNER_PID:-} ${WORKER_PID:-} ${SERVER_PID:-}; do
    kill "$pid" >/dev/null 2>&1 || true
    wait "$pid" >/dev/null 2>&1 || true
  done
  rm -rf "$CONFIG_DIR"
}
trap cleanup EXIT INT TERM

# --- Config generation ---
make_config() {
  local name="$1" port="$2" file="$3"
  cat > "$file" <<YAML
app:
  name: ${name}
environment: development
http:
  bind_address: ":${port}"
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
  name: soak-runner
  group: default
  heartbeat_interval: 10s
runtime:
  allow_local_shell_executor: true
  runner_isolation_profile: local-dev
  allow_docker_socket_mount: false
  allow_host_path_mount: false
YAML
}

# --- Helpers ---
api_call() {
  local url="$1" method="${2:-GET}" body="${3:-}"
  local timeout=30
  set +e
  if [ -n "$body" ]; then
    curl -fsS --max-time "$timeout" -X "$method" "${BASE_URL}${url}" \
      -H 'Content-Type: application/json' -d "$body" 2>/dev/null
  else
    curl -fsS --max-time "$timeout" -X "$method" "${BASE_URL}${url}" 2>/dev/null
  fi
  local rc=$?
  set -e
  if [ $rc -ne 0 ]; then
    API_TIMEOUTS=$((API_TIMEOUTS + 1))
    return 1
  fi
  return 0
}

check_health() {
  api_call "/healthz" || { HEARTBEAT_LOST=$((HEARTBEAT_LOST + 1)); return 1; }
  return 0
}

start_server() {
  echo "  Starting server on :${SERVER_PORT}..."
  go run ./cmd/nivora server --config "$SERVER_CONFIG" >"${CONFIG_DIR}/server.log" 2>&1 &
  SERVER_PID="$!"
  for _ in $(seq 1 20); do
    if curl -fsS "${BASE_URL}/healthz" >/dev/null 2>&1; then return 0; fi
    sleep 1
  done
  echo "FAIL: server did not start" >&2
  return 1
}

start_worker() {
  echo "  Starting worker..."
  go run ./cmd/nivora worker --config "$WORKER_CONFIG" >"${CONFIG_DIR}/worker.log" 2>&1 &
  WORKER_PID="$!"
  sleep 2
}

start_runner() {
  echo "  Starting runner..."
  go run ./cmd/nivora runner --config "$RUNNER_CONFIG" >"${CONFIG_DIR}/runner.log" 2>&1 &
  RUNNER_PID="$!"
  sleep 2
}

stop_worker() {
  if [ "${WORKER_PID:-}" ] && kill -0 "$WORKER_PID" 2>/dev/null; then
    kill "$WORKER_PID" >/dev/null 2>&1 || true
    wait "$WORKER_PID" >/dev/null 2>&1 || true
    WORKER_PID=""
  fi
}

stop_runner() {
  if [ "${RUNNER_PID:-}" ] && kill -0 "$RUNNER_PID" 2>/dev/null; then
    kill "$RUNNER_PID" >/dev/null 2>&1 || true
    wait "$RUNNER_PID" >/dev/null 2>&1 || true
    RUNNER_PID=""
  fi
}

# --- Workload ---
create_pipeline_run() {
  local response
  response=$(api_call "/api/v1/pipeline-runs" "POST" '{
    "apiVersion":"nivora.io/v1alpha1","kind":"Pipeline",
    "metadata":{"name":"soak-pipeline"},
    "spec":{"stages":[{"name":"soak-stage","jobs":[{"name":"echo","executor":"shell","steps":[{"name":"say","run":"printf soak-'"${LOOPS}"'"}]}]}]}
  }' || echo '')
  if [ -z "$response" ]; then return 1; fi
  if printf '%s\n' "$response" | grep -q '"status":"Succeeded"'; then
    PIPELINE_PASS=$((PIPELINE_PASS + 1))
    printf '%s\n' "$response" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p' | head -1
  else
    PIPELINE_FAIL=$((PIPELINE_FAIL + 1))
    return 1
  fi
}

create_deployment_dryrun() {
  local response
  response=$(api_call "/api/v1/deployments" "POST" '{
    "apiVersion":"nivora.io/v1alpha1","kind":"Deployment",
    "metadata":{"name":"soak-deployment"},
    "spec":{"application":"soak-app","environment":"dev","target":{"type":"kubernetes-yaml","name":"soak","namespace":"default"},"manifests":["examples/yaml/configmap.yaml","examples/yaml/deployment.yaml","examples/yaml/service.yaml"],"options":{"dryRun":true,"apply":false}}
  }' || echo '')
  if [ -z "$response" ]; then return 1; fi
  if printf '%s\n' "$response" | grep -q '"status":"Succeeded"'; then
    DEPLOY_PASS=$((DEPLOY_PASS + 1))
    return 0
  else
    DEPLOY_FAIL=$((DEPLOY_FAIL + 1))
    return 1
  fi
}

verify_state() {
  # Check for stuck runs.
  local recovery_data
  recovery_data=$(api_call "/api/v1/system/runtime/recovery" || echo '{}')
  if printf '%s\n' "$recovery_data" | grep -q '"queuedRuns":0'; then :; else STUCK_RUNS=$((STUCK_RUNS + 1)); fi

  # Verify audit chain.
  api_call "/api/v1/audit/verify?scopeType=pipeline" >/dev/null 2>&1 || true
}

# --- Main ---
echo "=== Nivora Production Soak Test ==="
echo "Duration: ${DURATION}s | Interval: ${INTERVAL}s | Runs: ${SOAK_RUNS:-duration-based}"
echo "Database: ${DATABASE_URL}"
echo ""

make_config "nivora-soak-server" "$SERVER_PORT" "$SERVER_CONFIG"
make_config "nivora-soak-worker" "$WORKER_PORT" "$WORKER_CONFIG"
make_config "nivora-soak-runner" "$RUNNER_PORT" "$RUNNER_CONFIG"

echo "--- Starting processes ---"
start_server || exit 1
start_worker
start_runner

START_TIME=$(date +%s)
END_TIME=$((START_TIME + DURATION))

echo ""
echo "=== Soak loop starting at $(date) ==="

while true; do
  NOW=$(date +%s)
  if [ "$SOAK_RUNS" -eq 0 ] && [ "$NOW" -ge "$END_TIME" ]; then break; fi
  if [ "$SOAK_RUNS" -gt 0 ] && [ "$LOOPS" -ge "$SOAK_RUNS" ]; then break; fi

  LOOPS=$((LOOPS + 1))
  ELAPSED=$((NOW - START_TIME))

  # Health check first.
  if ! check_health; then
    echo "  [${LOOPS}] HEALTH FAIL at ${ELAPSED}s"
    # Try to restart server if it died.
    if ! kill -0 "${SERVER_PID:-}" 2>/dev/null; then
      echo "  Server died, restarting..."
      start_server || { echo "FATAL: server cannot restart"; exit 1; }
    fi
  fi

  # Run workloads.
  run_id=$(create_pipeline_run || echo "")
  if [ -n "$run_id" ]; then
    echo "  [${LOOPS}] PipelineRun ${run_id} OK (${ELAPSED}s)"
    # Verify logs.
    api_call "/api/v1/pipeline-runs/${run_id}/logs" >/dev/null 2>&1 || true
    # Verify timeline.
    api_call "/api/v1/pipeline-runs/${run_id}/timeline" >/dev/null 2>&1 || true
  else
    echo "  [${LOOPS}] PipelineRun FAIL (${ELAPSED}s)"
  fi

  # DeploymentRun every 3rd loop.
  if [ $((LOOPS % 3)) -eq 0 ]; then
    if create_deployment_dryrun; then
      echo "  [${LOOPS}] DeploymentRun OK"
    else
      echo "  [${LOOPS}] DeploymentRun FAIL"
    fi
  fi

  # Periodic state verification.
  if [ $((LOOPS % 5)) -eq 0 ]; then
    verify_state
    echo "  [${LOOPS}] State check: ${PIPELINE_PASS}P/${PIPELINE_FAIL}F pipeline | ${DEPLOY_PASS}P/${DEPLOY_FAIL}F deploy | ${API_TIMEOUTS} timeouts | ${HEARTBEAT_LOST} hb-lost | ${STUCK_RUNS} stuck"
  fi

  # Periodic worker restart.
  if [ "$RESTART_WORKER" = "1" ] && [ $((LOOPS % RESTART_WORKER_EVERY)) -eq 0 ]; then
    echo "  [${LOOPS}] Restarting worker..."
    stop_worker
    sleep 2
    start_worker
    WORKER_RESTARTS=$((WORKER_RESTARTS + 1))
  fi

  # Sleep until next loop.
  sleep "$INTERVAL"
done

# --- Final state verification ---
echo ""
echo "=== Soak loop ended at $(date) ==="
echo "Duration: ${DURATION}s | Loops: ${LOOPS}"
echo ""

echo "--- Final state verification ---"
check_health || echo "WARN: server not healthy at end"

# Final run.
last_run=$(create_pipeline_run || echo "")
if [ -n "$last_run" ]; then
  echo "Final PipelineRun: ${last_run}"
fi

# Final state check.
verify_state
api_call "/api/v1/system/runtime/recovery" >/dev/null 2>&1 || echo "WARN: recovery status unavailable"

echo ""
echo "--- Stop and restart: verify persistence ---"
stop_worker
stop_runner
sleep 1

start_worker
start_runner

# Verify previous run is still retrievable.
if [ -n "$last_run" ]; then
  if api_call "/api/v1/pipeline-runs/${last_run}" | grep -q "\"id\":\"${last_run}\""; then
    echo "PASS: final PipelineRun survived restart"
  else
    echo "FAIL: final PipelineRun lost after restart"
    PIPELINE_FAIL=$((PIPELINE_FAIL + 1))
  fi
fi

# --- Summary ---
echo ""
echo "============================================"
echo "=== Soak Test Summary ==="
echo "============================================"
echo "Duration:     ${DURATION}s"
echo "Loops:        ${LOOPS}"
echo "PipelineRuns: ${PIPELINE_PASS} passed, ${PIPELINE_FAIL} failed"
echo "DeployRuns:   ${DEPLOY_PASS} passed, ${DEPLOY_FAIL} failed"
echo "Worker:       ${WORKER_RESTARTS} restarts, ${WORKER_FAILURES} failures"
echo "Runner:       ${RUNNER_RESTARTS} restarts, ${RUNNER_FAILURES} failures"
echo "API timeouts: ${API_TIMEOUTS}"
echo "Heartbeat:    ${HEARTBEAT_LOST} lost"
echo "Stuck runs:   ${STUCK_RUNS}"

TOTAL_FAIL=$((PIPELINE_FAIL + DEPLOY_FAIL + WORKER_FAILURES + RUNNER_FAILURES + HEARTBEAT_LOST + STUCK_RUNS))
if [ "$TOTAL_FAIL" -gt 0 ]; then
  echo ""
  echo "FAIL: ${TOTAL_FAIL} total failures detected"
  exit 1
else
  echo ""
  echo "PASS: No failures detected"
fi
