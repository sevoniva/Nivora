#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${NIVORA_OCI_SMOKE_REFERENCE:-}" ]]; then
  echo "skipping OCI resolve smoke: NIVORA_OCI_SMOKE_REFERENCE is not set"
  exit 0
fi

args=(artifact resolve "${NIVORA_OCI_SMOKE_REFERENCE}")

if [[ -n "${NIVORA_OCI_SMOKE_REGISTRY:-}" ]]; then
  args+=(--registry "${NIVORA_OCI_SMOKE_REGISTRY}")
fi

if [[ "${NIVORA_ALLOW_INSECURE_OCI:-false}" == "true" ]]; then
  args+=(--insecure)
fi

go run ./cmd/nivora "${args[@]}"
