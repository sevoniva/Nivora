#!/usr/bin/env sh
# Production install profile smoke validation.
# Validates that both Helm and Docker Compose production profiles are safe.
# Sources verify-helm-safety.sh for Helm checks and adds Compose checks.
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
cd "$ROOT_DIR"

PASS=0
FAIL=0

pass() { echo "PASS: $*"; PASS=$((PASS + 1)); }
fail() { echo "FAIL: $*" >&2; FAIL=$((FAIL + 1)); }

echo "=== Production Install Smoke ==="
echo ""

# --- Helm production validation ---
echo "--- Helm production profile ---"

if command -v helm >/dev/null 2>&1; then
  CHART_DIR="deployments/helm"
  PROD_OUT=$(helm template nivora "$CHART_DIR" -f "$CHART_DIR/values-production.yaml" 2>/dev/null)

  # Runtime store must be postgres.
  if echo "$PROD_OUT" | grep -q 'runtime_store: "postgres"'; then
    pass "Helm production uses postgres runtime store"
  else
    fail "Helm production must use postgres runtime store"
  fi

  # Auth must be enabled.
  if echo "$PROD_OUT" | grep -q 'enabled: true' && echo "$PROD_OUT" | grep -q 'mode: "token"'; then
    pass "Helm production enables auth"
  else
    fail "Helm production must enable auth"
  fi

  # All unsafe flags must be false.
  for flag in allow_local_shell_executor allow_privileged_executor allow_remote_host_deploy allow_kubernetes_apply allow_argo_sync allow_insecure_registry; do
    if echo "$PROD_OUT" | grep -q "${flag}: true"; then
      fail "Helm production has unsafe flag: $flag=true"
    else
      pass "Helm production $flag=false"
    fi
  done

  # No inline secrets.
  if echo "$PROD_OUT" | grep -qiE 'password: "[^"]+"|secret: "[^"]{8,}"'; then
    fail "Helm production has possible inline secret"
  else
    pass "Helm production has no inline secrets"
  fi

  # Environment must be production.
  if echo "$PROD_OUT" | grep -q 'environment: "production"'; then
    pass "Helm production environment is production"
  else
    fail "Helm production environment must be production"
  fi

  # Optional helm lint.
  if helm lint "$CHART_DIR" >/dev/null 2>&1; then
    pass "Helm lint passed"
  else
    echo "WARN: helm lint not available or failed"
  fi
else
  echo "SKIP: helm not found"
fi

# --- Docker Compose production validation ---
echo ""
echo "--- Docker Compose production profile ---"

COMPOSE_FILE="deployments/docker-compose/docker-compose.production.example.yaml"
if [ -f "$COMPOSE_FILE" ]; then
  compose_text=$(cat "$COMPOSE_FILE")

  # Must require env placeholders for sensitive values.
  if echo "$compose_text" | grep -q 'NIVORA_POSTGRES_PASSWORD'; then
    pass "Compose production requires NIVORA_POSTGRES_PASSWORD env placeholder"
  else
    fail "Compose production must use env placeholder for Postgres password"
  fi

  if echo "$compose_text" | grep -q 'NIVORA_AUTH_TOKEN'; then
    pass "Compose production requires NIVORA_AUTH_TOKEN env placeholder"
  else
    fail "Compose production must use env placeholder for auth token"
  fi

  if echo "$compose_text" | grep -q 'NIVORA_PRODUCTION_CONFIG'; then
    pass "Compose production requires external production config"
  else
    fail "Compose production must mount external production config"
  fi

  # Must not have inline credentials.
  if echo "$compose_text" | grep -qiE 'POSTGRES_PASSWORD: (?!\$\{)[a-z0-9]{8,}'; then
    fail "Compose production must not have inline Postgres password"
  else
    pass "Compose production has no inline credentials"
  fi

  # Must not use trust auth.
  if echo "$compose_text" | grep -qi 'trust'; then
    fail "Compose production must not use trust auth"
  else
    pass "Compose production does not use trust auth"
  fi

  # Image tag check.
  if echo "$compose_text" | grep -q '1.0.0'; then
    echo "WARN: Compose production image tag is 1.0.0; should match VERSION"
  else
    pass "Compose production image tag is not 1.0.0"
  fi
else
  echo "SKIP: production compose file not found at $COMPOSE_FILE"
fi

# --- Summary ---
echo ""
echo "=== Production install smoke: $PASS passed, $FAIL failed ==="
if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
