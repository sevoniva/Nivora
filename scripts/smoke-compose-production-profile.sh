#!/usr/bin/env bash
set -euo pipefail

profile="deployments/docker-compose/docker-compose.production.example.yaml"

test -f "$profile"
grep -q 'NIVORA_PRODUCTION_CONFIG' "$profile"
grep -q 'NIVORA_AUTH_TOKEN' "$profile"
grep -q 'NIVORA_POSTGRES_PASSWORD' "$profile"

if grep -q 'POSTGRES_HOST_AUTH_METHOD: trust' "$profile"; then
  echo "production-like compose profile must not use trust authentication"
  exit 1
fi
if grep -q 'runtime_store:[[:space:]]*memory' "$profile"; then
  echo "production-like compose profile must not force memory runtime store"
  exit 1
fi
if grep -q 'auth:[[:space:]]*disabled' "$profile"; then
  echo "production-like compose profile must not disable auth"
  exit 1
fi
if grep -E 'password[[:space:]]*[:=][[:space:]]*["'\''][^"$'\''{][^"'\'']+["'\'']' "$profile" >/dev/null; then
  echo "production-like compose profile appears to include an inline password"
  exit 1
fi

if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
  NIVORA_POSTGRES_PASSWORD=placeholder-required-by-compose \
  NIVORA_AUTH_TOKEN=placeholder-required-by-compose \
  NIVORA_PRODUCTION_CONFIG=./configs/production.example.yaml \
    docker compose -f "$profile" config >/dev/null
else
  echo "docker compose not found; skipped compose config rendering"
fi

echo "production-like compose profile smoke check passed"
