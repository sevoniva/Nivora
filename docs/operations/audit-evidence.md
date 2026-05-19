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

## Limitations

- The default local runtime stores evidence and retention state in memory.
- Retention policy is modeled but not enforced by a background deletion job yet.
- Evidence bundles include records available in the current runtime services.
- Nivora remains early-stage and not production-ready.
