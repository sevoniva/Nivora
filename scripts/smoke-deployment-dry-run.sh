#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
cd "$ROOT_DIR"

echo "==> Planning YAML deployment"
go run ./cmd/nivora deployment plan --local examples/deployments/yaml-dry-run.yaml | grep 'Manifests: 3' >/dev/null

echo "==> Running YAML deployment dry-run"
go run ./cmd/nivora deployment dry-run --local examples/deployments/yaml-dry-run.yaml | grep 'Status: Succeeded' >/dev/null

echo "deployment dry-run smoke test passed"
