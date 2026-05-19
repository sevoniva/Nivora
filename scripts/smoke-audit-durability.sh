#!/usr/bin/env bash
set -euo pipefail

echo "Running audit/evidence durability smoke checks..."
go test ./internal/usecase/compliance ./internal/adapters/repository/postgres -run 'TestEvidenceBundleGeneration|TestRetentionPolicy|TestPostgresIntegrationComplianceEvidenceAndRetentionRecovery'

if [[ "${NIVORA_RUN_POSTGRES_INTEGRATION:-}" != "true" ]]; then
  echo "PostgreSQL integration recovery test was skipped; set NIVORA_RUN_POSTGRES_INTEGRATION=true and DATABASE_URL to exercise the durable store."
fi
