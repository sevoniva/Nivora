#!/usr/bin/env sh
# Runbook: Runtime health check for PipelineRun and DeploymentRun state.
# Read-only. Never mutates state.
set -eu

SERVER="${NIVORA_SERVER_URL:-http://localhost:8080}"

echo "=== Runtime Health Check ==="
echo "Server: ${SERVER}"
echo ""

# 1. Basic health.
echo "--- Health ---"
if curl -fsS "${SERVER}/healthz" >/dev/null 2>&1; then
  echo "  healthz: OK"
else
  echo "  healthz: FAIL — server is not healthy"
  echo "  → Check server logs, restart if needed."
  echo "  → Runbook: docs/operations/runbooks/failed-server.md"
  exit 1
fi

# 2. Readiness.
if curl -fsS "${SERVER}/readyz" >/dev/null 2>&1; then
  echo "  readyz: OK"
else
  echo "  readyz: DEGRADED — server may have dependency issues"
fi

# 3. Runtime recovery status.
echo ""
echo "--- Runtime Recovery ---"
recovery=$(curl -fsS "${SERVER}/api/v1/system/runtime/recovery" 2>/dev/null || echo '{}')
echo "$recovery" | sed 's/^/  /'

queued=$(echo "$recovery" | grep -o '"queuedRuns":[0-9]*' | grep -o '[0-9]*' || echo "0")
stale=$(echo "$recovery" | grep -o '"staleRunningRuns":[0-9]*' | grep -o '[0-9]*' || echo "0")
expired=$(echo "$recovery" | grep -o '"expiredJobClaims":[0-9]*' | grep -o '[0-9]*' || echo "0")

echo ""
if [ "$queued" -gt 0 ]; then
  echo "  ⚠️  $queued queued PipelineRuns — worker may be down or overloaded"
  echo "  → Runbook: docs/operations/runbooks/stuck-pipelinerun.md"
fi
if [ "$stale" -gt 0 ]; then
  echo "  ⚠️  $stale stale running PipelineRuns — may need reconciliation"
  echo "  → Run: curl -X POST ${SERVER}/api/v1/system/runtime/reconcile"
fi
if [ "$expired" -gt 0 ]; then
  echo "  ⚠️  $expired expired job claims — runner may be offline"
  echo "  → Runbook: docs/operations/runbooks/offline-runner.md"
fi
if [ "$queued" -eq 0 ] && [ "$stale" -eq 0 ] && [ "$expired" -eq 0 ]; then
  echo "  ✅ No stuck or stale runs detected"
fi

# 4. Recent PipelineRuns.
echo ""
echo "--- Recent PipelineRuns ---"
runs=$(curl -fsS "${SERVER}/api/v1/pipeline-runs" 2>/dev/null || echo '[]')
run_count=$(echo "$runs" | grep -o '"id":"prun-[^"]*"' | wc -l | tr -d ' ')
echo "  Total: $run_count runs"

failed=$(echo "$runs" | grep -o '"status":"Failed"' | wc -l | tr -d ' ')
queued_runs=$(echo "$runs" | grep -o '"status":"Queued"' | wc -l | tr -d ' ')
if [ "$failed" -gt 0 ]; then
  echo "  ⚠️  $failed failed PipelineRuns"
  echo "  → Check logs: curl ${SERVER}/api/v1/pipeline-runs/<id>/logs"
fi
if [ "$queued_runs" -gt 0 ]; then
  echo "  ⚠️  $queued_runs queued PipelineRuns — check worker"
fi

# 5. Recent DeploymentRuns.
echo ""
echo "--- Recent DeploymentRuns ---"
deployments=$(curl -fsS "${SERVER}/api/v1/deployments" 2>/dev/null || echo '[]')
dep_count=$(echo "$deployments" | grep -o '"id":"drun-[^"]*"' | wc -l | tr -d ' ')
dep_failed=$(echo "$deployments" | grep -o '"status":"Failed"' | wc -l | tr -d ' ')
echo "  Total: $dep_count runs"
if [ "$dep_failed" -gt 0 ]; then
  echo "  ⚠️  $dep_failed failed DeploymentRuns"
  echo "  → Runbook: docs/operations/runbooks/failed-deploymentrun.md"
fi

echo ""
echo "=== Runtime check complete ==="
