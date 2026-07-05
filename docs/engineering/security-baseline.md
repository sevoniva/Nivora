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

Phase 7.1 adds rotation, provider validation, external provider skeletons, and simple usage policy metadata. These additions preserve the same rule: secret values may enter only at create/rotate boundaries and must not appear in logs, audit records, events, or normal API responses.

SecretProvider validation status is adapter output and is treated as untrusted. Provider names, messages, capabilities, and metadata are sanitized before they are returned through API or CLI paths. Validation responses are metadata checks only; they are not evidence of production external-secret readiness.

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
