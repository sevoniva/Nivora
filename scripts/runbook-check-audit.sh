#!/usr/bin/env sh
# Runbook: Audit integrity and tamper-evidence check.
# Read-only. Never mutates state.
set -eu

SERVER="${NIVORA_SERVER_URL:-http://localhost:8080}"

echo "=== Audit Health Check ==="
echo "Server: ${SERVER}"
echo ""

# 1. Audit chain verification for all scopes.
echo "--- Audit Hash Chain Verification ---"
SCOPES="pipeline deployment release release_execution auth credential security approval cloud"

all_valid=true
for scope in $SCOPES; do
  result=$(curl -fsS "${SERVER}/api/v1/audit/verify?scopeType=${scope}" 2>/dev/null || echo '{"valid":false,"message":"unavailable"}')
  valid=$(echo "$result" | grep -o '"valid":[a-z]*' | grep -o '[a-z]*$')
  if [ "$valid" = "true" ]; then
    echo "  ✅ $scope: chain valid"
  elif [ "$valid" = "false" ]; then
    broken=$(echo "$result" | grep -o '"firstBrokenId":"[^"]*"' | sed 's/"firstBrokenId":"//;s/"//')
    echo "  ❌ $scope: chain INVALID (first broken: ${broken:-unknown})"
    all_valid=false
  else
    echo "  ⚠️  $scope: verification unavailable (memory store or no records)"
  fi
done

echo ""
if $all_valid; then
  echo "  ✅ All available audit chains are valid"
else
  echo "  ❌ Some audit chains are broken — possible tampering detected"
  echo "  → Investigate broken records immediately"
  echo "  → Check compliance_audit_records table for tampered entries"
fi

# 2. Recent audit records.
echo ""
echo "--- Recent Audit Records ---"
audit_search=$(curl -fsS "${SERVER}/api/v1/audit/search" 2>/dev/null || echo '{}')
audit_count=$(echo "$audit_search" | grep -o '"action":"[^"]*"' | wc -l | tr -d ' ')
echo "  Recent audit entries: $audit_count"

if [ "$audit_count" -eq 0 ]; then
  echo "  ⚠️  No audit records found — audit may not be enabled or persisted"
  echo "  → Ensure database.runtime_store: postgres is configured"
fi

# 3. Evidence bundles.
echo ""
echo "--- Evidence Bundles ---"
evidence_count=0
for st in pipeline deployment release; do
  resp=$(curl -fsS "${SERVER}/api/v1/evidence/${st}/latest" 2>/dev/null || echo '{}')
  if echo "$resp" | grep -q '"id"'; then
    evidence_count=$((evidence_count + 1))
    echo "  ✅ $st evidence available"
  fi
done
if [ "$evidence_count" -eq 0 ]; then
  echo "  ⚠️  No evidence bundles found (may be expected with memory store)"
fi

# 4. Retention policy.
echo ""
echo "--- Retention Policy ---"
policy=$(curl -fsS "${SERVER}/api/v1/retention-policy?scopeType=global&scopeId=" 2>/dev/null || echo '{}')
if echo "$policy" | grep -q '"logDays"'; then
  echo "  ✅ Retention policy configured"
  echo "$policy" | sed 's/^/  /'
else
  echo "  ⚠️  No retention policy configured"
  echo "  → Set a retention policy: POST ${SERVER}/api/v1/retention-policy"
fi

# 5. Recommended actions.
echo ""
echo "--- Recommended Actions ---"
if ! $all_valid; then
  echo "  ❌ Audit chain integrity failure"
  echo "  1. Query compliance_audit_records WHERE scope_type=<scope> ORDER BY created_at"
  echo "  2. Check for tampered record_hash values"
  echo "  3. Restore from backup if tampering is confirmed"
  echo "  4. Investigate access logs for unauthorized database access"
fi
if [ "$audit_count" -eq 0 ]; then
  echo "  → Verify Postgres runtime store is configured"
  echo "  → Check that audit-producing operations have been performed"
fi

echo ""
echo "=== Audit check complete ==="
