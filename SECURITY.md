# Security

## Responsible Disclosure

Nivora is in Phase 0 and is not production-ready. If you find a security issue, please open a private disclosure channel with the maintainers once the project publishes one. Until then, avoid posting exploit details publicly.

## Secret Handling Principles

- Do not commit real secrets.
- Store only references to secrets in the database.
- Keep secret provider implementations behind `internal/ports/secret`.
- Prefer short-lived credentials and explicit scope.

## Log Redaction

Logs must not include tokens, passwords, private keys, or full credential payloads. Future logging helpers should redact known sensitive fields before writing structured logs.

