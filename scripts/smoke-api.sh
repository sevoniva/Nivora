#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
cd "$ROOT_DIR"

PORT="${NIVORA_SMOKE_PORT:-18080}"
BASE_URL="http://127.0.0.1:${PORT}"
CONFIG_FILE="$(mktemp "${TMPDIR:-/tmp}/nivora-smoke-server.XXXXXX.yaml")"
LOG_FILE="$(mktemp "${TMPDIR:-/tmp}/nivora-smoke-server.XXXXXX.log")"

cleanup() {
  if [ "${SERVER_PID:-}" ]; then
    kill "$SERVER_PID" >/dev/null 2>&1 || true
    wait "$SERVER_PID" >/dev/null 2>&1 || true
  fi
  rm -f "$CONFIG_FILE" "$LOG_FILE"
}
trap cleanup EXIT INT TERM

sed "s/\":8080\"/\":${PORT}\"/" configs/server.yaml > "$CONFIG_FILE"

echo "==> Starting nivora-server on ${BASE_URL}"
go run ./cmd/nivora-server "$CONFIG_FILE" >"$LOG_FILE" 2>&1 &
SERVER_PID="$!"

for _ in 1 2 3 4 5 6 7 8 9 10; do
  if curl -fsS "${BASE_URL}/healthz" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

curl -fsS "${BASE_URL}/healthz" >/dev/null || {
  echo "server did not become healthy" >&2
  cat "$LOG_FILE" >&2
  exit 1
}

echo "==> Creating PipelineRun through API"
response="$(curl -fsS -X POST "${BASE_URL}/api/v1/pipeline-runs" \
  -H 'Content-Type: application/json' \
  -d '{
    "apiVersion": "nivora.io/v1alpha1",
    "kind": "Pipeline",
    "metadata": {"name": "smoke-shell"},
    "spec": {
      "stages": [{
        "name": "build",
        "jobs": [{
          "name": "echo",
          "executor": "shell",
          "steps": [{"name": "say", "run": "printf smoke"}]
        }]
      }]
    }
  }')"

printf '%s\n' "$response" | grep '"status":"Succeeded"' >/dev/null || {
  echo "PipelineRun did not succeed" >&2
  printf '%s\n' "$response" >&2
  exit 1
}

run_id="$(printf '%s\n' "$response" | sed -n 's/.*"id":"\(prun-[^"]*\)".*/\1/p' | head -1)"
if [ -z "$run_id" ]; then
  echo "could not extract PipelineRun ID" >&2
  printf '%s\n' "$response" >&2
  exit 1
fi

echo "==> Checking logs and timeline for ${run_id}"
curl -fsS "${BASE_URL}/api/v1/pipeline-runs/${run_id}/logs" | grep 'smoke' >/dev/null || {
  echo "PipelineRun logs were not accessible" >&2
  exit 1
}
curl -fsS "${BASE_URL}/api/v1/pipeline-runs/${run_id}/timeline" | grep 'devops.pipeline.run.completed' >/dev/null || {
  echo "PipelineRun timeline was not accessible" >&2
  exit 1
}

echo "==> Creating DeploymentRun dry-run through API"
deployment_response="$(curl -fsS -X POST "${BASE_URL}/api/v1/deployments" \
  -H 'Content-Type: application/json' \
  -d '{
    "apiVersion": "nivora.io/v1alpha1",
    "kind": "Deployment",
    "metadata": {"name": "smoke-yaml-deployment"},
    "spec": {
      "application": "smoke-app",
      "environment": "dev",
      "target": {"type": "kubernetes-yaml", "name": "local-dry-run", "namespace": "default"},
      "manifests": ["examples/yaml/configmap.yaml", "examples/yaml/deployment.yaml", "examples/yaml/service.yaml"],
      "options": {"dryRun": true, "apply": false}
    }
  }')"

printf '%s\n' "$deployment_response" | grep '"status":"Succeeded"' >/dev/null || {
  echo "DeploymentRun dry-run did not succeed" >&2
  printf '%s\n' "$deployment_response" >&2
  exit 1
}

deployment_run_id="$(printf '%s\n' "$deployment_response" | sed -n 's/.*"id":"\(drun-[^"]*\)".*/\1/p' | head -1)"
if [ -z "$deployment_run_id" ]; then
  echo "could not extract DeploymentRun ID" >&2
  printf '%s\n' "$deployment_response" >&2
  exit 1
fi

echo "==> Checking deployment plan, logs, and timeline for ${deployment_run_id}"
curl -fsS "${BASE_URL}/api/v1/deployments/${deployment_run_id}/plan" | grep '"manifestCount":3' >/dev/null || {
  echo "DeploymentRun plan was not accessible" >&2
  exit 1
}
curl -fsS "${BASE_URL}/api/v1/deployments/${deployment_run_id}/resources" | grep 'ConfigMap' >/dev/null || {
  echo "DeploymentRun resources were not accessible" >&2
  exit 1
}
curl -fsS "${BASE_URL}/api/v1/deployments/${deployment_run_id}/logs" | grep 'dry-run validation completed' >/dev/null || {
  echo "DeploymentRun logs were not accessible" >&2
  exit 1
}
curl -fsS "${BASE_URL}/api/v1/deployments/${deployment_run_id}/timeline" | grep 'devops.deployment.succeeded' >/dev/null || {
  echo "DeploymentRun timeline was not accessible" >&2
  exit 1
}

echo "API smoke test passed"
