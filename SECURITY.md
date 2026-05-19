# Security

## Responsible Disclosure

Nivora is preparing a public alpha and is not production-ready. If you find a security issue, please open a private disclosure channel with the maintainers once the project publishes one. Until then, avoid posting exploit details publicly.

Security review docs:

- [Threat Model](docs/security/threat-model.md)
- [Security Review Checklist](docs/security/security-review-checklist.md)
- [Route Permission Matrix](docs/security/ROUTE_PERMISSION_MATRIX.md)
- [Runner Trust Boundary](docs/security/runner-trust-boundary.md)

## Secret Handling Principles

- Do not commit real secrets.
- Store only references to secrets in the database.
- Keep secret provider implementations behind `internal/ports/secret`.
- Prefer short-lived credentials and explicit scope.

## Log Redaction

Logs must not include tokens, passwords, private keys, or full credential payloads. Future logging helpers should redact known sensitive fields before writing structured logs.

## Authentication and Authorization

Phase 3.2 includes local development auth and RBAC foundations. Dev auth and static token auth are not production SSO. Token values must come from environment variables or a future secret provider, never from committed config files. OIDC and Keycloak integration remain future work.

## Secure Defaults

Production-shaped configuration must keep auth enabled, insecure registries explicit, Kubernetes apply guarded, Argo CD sync guarded, host remote deployment guarded, and secret values redacted from logs, audit records, events, diagnostics, release notes, and examples.

Production config validation rejects disabled auth, memory runtime storage, local shell executor allowance, privileged executor allowance, remote host deployment allowance, Kubernetes apply allowance, Argo CD sync allowance, and global insecure registry allowance.
