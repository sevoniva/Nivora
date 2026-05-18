#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
cd "$ROOT_DIR"

if [ "${NIVORA_ALLOW_LOCAL_APPLY:-}" != "true" ]; then
  echo "set NIVORA_ALLOW_LOCAL_APPLY=true to run local apply smoke test" >&2
  exit 1
fi

echo "==> Running explicit local YAML apply through the no-op manifest client"
go run ./cmd/nivora deployment apply --local examples/deployments/yaml-apply-local.yaml --confirm | grep 'Status: Succeeded' >/dev/null

echo "local apply smoke test passed"
