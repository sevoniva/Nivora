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

echo "API smoke test passed"
