# Security Model

Security is a first-class architecture concern.

## Secrets and Credentials

Credential records should store metadata and SecretRefs, not raw secret values. SecretProvider Adapters own secret material. Secret values must not appear in logs, audit records, normal API responses, or test fixtures.

## Log Redaction

Logs should redact tokens, passwords, private keys, kubeconfigs, cloud credentials, and authorization headers.

## Audit

Audit records should preserve accountability without exposing secrets. Important operations include project changes, credential changes, pipeline triggers, approvals, deployments, rollbacks, policy changes, and runner registration.

## Policy Gates

Policy is an enforceable gate. PolicyResults should record outcomes and reasons without leaking sensitive data.

## Runner Trust Boundary

Runners execute work and may access target systems. Runner protocol, credentials, cancellation, log streaming, and isolation require conservative design. Privileged execution requires an RFC.

## Least Privilege

Adapters should request the smallest permissions needed. Cloud, Kubernetes, Git, and registry credentials should be scoped to the operation.

## No Realistic Fake Credentials

Docs, examples, and tests must not include real credentials or realistic-looking fake credentials.

