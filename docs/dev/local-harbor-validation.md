# Local Harbor Validation

Harbor is treated as an OCI-compatible registry in Phase 2.5. Nivora does not implement Harbor project, user, robot account, scanning, retention, or replication APIs.

Optional local validation may use a Harbor-compatible endpoint, but it is never required by tests or CI.

Suggested environment variables for local experiments:

```bash
export NIVORA_EXTERNAL_HARBOR_URL='https://harbor.example.com'
export NIVORA_EXTERNAL_HARBOR_USERNAME='<username>'
export NIVORA_EXTERNAL_HARBOR_PASSWORD='<password>'
```

Do not commit these values. Do not paste real credentials into issues, docs, examples, tests, logs, or audit records.

For an HTTP local registry, insecure mode must be explicit:

```bash
go run ./cmd/nivora artifact resolve localhost:30500/team/app:1.0.0 --insecure
```

This validates digest resolution behavior only. It does not prove production readiness.
