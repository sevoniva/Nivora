#!/usr/bin/env sh
# Runbook: Database health and connectivity check.
# Read-only. Never mutates state unless reconcile is explicitly invoked.
set -eu

SERVER="${NIVORA_SERVER_URL:-http://localhost:8080}"

echo "=== Database Health Check ==="
echo "Server: ${SERVER}"
echo ""

# 1. System diagnostics.
echo "--- System Diagnostics ---"
diag=$(curl -fsS "${SERVER}/api/v1/system/diagnostics" 2>/dev/null || echo '{}')
echo "$diag" | sed 's/^/  /'

# Check runtime mode.
runtime_mode=$(echo "$diag" | grep -o '"runtimeStore":"[^"]*"' | sed 's/"runtimeStore":"//;s/"//')
if [ "$runtime_mode" = "postgres" ]; then
  echo "  ✅ Runtime store: postgres"
elif [ "$runtime_mode" = "memory" ]; then
  echo "  ⚠️  Runtime store: memory — state will be lost on restart"
  echo "  → Production should use runtime_store: postgres"
else
  echo "  Runtime store: ${runtime_mode:-unknown}"
fi

# 2. System info.
echo ""
echo "--- System Info ---"
info=$(curl -fsS "${SERVER}/api/v1/system/info" 2>/dev/null || echo '{}')
echo "$info" | sed 's/^/  /'

# 3. Database dependency check via readyz.
echo ""
echo "--- Database Connectivity ---"
ready=$(curl -fsS "${SERVER}/readyz" 2>/dev/null || echo '{}')
if echo "$ready" | grep -q '"status":"ready"'; then
  echo "  ✅ Server reports ready — database is accessible"
elif echo "$ready" | grep -q '"status":"degraded"'; then
  echo "  ⚠️  Server reports degraded — database may be unavailable"
  echo "  → Check PostgreSQL process: pg_isready"
  echo "  → Check DATABASE_URL configuration"
  echo "  → Runbook: docs/operations/runbooks/db-unavailable.md"
  echo ""
  # Show degraded checks.
  echo "$ready" | grep -o '"name":"[^"]*","status":"[^"]*"' | sed 's/^/  /'
else
  echo "  ⚠️  Cannot determine database status from readyz"
fi

# 4. Migration status.
echo ""
echo "--- Migration Status ---"
runtime_info=$(curl -fsS "${SERVER}/api/v1/system/runtime" 2>/dev/null || echo '{}')
if echo "$runtime_info" | grep -q 'migration'; then
  echo "$runtime_info" | grep -o '"migration[^"]*"[^,}]*' | sed 's/^/  /'
else
  echo "  Migration status not directly available via API"
  echo "  → Check migration files: ls internal/infra/migration/"
  echo "  → Run migrations: make migrate-up"
fi

# 5. Recommended actions.
echo ""
echo "--- Recommended Actions ---"
if [ "$runtime_mode" = "memory" ]; then
  echo "  ⚠️  Switch to postgres runtime store for production"
  echo "  1. Set database.runtime_store: postgres in config"
  echo "  2. Run migrations: make migrate-up"
  echo "  3. Restart server"
fi
if echo "$ready" | grep -q '"status":"degraded"'; then
  echo "  ⚠️  Database connectivity issue detected"
  echo "  1. Check PostgreSQL is running"
  echo "  2. Verify DATABASE_URL is correct"
  echo "  3. Check network/firewall rules"
  echo "  4. Runbook: docs/operations/runbooks/db-unavailable.md"
fi

echo ""
echo "=== Database check complete ==="
