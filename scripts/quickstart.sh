#!/usr/bin/env bash
# Quickstart: start a single-process Nivora instance for local trial.
#
# Memory runtime + dev auth + built-in local runner. No postgres, no docker.
# The server is ready in seconds. Stop with Ctrl-C or `kill $(cat .nivora/quickstart.pid)`.
#
# Usage:
#   ./scripts/quickstart.sh              # start on :18091
#   NIVORA_QUICKSTART_PORT=19000 ./scripts/quickstart.sh
#
# Once running, try:
#   go run ./cmd/nivora pipeline run --local examples/pipelines/simple-shell.yaml
#   go run ./cmd/nivora --server http://127.0.0.1:18091 pipeline definition create --project-id demo --file examples/pipelines/simple-shell.yaml
#   go run ./cmd/nivora mcp list-resources --local
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

PORT="${NIVORA_QUICKSTART_PORT:-18091}"
CONFIG="$(mktemp "${TMPDIR:-/tmp}/nivora-quickstart.XXXXXX.yaml")"
PID_FILE="$ROOT_DIR/.nivora/quickstart.pid"
LOG_FILE="$ROOT_DIR/.nivora/quickstart.log"

mkdir -p "$ROOT_DIR/.nivora"

# Generate a dev config that avoids the default :8080 conflict and uses memory runtime.
sed "s/\":8080\"/\":${PORT}\"/" configs/server.yaml > "$CONFIG"

cleanup() {
  if [ -n "${SERVER_PID:-}" ] && kill -0 "$SERVER_PID" 2>/dev/null; then
    kill "$SERVER_PID" 2>/dev/null || true
    wait "$SERVER_PID" 2>/dev/null || true
  fi
  rm -f "$CONFIG"
}
trap cleanup EXIT INT TERM

echo "==> Starting nivora-server on http://127.0.0.1:${PORT} (memory runtime, dev auth)"
go run ./cmd/nivora-server "$CONFIG" >"$LOG_FILE" 2>&1 &
SERVER_PID="$!"
echo "$SERVER_PID" > "$PID_FILE"

# Wait for readiness.
for _ in $(seq 1 60); do
  if command curl -fsS "http://127.0.0.1:${PORT}/api/v1/version" >/dev/null 2>&1; then
    break
  fi
  if ! kill -0 "$SERVER_PID" 2>/dev/null; then
    echo "[quickstart] server exited early; log:" >&2
    tail -40 "$LOG_FILE" >&2
    exit 1
  fi
  sleep 1
done

if ! command curl -fsS "http://127.0.0.1:${PORT}/api/v1/version" >/dev/null 2>&1; then
  echo "[quickstart] server did not become ready; log:" >&2
  tail -40 "$LOG_FILE" >&2
  exit 1
fi

cat <<EOF

==> Nivora is ready at http://127.0.0.1:${PORT}

Auth: dev mode (auth disabled). CLI does not need a token for local trial.
PID:  $SERVER_PID (saved to .nivora/quickstart.pid)
Log:  $LOG_FILE

Try these (in another shell):

  # Run a local PipelineRun (no server needed):
  go run ./cmd/nivora pipeline run --local examples/pipelines/simple-shell.yaml

  # Store and run a pipeline definition against the server:
  go run ./cmd/nivora --server http://127.0.0.1:${PORT} pipeline definition create \\
    --project-id demo --file examples/pipelines/simple-shell.yaml
  go run ./cmd/nivora --server http://127.0.0.1:${PORT} pipeline definition list --project-id demo

  # Validate and plan a workflow:
  go run ./cmd/nivora workflow validate --file examples/workflows/go-ci.yaml
  go run ./cmd/nivora workflow plan --file examples/workflows/go-ci.yaml

  # List MCP read-only resources:
  go run ./cmd/nivora mcp list-resources --local

  # Inspect a local repository:
  go run ./cmd/nivora repository inspect --path . --name nivora-self

Stop the server: Ctrl-C, or: kill \$(cat .nivora/quickstart.pid)

EOF

wait "$SERVER_PID"
