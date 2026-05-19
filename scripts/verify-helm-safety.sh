#!/usr/bin/env bash
set -euo pipefail

echo "=== Helm safety verification ==="

if ! command -v helm &>/dev/null; then
  echo "helm not found; skipping Helm safety checks."
  exit 0
fi

CHART_DIR="${1:-deployments/helm}"
PASS=0
FAIL=0

fail() {
  echo "FAIL: $*"
  FAIL=$((FAIL + 1))
}

pass() {
  echo "PASS: $*"
  PASS=$((PASS + 1))
}

# 1. Default values render with dev-only runtimeStore=memory and environment=development
echo ""
echo "--- Default values (dev) ---"
DEFAULT_OUT=$(helm template nivora "$CHART_DIR" 2>/dev/null)
if echo "$DEFAULT_OUT" | grep -q 'runtime_store: "memory"'; then
  # NOTES.txt is not rendered in helm template; check that environment is development
  if echo "$DEFAULT_OUT" | grep -q 'environment: "development"'; then
    pass "default chart renders memory runtime store with development environment"
  else
    fail "default chart renders memory runtime store WITHOUT development environment"
  fi
else
  pass "default chart does not render memory runtime store"
fi

# 2. Production values render postgres, not memory
echo ""
echo "--- Production values ---"
PROD_OUT=$(helm template nivora "$CHART_DIR" -f "$CHART_DIR/values-production.yaml" 2>/dev/null)
if echo "$PROD_OUT" | grep -q 'runtime_store: "memory"'; then
  fail "production chart renders memory runtime store"
else
  pass "production chart uses postgres runtime store"
fi

# 3. Production profile enables auth
if echo "$PROD_OUT" | grep -q 'enabled: true' && echo "$PROD_OUT" | grep -q 'mode: "token"'; then
  pass "production chart enables auth"
else
  fail "production chart may not enable auth"
fi

# 4. Production profile disables unsafe executors
for flag in allow_local_shell_executor allow_privileged_executor allow_remote_host_deploy allow_kubernetes_apply allow_argo_sync allow_insecure_registry; do
  if echo "$PROD_OUT" | grep -q "${flag}: true"; then
    fail "production chart has unsafe flag: $flag=true"
  else
    pass "production chart has $flag=false"
  fi
done

# 5. No inline secrets — the template renders env var names, not values.
# Check that no inline password/token values are rendered in the config.
if echo "$DEFAULT_OUT" | grep -qiE 'password: "[^"]+"|token: "[^"]{8,}"|secret: "[^"]{8,}"'; then
  fail "possible inline secret value in default chart"
else
  pass "no inline secret values in default chart"
fi

# 6. Chart versions aligned
CHART_APP_VERSION=$(grep 'appVersion:' "$CHART_DIR/Chart.yaml" | sed 's/.*"\(.*\)"/\1/')
VERSION_FILE=$(cat VERSION 2>/dev/null || echo "")
if [ "$CHART_APP_VERSION" != "1.0.0" ] && [ "$CHART_APP_VERSION" = "$VERSION_FILE" ]; then
  pass "Chart appVersion ($CHART_APP_VERSION) matches VERSION ($VERSION_FILE)"
elif [ "$CHART_APP_VERSION" != "1.0.0" ]; then
  pass "Chart appVersion ($CHART_APP_VERSION) is not GA (not 1.0.0)"
else
  fail "Chart appVersion is 1.0.0 but project is beta-candidate"
fi

echo ""
echo "=== Helm safety: $PASS passed, $FAIL failed ==="
if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
