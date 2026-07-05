#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SERVER_URL="${NIVORA_WEB_SMOKE_SERVER_URL:-http://127.0.0.1:8080}"
WEB_URL="${NIVORA_WEB_SMOKE_WEB_URL:-http://localhost:5173}"
TMP_DIR="$(mktemp -d)"
SERVER_LOG="$TMP_DIR/server.log"
WEB_LOG="$TMP_DIR/web.log"
SERVER_PID=""
WEB_PID=""

cleanup() {
  if [ -n "$WEB_PID" ]; then
    kill "$WEB_PID" >/dev/null 2>&1 || true
  fi
  if [ -n "$SERVER_PID" ]; then
    kill "$SERVER_PID" >/dev/null 2>&1 || true
  fi
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "[smoke-web-console] SKIPPED — $1 not found."
    exit 0
  fi
}

wait_for_url() {
  local url="$1"
  local name="$2"
  local attempts="${3:-60}"
  for _ in $(seq 1 "$attempts"); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  echo "[smoke-web-console] $name did not become ready at $url"
  if [ -s "$SERVER_LOG" ]; then
    echo "--- server log ---"
    tail -80 "$SERVER_LOG"
  fi
  if [ -s "$WEB_LOG" ]; then
    echo "--- web log ---"
    tail -80 "$WEB_LOG"
  fi
  return 1
}

assert_proxy_json_contains() {
  local path="$1"
  local marker="$2"
  local body
  body="$(curl -fsS "$WEB_URL$path")"
  case "$body" in
    *"$marker"*) ;;
    *)
      echo "[smoke-web-console] expected $path response to contain $marker"
      echo "$body"
      exit 1
      ;;
  esac
}

require_cmd curl
require_cmd npm

echo "==> Running web console smoke test"

if curl -fsS "$SERVER_URL/api/v1/version" >/dev/null 2>&1; then
  echo "Using existing Nivora server at $SERVER_URL"
else
  echo "Starting Nivora server for web smoke"
  (cd "$ROOT_DIR" && go run ./cmd/nivora server --config configs/server.yaml >"$SERVER_LOG" 2>&1) &
  SERVER_PID="$!"
  wait_for_url "$SERVER_URL/api/v1/version" "Nivora server"
fi

if [ ! -x "$ROOT_DIR/web/node_modules/.bin/vite" ]; then
  (cd "$ROOT_DIR/web" && npm ci)
fi

if curl -fsS "$WEB_URL" >/dev/null 2>&1; then
  echo "Using existing web console at $WEB_URL"
else
  echo "Starting Vite web console for smoke"
  (cd "$ROOT_DIR/web" && NIVORA_WEB_PROXY_TARGET="$SERVER_URL" npm run dev -- --host localhost --strictPort >"$WEB_LOG" 2>&1) &
  WEB_PID="$!"
  wait_for_url "$WEB_URL" "web console"
fi

html="$(curl -fsS "$WEB_URL")"
case "$html" in
  *"Nivora Control Plane"*|*"id=\"root\""*) ;;
  *)
    echo "[smoke-web-console] web root did not look like the Nivora console"
    exit 1
    ;;
esac

version_json="$(curl -fsS "$WEB_URL/api/v1/version")"
case "$version_json" in
  *"version"*) ;;
  *)
    echo "[smoke-web-console] Vite proxy did not return API version JSON"
    echo "$version_json"
    exit 1
    ;;
esac

assert_proxy_json_contains "/api/v1/artifacts" "artifacts"
assert_proxy_json_contains "/api/v1/policies/results" "results"
assert_proxy_json_contains "/api/v1/evidence/bundles" "bundles"
assert_proxy_json_contains "/api/v1/integrations" "integrations"
assert_proxy_json_contains "/api/v1/plugins" "artifact-oci"
assert_proxy_json_contains "/api/v1/system/runtime" "runtime_mode"

if grep -qiE "Cannot find package 'react-refresh'|\\[plugin:vite:react-babel\\]" "$WEB_LOG" 2>/dev/null; then
  echo "[smoke-web-console] Vite log contains React plugin dependency error"
  cat "$WEB_LOG"
  exit 1
fi

echo "web console smoke test passed"
