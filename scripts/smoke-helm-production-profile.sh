#!/usr/bin/env bash
set -euo pipefail

if ! command -v helm >/dev/null 2>&1; then
  echo "helm not found; skipping production Helm profile smoke check"
  exit 0
fi

tmp_default="$(mktemp)"
tmp_prod="$(mktemp)"
trap 'rm -f "$tmp_default" "$tmp_prod"' EXIT

helm template nivora deployments/helm >"$tmp_default"
helm template nivora deployments/helm -f deployments/helm/values-production.yaml >"$tmp_prod"

grep -q 'environment: "production"' "$tmp_prod"
grep -q 'runtime_store: "postgres"' "$tmp_prod"
grep -q 'enabled: true' "$tmp_prod"
grep -q 'allow_local_shell_executor: false' "$tmp_prod"
grep -q 'allow_privileged_executor: false' "$tmp_prod"
grep -q 'allow_remote_host_deploy: false' "$tmp_prod"
grep -q 'allow_kubernetes_apply: false' "$tmp_prod"
grep -q 'allow_argo_sync: false' "$tmp_prod"
grep -q 'allow_insecure_registry: false' "$tmp_prod"

if grep -q 'runtime_store: "memory"' "$tmp_prod"; then
  echo "production Helm profile rendered memory runtime store"
  exit 1
fi
if grep -q 'placeholder: ""' "$tmp_prod"; then
  echo "production Helm profile rendered an empty placeholder secret"
  exit 1
fi
if grep -E 'password[[:space:]]*[:=][[:space:]]*["'\''][^"'\'']+["'\'']' "$tmp_prod" >/dev/null; then
  echo "production Helm profile rendered an inline password-like value"
  exit 1
fi

echo "production Helm profile smoke check passed"
