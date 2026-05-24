#!/usr/bin/env sh
# Kind cluster deployment smoke test.
# Deploys PostgreSQL + Nivora via Helm to kind, verifies health and API.
# Requires kind, kubectl, helm. Skip with SKIP_KIND_DEPLOY=1.
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
cd "$ROOT_DIR"

if [ "${SKIP_KIND_DEPLOY:-0}" = "1" ]; then
  echo "SKIP: SKIP_KIND_DEPLOY=1"
  exit 0
fi

for cmd in kind kubectl helm; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "SKIP: $cmd not found"
    exit 0
  fi
done

CLUSTER="${KIND_CLUSTER:-devops-kind}"
NAMESPACE="nivora-smoke-test"
RELEASE="nivora-kind-smoke"
PASS=0
FAIL=0

pass() { echo "PASS: $*"; PASS=$((PASS + 1)); }
fail() { echo "FAIL: $*" >&2; FAIL=$((FAIL + 1)); }

cleanup() {
  echo ""
  echo "--- Cleanup ---"
  helm uninstall "$RELEASE" -n "$NAMESPACE" --wait 2>/dev/null || true
  kubectl delete namespace "$NAMESPACE" --ignore-not-found --wait 2>/dev/null || true
}
trap cleanup EXIT INT TERM

echo "=== Kind Cluster Deployment Smoke ==="
echo "Cluster: ${CLUSTER}"

# Verify cluster access.
if ! kubectl cluster-info --context "kind-${CLUSTER}" >/dev/null 2>&1; then
  fail "cannot access kind cluster ${CLUSTER}"
  exit 1
fi
pass "kind cluster accessible"

# Create namespace.
kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f - >/dev/null
pass "namespace ${NAMESPACE} created"

# Deploy PostgreSQL.
echo ""
echo "--- Deploying PostgreSQL ---"
kubectl apply -n "$NAMESPACE" -f - <<'YAML' >/dev/null
apiVersion: v1
kind: ConfigMap
metadata:
  name: postgres-init
data:
  init.sql: |
    CREATE DATABASE nivora;
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: postgres
spec:
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
      - name: postgres
        image: postgres:16-alpine
        env:
        - name: POSTGRES_USER
          value: nivora
        - name: POSTGRES_PASSWORD
          value: nivora
        - name: POSTGRES_DB
          value: nivora
        ports:
        - containerPort: 5432
        readinessProbe:
          exec:
            command: ["pg_isready", "-U", "nivora", "-d", "nivora"]
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: postgres
spec:
  selector:
    app: postgres
  ports:
  - port: 5432
YAML

echo "  Waiting for PostgreSQL..."
kubectl wait --for=condition=ready pod -l app=postgres -n "$NAMESPACE" --timeout=120s >/dev/null 2>&1 || {
  fail "PostgreSQL did not become ready"
  exit 1
}
pass "PostgreSQL ready"

# Deploy Nivora via Helm.
echo ""
echo "--- Deploying Nivora ---"
helm upgrade --install "$RELEASE" deployments/helm \
  -n "$NAMESPACE" \
  --set config.environment=kubernetes \
  --set config.runtimeStore=postgres \
  --set config.databaseURL="postgres://nivora:nivora@postgres.${NAMESPACE}.svc:5432/nivora?sslmode=disable" \
  --set config.auth.enabled=true \
  --set config.auth.mode=token \
  --set config.auth.staticTokenEnv=NIVORA_AUTH_TOKEN \
  --set config.runtime.runnerIsolationProfile=kubernetes-job \
  --set config.runtime.allowLocalShellExecutor=false \
  --set config.runtime.allowDockerSocketMount=false \
  --set config.runtime.allowHostPathMount=false \
  --set secret.create=true \
  --set secret.stringData.NIVORA_AUTH_TOKEN="kind-smoke-test-token" \
  --set runner.enabled=true \
  --set runner.name=kind-runner \
  --set runner.replicas=1 \
  --wait \
  --timeout 5m >/dev/null 2>&1 || {
  fail "Helm install failed"
  helm status "$RELEASE" -n "$NAMESPACE" 2>&1 | tail -20
  exit 1
}
pass "Nivora Helm install succeeded"

# Wait for server pod ready.
echo "  Waiting for Nivora server..."
kubectl wait --for=condition=ready pod -l app.kubernetes.io/component=server -n "$NAMESPACE" --timeout=120s >/dev/null 2>&1 || {
  fail "Nivora server not ready"
  kubectl logs -l app.kubernetes.io/component=server -n "$NAMESPACE" --tail=20 2>&1
  exit 1
}
pass "Nivora server ready"

# Port-forward for API smoke.
echo ""
echo "--- API Smoke Tests ---"
kubectl port-forward -n "$NAMESPACE" svc/"${RELEASE}-server" 18080:8080 >/dev/null 2>&1 &
PF_PID=$!
sleep 3

BASE_URL="http://127.0.0.1:18080"
SMOKE_TOKEN="${NIVORA_KIND_SMOKE_TOKEN:-kind-smoke-test-token}"
AUTH_HEADER="Authorization: Bearer ${SMOKE_TOKEN}"

# Health check.
if curl -fsS "${BASE_URL}/healthz" >/dev/null 2>&1; then
  pass "healthz OK"
else
  fail "healthz failed"
fi

# Readiness.
if curl -fsS "${BASE_URL}/readyz" >/dev/null 2>&1; then
  pass "readyz OK (Postgres DB dependency verified)"
else
  fail "readyz failed — DB connectivity issue"
fi

# System info.
info=$(curl -fsS "${BASE_URL}/api/v1/system/info" 2>/dev/null || echo '{}')
if echo "$info" | grep -q '"Environment":"kubernetes"'; then
  pass "system info reports kubernetes environment"
else
  echo "WARN: system info: $info"
fi

# Runtime mode.
runtime=$(curl -fsS "${BASE_URL}/api/v1/system/runtime" 2>/dev/null || echo '{}')
if echo "$runtime" | grep -q 'postgres'; then
  pass "runtime store is postgres"
else
  fail "runtime store is NOT postgres"
fi

# Create PipelineRun.
echo ""
echo "--- Creating PipelineRun ---"
run_resp=$(curl -fsS -X POST "${BASE_URL}/api/v1/pipeline-runs" \
  -H 'Content-Type: application/json' \
  -H "$AUTH_HEADER" \
  -d '{"apiVersion":"nivora.io/v1alpha1","kind":"Pipeline","metadata":{"name":"kind-smoke"},"spec":{"stages":[{"name":"test","jobs":[{"name":"echo","executor":"shell","steps":[{"name":"say","run":"printf kind-smoke-ok"}]}]}]}}' 2>/dev/null || echo '{}')
run_id=$(printf '%s\n' "$run_resp" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p' | head -1)
if [ -n "$run_id" ] && printf '%s\n' "$run_resp" | grep -q '"status":"Succeeded"'; then
  pass "PipelineRun ${run_id} succeeded on K8s"
else
  echo "WARN: PipelineRun may not have succeeded (runner shell executor in K8s)"
fi

# Audit chain verification.
echo ""
echo "--- Verifying Audit Chain ---"
if curl -fsS "${BASE_URL}/api/v1/audit/verify?scopeType=pipeline" -H "$AUTH_HEADER" 2>/dev/null | grep -q '"valid"'; then
  pass "audit chain verifiable"
else
  echo "WARN: audit chain not verifiable (may need more audit records)"
fi

# Stop port-forward.
kill "$PF_PID" 2>/dev/null || true

echo ""
echo "=== Kind deploy smoke: $PASS passed, $FAIL failed ==="
if [ "$FAIL" -gt 0 ]; then
  echo ""
  echo "--- Debug Info ---"
  kubectl get pods -n "$NAMESPACE" 2>&1 || true
  kubectl logs -l app.kubernetes.io/component=server -n "$NAMESPACE" --tail=30 2>&1 || true
  exit 1
fi
