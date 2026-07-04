# Production Doctor

`nivora doctor` is a read-only operator check for production-like configuration posture. It does not mutate runtime state and it does not prove Nivora is production-ready.

```bash
nivora doctor --file configs/production.example.yaml
nivora doctor config --file configs/production.example.yaml
nivora doctor security --file configs/production.example.yaml
nivora doctor runtime --file configs/production.example.yaml
```

Statuses:

- `PASS`: the checked value matches the current production-like guardrail.
- `WARN`: acceptable for local/dev, but review before production-like use.
- `FAIL`: unsafe or invalid for production-like use.
- `NOT_CHECKED`: reserved for checks that need live dependencies.

Current checks cover:

- `database.runtime_store` is `postgres` for production-like installs.
- auth is enabled and not `dev` or `disabled` in production-like installs.
- local shell, privileged executor, Kubernetes apply, Argo sync, remote host deploy, and global insecure registry flags are disabled in production-like installs.
- MCP action tools are disabled.
- event bus and object store types are explicit.
- audit and evidence persistence are tied to PostgreSQL for production-like installs.
- event outbox persistence is tied to PostgreSQL for production-like installs.
- runner identity and heartbeat settings are explicit.
- secret-like evidence values are redacted from command output.
- secret provider posture, runner token hash storage, OpenAPI route contract verification, repository secret scanning, database connectivity, migration status, runner heartbeat freshness, and audit-chain verification are reported as `NOT_CHECKED` by the local config doctor because they require live runtime state, repository checks, or CI verification.

Limitations:

- Database connectivity, migration drift, runner token storage, route contract drift, secret scan status, runner heartbeat freshness, and audit chain status are still checked through the existing API, scripts, tests, or CI rather than this local config command.
- The doctor command is a posture check. Passing it is not a production-readiness claim.
