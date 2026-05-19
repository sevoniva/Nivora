#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
cd "$ROOT_DIR"

echo "==> Validating example YAML, manifest references, and migration files"
GOPROXY="${GOPROXY:-https://proxy.golang.org,direct}" go test ./test/quality

echo "example validation passed"
