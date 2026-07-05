#!/usr/bin/env sh
# Release-to-release migration compatibility smoke.
#
# This is an opt-in PostgreSQL check for production-direction upgrade review.
# It does not mutate external services, does not print secrets, and skips with a
# clear reason when PostgreSQL is not available locally. CI runs it with a
# disposable PostgreSQL service.
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
cd "$ROOT_DIR"

if [ "${SKIP_UPGRADE_MIGRATION:-0}" = "1" ]; then
  echo "SKIP: SKIP_UPGRADE_MIGRATION=1"
  exit 0
fi

DATABASE_URL="${DATABASE_URL:-postgres://nivora:nivora@localhost:5432/nivora?sslmode=disable}"

redact_database_url() {
  printf '%s\n' "$1" | sed -E 's#(postgres(ql)?://)[^/@]*@#\1***@#'
}

case "$DATABASE_URL" in
  *prod*|*production*|*live*|*rds.amazonaws*|*rds.aliyuncs*|*tencentcdb*|*cloudsql*)
    if [ "${NIVORA_ALLOW_PRODUCTION_DRILL:-false}" != "true" ]; then
      echo "REFUSED: DATABASE_URL looks like a production database."
      echo "  Use a disposable database or set NIVORA_ALLOW_PRODUCTION_DRILL=true only in a controlled drill."
      exit 1
    fi
    ;;
esac

if ! command -v psql >/dev/null 2>&1; then
  echo "SKIP: psql not found; install PostgreSQL client tools to run upgrade migration compatibility smoke"
  exit 0
fi

if ! psql "$DATABASE_URL" -c 'SELECT 1' >/dev/null 2>&1; then
  echo "SKIP: cannot connect to PostgreSQL at $(redact_database_url "$DATABASE_URL")"
  exit 0
fi

version="$(tr -d '[:space:]' < VERSION)"
if [ -z "$version" ]; then
  echo "FAIL: VERSION is empty" >&2
  exit 1
fi

if ! grep -q "appVersion: \"${version}\"" deployments/helm/Chart.yaml; then
  echo "FAIL: Helm appVersion does not match VERSION ${version}" >&2
  exit 1
fi

up_count=$(find internal/infra/migration -name '*.up.sql' -type f | wc -l | tr -d ' ')
down_count=$(find internal/infra/migration -name '*.down.sql' -type f | wc -l | tr -d ' ')
if [ "$up_count" -eq 0 ] || [ "$up_count" -ne "$down_count" ]; then
  echo "FAIL: migration up/down file count mismatch: up=${up_count} down=${down_count}" >&2
  exit 1
fi

echo "=== Upgrade Migration Compatibility Smoke ==="
echo "Version: ${version}"
echo "Database: $(redact_database_url "$DATABASE_URL")"
echo "Migrations: ${up_count} up / ${down_count} down"

tests='TestPostgresIntegration(MigrationUpDown|RuntimeBootstrapUsesPostgresStores|PipelineRunRecovery|DeploymentRunRecovery|ReleaseExecutionRecovery|ComplianceEvidenceAndRetentionRecovery)$'
NIVORA_RUN_POSTGRES_INTEGRATION=true DATABASE_URL="$DATABASE_URL" \
  go test -p 1 -count=1 -run "$tests" \
    ./internal/adapters/repository/postgres ./internal/app/runtime

echo "upgrade migration compatibility smoke passed"
