# Operator Guide

This guide is for people installing, configuring, validating, and operating Nivora. It is a production-direction guide, not a production-readiness claim.

## Install

Install paths:

- [Production-direction install](../operations/production-install.md)
- [Docker Compose install](../operations/install-docker-compose.md)
- [Kubernetes install](../operations/install-kubernetes.md)

Baseline checks:

```sh
make verify
make helm-template
make helm-lint
```

`make docker-build` depends on external base image registry access and should be run where network policy allows it.

## Config

Start with:

- [Configuration](../operations/configuration.md)
- `configs/server.yaml`
- `configs/worker.yaml`
- `configs/runner.yaml`
- `configs/production.example.yaml`

Production-shaped config should keep `auth.enabled: true`, use explicit secret references or environment variable names, and avoid raw secret values.

## Auth

Auth/RBAC docs:

- [Auth model](../architecture/auth-model.md)
- [OIDC auth operations](../operations/auth-oidc.md)
- [RBAC operations](../operations/rbac.md)

Local dev auth is not production SSO. Token values and OIDC credentials must come from environment variables or a secret provider.

## Secrets

Secret docs:

- [Secret concept](../concepts/secret.md)
- [Credential concept](../concepts/credential.md)
- [Secret model](../architecture/secret-model.md)
- [Vault provider foundation](../operations/secrets-vault.md)
- [KMS and external providers](../operations/secrets-kms.md)

Normal APIs should return secret metadata only. Secret values must not appear in logs, audit records, diagnostics, examples, or release notes.

## Backup

Backup and restore docs:

- [Backup and restore](../operations/backup-restore.md)
- [Database operations](../operations/database.md)
- [HA and disaster recovery](../operations/ha-disaster-recovery.md)
- [Upgrade guide](../operations/upgrade.md)

Back up PostgreSQL, object-store data, sanitized configs, and secret values through the configured secret provider before migration or upgrade validation.

## Observability

Operations docs:

- [Observability](../operations/observability.md)
- [Performance](../operations/performance.md)
- [Runtime recovery](../operations/runtime-recovery.md)
- [Runner fleet](../operations/runner-fleet.md)

Useful checks:

```sh
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
curl http://localhost:8080/api/v1/system/diagnostics
curl http://localhost:8080/api/v1/system/runtime/recovery
```

## Troubleshooting

Start with:

- [Troubleshooting](../operations/troubleshooting.md)
- [Runbook: stuck PipelineRun](../operations/runbooks/stuck-pipelinerun.md)
- [Runbook: failed DeploymentRun](../operations/runbooks/failed-deploymentrun.md)
- [Runbook: offline runner](../operations/runbooks/offline-runner.md)
- [Runbook: DB unavailable](../operations/runbooks/db-unavailable.md)
- [Runbook: object store unavailable](../operations/runbooks/object-store-unavailable.md)
- [Runbook: policy gate denied](../operations/runbooks/policy-gate-denied.md)

## AI / MCP

Local stdio MCP can help maintainers inspect Nivora state with AI tools. It is read-only and plan-only, records MCP audit events, rejects runner tokens, redacts secret-like output, and denies destructive action-shaped calls.

Start with:

- [MCP server](../dev/mcp-server.md)
- [MCP security](../security/mcp-security.md)
- [MCP threat model](../security/mcp-threat-model.md)
- [MCP control-plane review](../status/MCP_CONTROL_PLANE_REVIEW.md)
- [AI control-plane product review](../status/AI_CONTROL_PLANE_PRODUCT_REVIEW.md)
- [Remote read-only MCP RFC](../rfcs/remote-mcp-read-only.md)

Remote MCP is not implemented. Action MCP is blocked.
