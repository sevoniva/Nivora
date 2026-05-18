#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
cd "$ROOT_DIR"

echo "==> Running local PipelineRun smoke test"
output="$(go run ./cmd/nivora pipeline run --local examples/pipelines/simple-shell.yaml)"
printf '%s\n' "$output"

printf '%s\n' "$output" | grep 'PipelineRun:' >/dev/null || {
  echo "missing PipelineRun ID in CLI output" >&2
  exit 1
}
printf '%s\n' "$output" | grep 'Status: Succeeded' >/dev/null || {
  echo "local PipelineRun did not succeed" >&2
  exit 1
}
printf '%s\n' "$output" | grep 'hello from nivora' >/dev/null || {
  echo "local PipelineRun logs were not printed" >&2
  exit 1
}

echo "local PipelineRun smoke test passed"
