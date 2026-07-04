# Audit Evidence And Retention

Phase 7.3 adds a backend foundation for compliance-oriented audit search, evidence bundles, retention policy metadata, and guarded retention runs for evidence bundle cleanup.

## Audit Search

Audit records are structured around:

- actor
- action
- subject
- scope
- reason
- request ID
- correlation ID
- safe before/after metadata

Search API:

```bash
curl "http://localhost:8080/api/v1/audit/search?subject=run"
```

CLI:

```bash
go run ./cmd/nivora audit search --subject run
```

## Evidence Bundles

Evidence bundles collect available runtime evidence for a subject:

- subject summary
- release metadata
- release plans
- release executions
- deployment runs
- artifacts
- approvals
- policy results
- security findings
- deployment plans
- log references
- events
- audit records

JSON export:

```bash
go run ./cmd/nivora evidence generate --subject-type pipelineRun --subject-id <pipeline-run-id> --token-env NIVORA_AUTH_TOKEN
go run ./cmd/nivora evidence export <evidence-bundle-id> --token-env NIVORA_AUTH_TOKEN
```

Markdown summary:

```bash
go run ./cmd/nivora evidence export <evidence-bundle-id> --format markdown --token-env NIVORA_AUTH_TOKEN
```

The legacy subject export path is still available:

```bash
go run ./cmd/nivora evidence export pipelineRun <pipeline-run-id> --token-env NIVORA_AUTH_TOKEN
```

Evidence includes log references rather than raw log content by default. Secret-like values are redacted before export.

Release evidence now also follows the ReleaseExecution records for the same release when the release orchestration service is configured. It includes execution records, release plans, target DeploymentRun summaries, DeploymentPlans, approval gates, policy results, security findings, events, audits, and log references that are already present in the runtime services. Each bundle includes a deterministic `digest` over the redacted evidence content and `generatedBy: nivora`.

## Retention Policy

Retention policy metadata covers:

- logs
- audit records
- events
- evidence bundles

API:

```bash
curl "http://localhost:8080/api/v1/retention-policy?scopeType=project&scopeId=demo"
```

CLI:

```bash
go run ./cmd/nivora retention-policy get --scope-type project --scope-id demo
go run ./cmd/nivora retention-policy set --scope-type project --scope-id demo --log-days 30 --audit-days 365 --event-days 90 --evidence-days 730
go run ./cmd/nivora retention-policy run --scope-type project --scope-id demo --dry-run
go run ./cmd/nivora retention-policy run --scope-type project --scope-id demo --dry-run=false --confirm
```

The CLI sends only retention metadata. It does not accept secret material. Retention runs are guarded:

- `run --dry-run` previews candidates and is the default.
- `run --dry-run=false --confirm` can delete expired evidence bundles for the selected scope.
- audit records remain immutable; retention reports old audit candidates but does not delete them.
- log and event retention are reported as preview-only in this foundation because those records span runtime stores.

Automated scheduled retention jobs remain future production hardening work.

## Tamper-Evident Audit Hash Chain

All audit writes are protected by a SHA-256 hash chain stored in `compliance_audit_records`. Each record includes:

- `previous_hash`: SHA-256 hash of the previous record in the same scope chain
- `record_hash`: SHA-256 hash of canonical record fields (previous_hash + actor + action + subject + timestamp)

Coverage spans all 9 audit-producing stores (pipeline, deployment, release, release orchestration, auth, credential, security, approval, cloud).

### Verification

```bash
# API
curl "http://localhost:8080/api/v1/audit/verify?scopeType=pipeline"

# CLI
go run ./cmd/nivora audit verify --scope-type pipeline --token-env NIVORA_AUTH_TOKEN
```

The verify endpoint validates the entire chain for a given scope and identifies the first broken record if tampering is detected.

### Implementation

- `AppendHashChainedAudit` in `internal/adapters/repository/postgres/audit_chain.go`
- Governance stores call it directly in their `AppendAudit` methods
- Runtime stores call it after their per-store transaction commits
- `ComplianceStore.AppendAuditRecord` provides explicit hash chain writing
- `ComplianceStore.VerifyAuditChain` validates chain integrity

## Limitations

- Hash chain verification is available at API level and in the PostgreSQL integration job.
- Retention policy is modeled but not enforced by a background deletion job yet.
- Evidence bundles include records available in the current runtime services; they do not fetch external registries, clusters, Git providers, or scanners.
- Nivora remains a hardened beta-candidate and is not production-ready.
