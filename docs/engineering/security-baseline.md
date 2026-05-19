# Security Baseline

## Secrets

Secret values must not be:

- logged
- returned by normal APIs
- stored in plaintext
- included in audit records
- included in test fixtures as realistic credentials

## Credentials

Credential records should store metadata separately from secret material.

Phase 3.1 adds `SecretRef`, `Credential`, and `SecretUsage` records. Normal APIs return references and metadata only. Trusted internal use cases may resolve values through the `SecretProvider` port and should audit the purpose of access without storing the value.

The builtin provider is an early development provider. Production-grade Vault, Kubernetes Secret, cloud KMS, and external secret manager integrations remain future work.

## Logs

Logs must redact:

- tokens
- passwords
- private keys
- kubeconfigs
- cloud access keys
- authorization headers

## Runner

Runner execution must be treated as sensitive.

Do not add privileged execution features without:

- threat model
- permission boundary
- audit trail
- cancellation behavior
- log redaction

## External Integrations

Do not add real external integration code without considering:

- credential storage
- credential validation
- error redaction
- rate limits
- audit events
- timeout and retry behavior
