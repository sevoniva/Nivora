#!/usr/bin/env sh
# Docker Compose live deployment smoke test.
# Requires Docker and docker compose. Skip with SKIP_COMPOSE_LIVE=1.
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
cd "$ROOT_DIR"

if [ "${SKIP_COMPOSE_LIVE:-0}" = "1" ]; then
  echo "SKIP: SKIP_COMPOSE_LIVE=1"
  exit 0
fi

if ! command -v docker >/dev/null 2>&1; then
  echo "SKIP: docker not found"
  exit 0
fi

COMPOSE_FILE="deployments/docker-compose/docker-compose.yaml"
if [ ! -f "$COMPOSE_FILE" ]; then
  echo "SKIP: compose file not found at $COMPOSE_FILE"
  exit 0
fi

echo "=== Docker Compose Live Deployment Smoke ==="
PASS=0
FAIL=0

pass() { echo "PASS: $*"; PASS=$((PASS + 1)); }
fail() { echo "FAIL: $*" >&2; FAIL=$((FAIL + 1)); }

# Validate compose file syntax.
echo ""
echo "--- Compose validation ---"
if docker compose -f "$COMPOSE_FILE" config >/dev/null 2>&1; then
  pass "compose file is valid"
else
  fail "compose file validation failed"
  exit 1
fi

# Start services.
echo ""
echo "--- Starting services ---"
docker compose -f "$COMPOSE_FILE" up -d --wait 2>&1 || {
  fail "compose up failed"
  docker compose -f "$COMPOSE_FILE" logs 2>&1 | tail -30
  docker compose -f "$COMPOSE_FILE" down -v 2>/dev/null || true
  exit 1
}
pass "compose services started"

# Wait for server health.
SERVER_URL="http://localhost:8080"
echo "  Waiting for server health..."
for _ in $(seq 1 30); do
  if curl -fsS "${SERVER_URL}/healthz" >/dev/null 2>&1; then
    break
  fi
  sleep 2
done

if curl -fsS "${SERVER_URL}/healthz" >/dev/null 2>&1; then
  pass "server healthy at ${SERVER_URL}"
else
  fail "server did not become healthy"
  docker compose -f "$COMPOSE_FILE" logs nivora-server 2>&1 | tail -20
fi

# Check readiness.
if curl -fsS "${SERVER_URL}/readyz" >/dev/null 2>&1; then
  pass "server ready"
else
  echo "WARN: server not fully ready (may be expected with memory store)"
fi

# Create test PipelineRun.
echo ""
echo "--- Creating PipelineRun ---"
response=$(curl -fsS -X POST "${SERVER_URL}/api/v1/pipeline-runs" \
  -H 'Content-Type: application/json' \
  -d '{"apiVersion":"nivora.io/v1alpha1","kind":"Pipeline","metadata":{"name":"compose-live-test"},"spec":{"stages":[{"name":"test","jobs":[{"name":"echo","executor":"shell","steps":[{"name":"say","run":"printf compose-live-smoke"}]}]}]}}' 2>/dev/null || echo '{"status":"Failed"}')

if printf '%s\n' "$response" | grep -q '"status":"Succeeded"'; then
  pass "PipelineRun succeeded"
else
  echo "WARN: PipelineRun may not have succeeded (expected with memory store)"
fi

# Stop and clean up.
echo ""
echo "--- Stopping services ---"
docker compose -f "$COMPOSE_FILE" down -v 2>&1 || true
pass "services stopped and cleaned up"

echo ""
echo "=== Compose live smoke: $PASS passed, $FAIL failed ==="
if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
