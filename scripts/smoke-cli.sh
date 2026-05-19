#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
cd "$ROOT_DIR"

echo "==> Checking nivora version"
go run ./cmd/nivora version | grep 'nivora' >/dev/null || {
  echo "version command did not print version information" >&2
  exit 1
}

echo "==> Validating default server config"
go run ./cmd/nivora config validate --file configs/server.yaml | grep 'is valid' >/dev/null || {
  echo "config validation failed" >&2
  exit 1
}

echo "==> Running local PipelineRun through CLI"
go run ./cmd/nivora pipeline run --local examples/pipelines/simple-shell.yaml | grep 'Status: Succeeded' >/dev/null || {
  echo "pipeline local smoke failed" >&2
  exit 1
}

echo "==> Planning local DeploymentRun through CLI"
go run ./cmd/nivora deployment plan --local examples/deployments/yaml-dry-run.yaml | grep 'Manifests: 3' >/dev/null || {
  echo "deployment plan smoke failed" >&2
  exit 1
}

echo "==> Inspecting artifact reference through CLI"
go run ./cmd/nivora artifact inspect registry.example.com/team/demo:1.0.0 | grep 'registry.example.com/team/demo:1.0.0' >/dev/null || {
  echo "artifact inspect smoke failed" >&2
  exit 1
}

echo "CLI smoke test passed"
