# Cloud Providers

Phase 8.0 hardens Nivora's multi-cloud provider foundation for AWS, Aliyun, Tencent Cloud, and generic inventory providers.

This is still inventory and target-binding foundation work. Nivora does not deploy cloud resources, create clusters, mutate infrastructure, or claim production-ready cloud operations.

## Capabilities

The cloud provider port supports:

- provider metadata and capabilities
- config validation
- credential reference validation
- regions
- clusters
- hosts
- registries
- inventory snapshots
- target binding metadata

Provider metadata is available through:

```sh
go run ./cmd/nivora cloud providers --local
```

Server-backed cloud APIs are RBAC-protected. Use `--token-env` for server calls instead of passing tokens directly:

```sh
go run ./cmd/nivora cloud providers --token-env NIVORA_AUTH_TOKEN
go run ./cmd/nivora cloud account list --token-env NIVORA_AUTH_TOKEN
go run ./cmd/nivora cloud account get <account-id> --token-env NIVORA_AUTH_TOKEN
go run ./cmd/nivora cloud account validate <account-id> --token-env NIVORA_AUTH_TOKEN
```

## Credentials

Cloud accounts must reference credentials through `CredentialRef` or future `SecretRef`-backed configuration. Secret values must not be placed in cloud account specs, logs, audit records, examples, or API responses. `--token-env` carries the Nivora API token for RBAC and is separate from provider credentials.

## Provider Adapters

The baseline AWS, Aliyun, and Tencent adapters expose beta-direction capability metadata and deterministic inventory behavior without requiring SDKs or real credentials in CI.

Real cloud SDK calls remain optional future work and must stay inside `internal/adapters/cloud`.

## Inventory

Inventory snapshots include regions, clusters, hosts, registries, warnings, and target binding metadata. Target bindings prepare future release targets to bind to discovered resources, but Phase 8.0 does not execute cloud deployments.

```sh
go run ./cmd/nivora cloud inventory <account-id> --token-env NIVORA_AUTH_TOKEN
go run ./cmd/nivora cloud clusters <account-id> --token-env NIVORA_AUTH_TOKEN
go run ./cmd/nivora cloud hosts <account-id> --token-env NIVORA_AUTH_TOKEN
go run ./cmd/nivora cloud registries <account-id> --token-env NIVORA_AUTH_TOKEN
```

## Limitations

- No cloud deployment.
- No resource mutation.
- No baseline CI dependency on cloud credentials or network access.
- No production readiness claim.
- Real AWS EKS/EC2/ECR, Aliyun ACK/ECS/ACR, and Tencent TKE/CVM/TCR integration remains future adapter work.
