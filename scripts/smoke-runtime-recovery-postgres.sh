#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
cd "$ROOT_DIR"

if [ "${NIVORA_RUN_POSTGRES_INTEGRATION:-}" != "true" ]; then
  echo "Skipping PostgreSQL runtime recovery smoke; set NIVORA_RUN_POSTGRES_INTEGRATION=true and DATABASE_URL to run it."
  exit 0
fi

if [ -z "${DATABASE_URL:-}" ]; then
  echo "DATABASE_URL is required for PostgreSQL runtime recovery smoke." >&2
  exit 1
fi

echo "Running PostgreSQL runtime recovery integration tests."
go test -p 1 -run 'TestPostgresIntegration' ./internal/adapters/repository/postgres ./internal/app/runtime
