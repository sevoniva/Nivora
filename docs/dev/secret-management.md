# Secret Management

Phase 3.1 adds a local Secret and Credential foundation for development.

## Store A Secret

Prefer environment variables so values do not appear in shell history:

Set NIVORA_TOKEN in your shell, then run:

```bash
go run ./cmd/nivora secret put --name local-registry-token --value-env NIVORA_TOKEN
```

The response is a `SecretRef`. It does not include the secret value.

## Create Credential Metadata

```bash
go run ./cmd/nivora credential create --file examples/credentials/registry-credential.yaml --local
```

The example references a placeholder key. Before server-backed validation, create a matching secret with `secret put`.

## Validate A Credential

```bash
go run ./cmd/nivora credential validate <credential-id>
```

Validation only checks that the `SecretRef` can be resolved by the configured provider. It does not call Harbor, Argo CD, Kubernetes, cloud APIs, or SSH hosts in this phase.

## Limitations

- The builtin provider is development-only and in-memory.
- Normal APIs must not return secret values.
- Audit records track secret operations and usage, not values.
- Vault, Kubernetes Secret, cloud KMS, and external secret manager adapters are future work.
