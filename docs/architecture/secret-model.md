# Secret Model

Phase 3.1 introduced a minimal Secret and Credential foundation for Nivora. Phase 7.1 hardens that foundation with external provider shapes, provider validation metadata, rotation support, usage policy metadata, and broader redaction tests.

The model separates public metadata from secret material:

- `SecretRef` identifies a secret without carrying its value.
- `Credential` describes how a provider, registry, cluster, Argo CD instance, host, webhook, or future cloud adapter should reference a secret.
- `SecretUsage` records that an internal use case accessed a secret for a purpose.
- `SecretProvider` is the port used by use cases when a trusted internal operation needs the value.
- `SecretPolicy` can constrain a reference to allowed use names and environments.

Normal APIs should return only metadata. They must not return secret values.

## Current Implementation

The runtime includes an in-memory builtin provider for development and tests. It supports create, get, list, delete, rotate, usage audit, and provider validation. It is useful for local API and CLI workflows, but it is not a production secret store.

Phase 7.1 adds adapter foundations for Vault, Kubernetes Secrets, and AWS/Aliyun/Tencent KMS. These adapters expose configuration and capability validation shapes, but they do not require Vault, Kubernetes, or cloud accounts in CI and do not yet implement production value retrieval.

## Rotation

Rotation is modeled through the `SecretProvider` port. A caller supplies a new value at the rotation boundary, the provider updates its version metadata, and APIs return only the updated `SecretRef`. Audit and events record that rotation happened without recording the secret value.

## Security Rules

- Secret values are accepted only at creation or rotation boundaries.
- Secret values are not stored in domain objects returned by normal APIs.
- Audit records describe secret operations without recording values.
- Logs and errors should pass through redaction before including user-provided metadata.
- Provider validation returns capability and configuration status only.
- Examples must use placeholders and environment variable names only.
