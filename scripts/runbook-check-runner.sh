#!/usr/bin/env sh
# Runbook: Runner fleet health check.
# Read-only. Never mutates state.
set -eu

SERVER="${NIVORA_SERVER_URL:-http://localhost:8080}"

echo "=== Runner Fleet Health Check ==="
echo "Server: ${SERVER}"
echo ""

# 1. List runners.
echo "--- Registered Runners ---"
runners=$(curl -fsS "${SERVER}/api/v1/runners" 2>/dev/null || echo '[]')
runner_count=$(echo "$runners" | grep -o '"id":"runner-[^"]*"' | wc -l | tr -d ' ')
echo "  Total registered: $runner_count"

if [ "$runner_count" -eq 0 ]; then
  echo "  ⚠️  No runners registered"
  echo "  → Register a runner: curl -X POST ${SERVER}/api/v1/runners/register"
  echo "  → Runbook: docs/operations/runbooks/offline-runner.md"
  exit 1
fi

# 2. Runner status summary.
echo ""
echo "--- Runner Status ---"
offline_count=0
active_count=0
echo "$runners" | grep -o '"status":"[^"]*"' | sort | uniq -c | while read -r count status; do
  echo "  $count runners: $status"
done

# Check for offline runners.
offline_count=$(echo "$runners" | grep -o '"status":"Offline"' | wc -l | tr -d ' ')
if [ "$offline_count" -gt 0 ]; then
  echo ""
  echo "  ⚠️  $offline_count offline runners detected"
  echo "  → Run: curl -X POST ${SERVER}/api/v1/runners/offline-detect"
  echo "  → Runbook: docs/operations/runbooks/offline-runner.md"
fi

# 3. Heartbeat status.
echo ""
echo "--- Heartbeat Status ---"
# Check each runner's heartbeat by looking for last-heartbeat or similar.
heartbeat_runners=$(echo "$runners" | grep -o '"id":"runner-[^"]*"' | sed 's/"id":"//;s/"//')
if [ -z "$heartbeat_runners" ]; then
  echo "  No runner IDs found"
else
  for rid in $heartbeat_runners; do
    runner_detail=$(curl -fsS "${SERVER}/api/v1/runners/${rid}" 2>/dev/null || echo '{}')
    if echo "$runner_detail" | grep -q '"lastHeartbeatAt"'; then
      hb=$(echo "$runner_detail" | grep -o '"lastHeartbeatAt":"[^"]*"' | sed 's/"lastHeartbeatAt":"//;s/"//')
      echo "  $rid: last heartbeat at $hb"
    else
      echo "  $rid: heartbeat data not available"
    fi
  done
fi

# 4. Recommended actions.
echo ""
echo "--- Recommended Actions ---"
if [ "$offline_count" -gt 0 ]; then
  echo "  1. Run offline detection: curl -X POST ${SERVER}/api/v1/runners/offline-detect"
  echo "  2. Check runner process logs for crashes"
  echo "  3. Restart runner if needed"
  echo "  4. Runbook: docs/operations/runbooks/offline-runner.md"
elif [ "$runner_count" -gt 0 ] && [ "$offline_count" -eq 0 ]; then
  echo "  ✅ All runners appear healthy"
fi

echo ""
echo "=== Runner check complete ==="
