# Audit Evidence And Retention

Phase 7.3 adds a backend foundation for compliance-oriented audit search, evidence bundles, and retention policy metadata.

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

- release metadata
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
go run ./cmd/nivora evidence export pipelineRun <pipeline-run-id>
```

Markdown summary:

```bash
go run ./cmd/nivora evidence export pipelineRun <pipeline-run-id> --format markdown
```

Evidence includes log references rather than raw log content by default. Secret-like values are redacted before export.

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
go run ./cmd/nivora audit verify --scope-type pipeline
```

The verify endpoint validates the entire chain for a given scope and identifies the first broken record if tampering is detected.

### Implementation

- `AppendHashChainedAudit` in `internal/adapters/repository/postgres/audit_chain.go`
- Governance stores call it directly in their `AppendAudit` methods
- Runtime stores call it after their per-store transaction commits
- `ComplianceStore.AppendAuditRecord` provides explicit hash chain writing
- `ComplianceStore.VerifyAuditChain` validates chain integrity

## Limitations

- Hash chain verification is available at API level; not yet tested in CI with real PostgreSQL.
- Retention policy is modeled but not enforced by a background deletion job yet.
- Evidence bundles include records available in the current runtime services.
- Nivora remains a hardened beta-candidate and is not production-ready.
