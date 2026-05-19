# Secret Model

Phase 3.1 introduces a minimal Secret and Credential foundation for Nivora.

The model separates public metadata from secret material:

- `SecretRef` identifies a secret without carrying its value.
- `Credential` describes how a provider, registry, cluster, Argo CD instance, host, webhook, or future cloud adapter should reference a secret.
- `SecretUsage` records that an internal use case accessed a secret for a purpose.
- `SecretProvider` is the port used by use cases when a trusted internal operation needs the value.

Normal APIs should return only metadata. They must not return secret values.

## Current Implementation

The Phase 3.1 runtime includes an in-memory builtin provider for development and tests. It is useful for local API and CLI workflows, but it is not a production secret store.

Vault, Kubernetes Secret, cloud KMS, and external secret manager adapters remain future work.

## Security Rules

- Secret values are accepted only at creation or rotation boundaries.
- Secret values are not stored in domain objects returned by normal APIs.
- Audit records describe secret operations without recording values.
- Logs and errors should pass through redaction before including user-provided metadata.
- Examples must use placeholders and environment variable names only.
