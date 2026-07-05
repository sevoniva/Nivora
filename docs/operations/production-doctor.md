# Production Doctor

`nivora doctor` is a read-only operator check for production-like configuration posture. It does not mutate runtime state and it does not prove Nivora is production-ready.

```bash
nivora doctor --file configs/production.example.yaml
nivora doctor config --file configs/production.example.yaml
nivora doctor security --file configs/production.example.yaml
nivora doctor runtime --file configs/production.example.yaml
nivora doctor live --server http://localhost:8080 --token-env NIVORA_AUTH_TOKEN
nivora doctor live --server http://localhost:8080 --audit-scope-type deployment --audit-scope-id <scope-id>
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

`nivora doctor live` is the explicit live mode. It is still read-only. It calls the running server's existing diagnostics, runtime recovery, and audit verification endpoints:

- `GET /api/v1/system/diagnostics`
- `GET /api/v1/system/runtime/recovery`
- `GET /api/v1/audit/verify`

The live report checks:

- whether the server reports `postgres` or memory runtime mode
- whether live diagnostics include degraded critical dependencies
- whether runtime recovery reports healthy, warning, or degraded state
- whether pending or failed event outbox records are present
- whether audit hash-chain verification returns `valid: true` for the selected audit scope

This mode does not run migrations, reconcile runtime state, publish outbox events, mutate audit records, or contact external services.

Limitations:

- Migration drift, runner token storage, route contract drift, secret scan status, and runner heartbeat freshness are still checked through existing scripts, tests, API routes, or CI rather than by the local config doctor. `doctor live` covers diagnostics, runtime recovery posture, event outbox counts, and audit-chain verification against a running server.
- The doctor command is a posture check. Passing it is not a production-readiness claim.
