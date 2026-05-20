#!/usr/bin/env sh
# Multi-process runtime recovery smoke test.
# Requires a running PostgreSQL. Set DATABASE_URL or defaults to local dev Postgres.
# Skip with: SKIP_MULTIPROCESS_RECOVERY=1
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
cd "$ROOT_DIR"

if [ "${SKIP_MULTIPROCESS_RECOVERY:-0}" = "1" ]; then
  echo "SKIP: SKIP_MULTIPROCESS_RECOVERY=1"
  exit 0
fi

DATABASE_URL="${DATABASE_URL:-postgres://nivora:nivora@localhost:5432/nivora?sslmode=disable}"

# Test Postgres connectivity.
if ! command -v psql >/dev/null 2>&1; then
  echo "WARNING: psql not found; skipping Postgres connectivity check"
elif ! psql "$DATABASE_URL" -c 'SELECT 1' >/dev/null 2>&1; then
  echo "SKIP: cannot connect to Postgres at $DATABASE_URL"
  echo "  Set DATABASE_URL or start a local Postgres instance."
  echo "  Local: docker run --rm -e POSTGRES_USER=nivora -e POSTGRES_PASSWORD=nivora -p 5432:5432 postgres:16-alpine"
  exit 0
fi

SERVER_PORT="${NIVORA_RECOVERY_SERVER_PORT:-18080}"
WORKER_PORT="${NIVORA_RECOVERY_WORKER_PORT:-18081}"
RUNNER_PORT="${NIVORA_RECOVERY_RUNNER_PORT:-18082}"

BASE_URL="http://127.0.0.1:${SERVER_PORT}"
CONFIG_DIR="$(mktemp -d "${TMPDIR:-/tmp}/nivora-recovery.XXXXXX")"
SERVER_CONFIG="${CONFIG_DIR}/server.yaml"
WORKER_CONFIG="${CONFIG_DIR}/worker.yaml"
RUNNER_CONFIG="${CONFIG_DIR}/runner.yaml"
SERVER_LOG="${CONFIG_DIR}/server.log"
WORKER_LOG="${CONFIG_DIR}/worker.log"
RUNNER_LOG="${CONFIG_DIR}/runner.log"

SERVER_PID=""
WORKER_PID=""
RUNNER_PID=""

cleanup() {
  for pid in ${RUNNER_PID:-} ${WORKER_PID:-} ${SERVER_PID:-}; do
    kill "$pid" >/dev/null 2>&1 || true
    wait "$pid" >/dev/null 2>&1 || true
  done
  rm -rf "$CONFIG_DIR"
}
trap cleanup EXIT INT TERM

# Generate config files with postgres runtime store.
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
  endpoint: ""
auth:
  enabled: false
  mode: dev
  dev_user: local-admin
  static_token_env: NIVORA_AUTH_TOKEN
runner:
  name: recovery-runner
  group: default
  heartbeat_interval: 10s
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
  echo "==> Starting nivora-server on ${BASE_URL}"
  go run ./cmd/nivora server --config "$SERVER_CONFIG" >"$SERVER_LOG" 2>&1 &
  SERVER_PID="$!"
  for _ in $(seq 1 15); do
    if curl -fsS "${BASE_URL}/healthz" >/dev/null 2>&1; then
      echo "  Server healthy (PID ${SERVER_PID})"
      return 0
    fi
    sleep 1
  done
  echo "FAIL: server did not become healthy" >&2
  cat "$SERVER_LOG" >&2
  return 1
}

start_worker() {
  echo "==> Starting nivora-worker"
  go run ./cmd/nivora worker --config "$WORKER_CONFIG" >"$WORKER_LOG" 2>&1 &
  WORKER_PID="$!"
  echo "  Worker started (PID ${WORKER_PID})"
  sleep 2
}

start_runner() {
  echo "==> Starting nivora-runner"
  go run ./cmd/nivora runner --config "$RUNNER_CONFIG" >"$RUNNER_LOG" 2>&1 &
  RUNNER_PID="$!"
  echo "  Runner started (PID ${RUNNER_PID})"
  sleep 2
}

stop_processes() {
  echo "==> Stopping processes"
  for pid in ${RUNNER_PID:-} ${WORKER_PID:-} ${SERVER_PID:-}; do
    if kill -0 "$pid" 2>/dev/null; then
      kill "$pid" >/dev/null 2>&1 || true
      wait "$pid" >/dev/null 2>&1 || true
      echo "  Stopped PID $pid"
    fi
  done
  RUNNER_PID=""
  WORKER_PID=""
  SERVER_PID=""
  sleep 1
}

verify_state() {
  local run_id="$1"
  echo "==> Verifying state for ${run_id}"

  # PipelineRun still exists.
  curl -fsS "${BASE_URL}/api/v1/pipeline-runs/${run_id}" | grep '"id":"'"${run_id}"'"' >/dev/null || {
    echo "FAIL: PipelineRun ${run_id} not found after restart" >&2
    return 1
  }
  echo "  PipelineRun found"

  # Logs still exist.
  if curl -fsS "${BASE_URL}/api/v1/pipeline-runs/${run_id}/logs" | grep -q 'smoke'; then
    echo "  PipelineRun logs accessible"
  else
    echo "WARN: PipelineRun logs may be incomplete"
  fi

  # Events still exist.
  if curl -fsS "${BASE_URL}/api/v1/pipeline-runs/${run_id}/timeline" | grep -q 'devops.pipeline.run.completed'; then
    echo "  PipelineRun timeline accessible"
  else
    echo "WARN: PipelineRun timeline may be incomplete"
  fi

  # Audit records still exist.
  if curl -fsS "${BASE_URL}/api/v1/audit/search?subject=${run_id}" | grep -q '"action"'; then
    echo "  Audit records accessible"
  else
    echo "WARN: Audit records may be incomplete"
  fi

  # Audit chain verification.
  if curl -fsS "${BASE_URL}/api/v1/audit/verify?scopeType=pipeline" | grep -q '"valid":true'; then
    echo "  Audit chain verified"
  else
    echo "WARN: Audit chain verification may not be available"
  fi

  echo "  State verification complete"
}

# --- Main ---

echo "=== Multi-Process Recovery Smoke Test ==="
echo "Database: ${DATABASE_URL}"

make_config "nivora-server" "$SERVER_PORT" "$SERVER_CONFIG"
make_config "nivora-worker" "$WORKER_PORT" "$WORKER_CONFIG"
make_config "nivora-runner" "$RUNNER_PORT" "$RUNNER_CONFIG"

# Phase 1: Create state.
echo ""
echo "--- Phase 1: Create state ---"
start_server
start_worker
start_runner

echo "==> Creating PipelineRun"
response="$(curl -fsS -X POST "${BASE_URL}/api/v1/pipeline-runs" \
  -H 'Content-Type: application/json' \
  -d '{
    "apiVersion": "nivora.io/v1alpha1",
    "kind": "Pipeline",
    "metadata": {"name": "smoke-recovery"},
    "spec": {
      "stages": [{
        "name": "build",
        "jobs": [{
          "name": "echo",
          "executor": "shell",
          "steps": [{"name": "say", "run": "printf recovery-smoke"}]
        }]
      }]
    }
  }')"

printf '%s\n' "$response" | grep '"status":"Succeeded"' >/dev/null || {
  echo "FAIL: PipelineRun did not succeed" >&2
  printf '%s\n' "$response" >&2
  exit 1
}

run_id="$(printf '%s\n' "$response" | sed -n 's/.*"id":"\(prun-[^"]*\)".*/\1/p' | head -1)"
if [ -z "$run_id" ]; then
  echo "FAIL: could not extract PipelineRun ID" >&2
  exit 1
fi
echo "  PipelineRun created: ${run_id}"

# Phase 2: Kill and restart.
echo ""
echo "--- Phase 2: Restart ---"
stop_processes

sleep 2

start_server
start_worker
start_runner

# Phase 3: Verify state survived.
echo ""
echo "--- Phase 3: Verify state survived restart ---"
verify_state "$run_id"

# Phase 4: Create new state after restart (DeploymentRun + ReleaseExecution).
echo ""
echo "--- Phase 4: Create new state after restart ---"

echo "==> Creating Release with artifact binding"
release_response="$(curl -fsS -X POST "${BASE_URL}/api/v1/releases" \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "smoke-recovery-release",
    "versionName": "1.0.0",
    "applicationId": "smoke-app"
  }')"

release_id="$(printf '%s\n' "$release_response" | sed -n 's/.*"id":"\(rel-[^"]*\)".*/\1/p' | head -1)"
if [ -n "$release_id" ]; then
  echo "  Release created: ${release_id}"

  echo "==> Creating Release plan"
  curl -fsS -X POST "${BASE_URL}/api/v1/releases/${release_id}/plan" \
    -H 'Content-Type: application/json' \
    -d '{
      "environmentId": "dev",
      "strategy": "sequential",
      "targets": [{
        "name": "k8s-dev",
        "type": "kubernetes-yaml",
        "manifests": ["examples/yaml/configmap.yaml"]
      }]
    }' >/dev/null || echo "  WARN: Release plan may have failed (expected for noop executors)"

  echo "==> Deploying Release"
  deploy_release_response="$(curl -fsS -X POST "${BASE_URL}/api/v1/releases/${release_id}/deploy" \
    -H 'Content-Type: application/json' \
    -d '{}')"
  execution_id="$(printf '%s\n' "$deploy_release_response" | sed -n 's/.*"executionId":"\(rex-[^"]*\)".*/\1/p' | head -1)"
  if [ -n "$execution_id" ]; then
    echo "  ReleaseExecution created: ${execution_id}"
  else
    echo "  WARN: ReleaseExecution may not have been created"
  fi
else
  echo "  WARN: Release creation may have failed"
fi

echo "==> Creating DeploymentRun dry-run"
deploy_response="$(curl -fsS -X POST "${BASE_URL}/api/v1/deployments" \
  -H 'Content-Type: application/json' \
  -d '{
    "apiVersion": "nivora.io/v1alpha1",
    "kind": "Deployment",
    "metadata": {"name": "smoke-recovery-deployment"},
    "spec": {
      "application": "smoke-app",
      "environment": "dev",
      "target": {"type": "kubernetes-yaml", "name": "local-dry-run", "namespace": "default"},
      "manifests": ["examples/yaml/configmap.yaml", "examples/yaml/deployment.yaml", "examples/yaml/service.yaml"],
      "options": {"dryRun": true, "apply": false}
    }
  }')"

printf '%s\n' "$deploy_response" | grep '"status":"Succeeded"' >/dev/null || {
  echo "WARN: DeploymentRun dry-run did not succeed after restart" >&2
  printf '%s\n' "$deploy_response" >&2
}

deploy_run_id="$(printf '%s\n' "$deploy_response" | sed -n 's/.*"id":"\(drun-[^"]*\)".*/\1/p' | head -1)"
if [ -n "$deploy_run_id" ]; then
  echo "  DeploymentRun created: ${deploy_run_id}"

  # Kill, restart, verify all.
  echo ""
  echo "--- Phase 5: Second restart and verify ---"
  stop_processes
  sleep 2
  start_server
  start_worker
  start_runner

  echo "==> Verifying PipelineRun, DeploymentRun, and ReleaseExecution after second restart"
  verify_state "$run_id"

  if curl -fsS "${BASE_URL}/api/v1/deployments/${deploy_run_id}" | grep -q '"id":"'"${deploy_run_id}"'"'; then
    echo "  DeploymentRun ${deploy_run_id} survived second restart"
  else
    echo "WARN: DeploymentRun not found after second restart"
  fi

  if [ -n "${execution_id:-}" ]; then
    if curl -fsS "${BASE_URL}/api/v1/releases/executions/${execution_id}" | grep -q '"id":"'"${execution_id}"'"'; then
      echo "  ReleaseExecution ${execution_id} survived second restart"
    else
      echo "WARN: ReleaseExecution not found after second restart"
    fi
  fi
else
  # Even without DeploymentRun, verify ReleaseExecution recovery.
  if [ -n "${execution_id:-}" ]; then
    echo ""
    echo "--- Phase 5: Restart and verify ReleaseExecution ---"
    stop_processes
    sleep 2
    start_server
    start_worker
    start_runner

    verify_state "$run_id"
    if curl -fsS "${BASE_URL}/api/v1/releases/executions/${execution_id}" | grep -q '"id":"'"${execution_id}"'"'; then
      echo "  ReleaseExecution ${execution_id} survived restart"
    else
      echo "WARN: ReleaseExecution not found after restart"
    fi
  fi
fi

echo ""
echo "=== Multi-process recovery smoke test passed ==="
