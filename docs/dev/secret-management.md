# Secret Management

Phase 3.1 added a local Secret and Credential foundation for development. Phase 7.1 adds rotation and provider validation surfaces while keeping secret values out of normal API responses.

## Store A Secret

Prefer environment variables so values do not appear in shell history:

Set NIVORA_TOKEN in your shell, then run:

```bash
go run ./cmd/nivora secret put --name local-registry-token --value-env NIVORA_TOKEN --token-env NIVORA_AUTH_TOKEN
```

The response is a `SecretRef`. It does not include the secret value.

For in-process development without a server, add `--local`.

## Rotate A Secret

Rotation also uses environment variable indirection:

```bash
go run ./cmd/nivora secret rotate <secret-id> --value-env NIVORA_ROTATED_TOKEN --token-env NIVORA_AUTH_TOKEN
```

The response is the updated `SecretRef` metadata. It does not include the new value.

## Validate The Secret Provider

```bash
go run ./cmd/nivora secret provider validate --token-env NIVORA_AUTH_TOKEN
```

Provider validation reports the configured provider name, capability metadata, and whether the provider is configured. It must not return secret values.

## Create Credential Metadata

```bash
go run ./cmd/nivora credential create --file examples/credentials/registry-credential.yaml --token-env NIVORA_AUTH_TOKEN
```

The example references a placeholder key. Before validation, create a matching secret with `secret put`. For in-process development without a server, add `--local`.

## Validate A Credential

```bash
go run ./cmd/nivora credential validate <credential-id> --token-env NIVORA_AUTH_TOKEN
```

Validation only checks that the `SecretRef` can be resolved by the configured provider. It does not call Harbor, Argo CD, Kubernetes, cloud APIs, or SSH hosts in this phase.

## Limitations

- The builtin provider is development-only and in-memory.
- Vault, Kubernetes Secret, and cloud KMS providers are foundations or placeholders in Phase 7.1.
- External provider validation does not require local Vault, Kubernetes, or cloud services in CI.
- Normal APIs must not return secret values.
- Audit records track secret operations and usage, not values.
- Production-grade external secret storage, enterprise KMS integration, and provider-specific credential workflows remain future work.
