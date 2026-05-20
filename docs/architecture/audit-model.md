# Audit Model

Nivora audit records provide a tamper-evident, hash-chained trail of all significant operations across the control plane.

## Audit Scopes

Each audit entry is scoped to its source domain. The verify API can validate the chain for any scope:

| Scope | Source | Hash-Chained |
|---|---|---|
| `pipeline` | PipelineStore.AppendAudit | ✅ |
| `deployment` | DeploymentStore.AppendAudit | ✅ |
| `release` | ReleaseStore.AppendAudit | ✅ |
| `release_execution` | ReleaseOrchestrationStore.AppendAudit | ✅ |
| `auth` | AuthStore.AppendAudit | ✅ |
| `credential` | CredentialStore.AppendAudit | ✅ |
| `security` | SecurityStore.AppendAudit | ✅ |
| `approval` | ApprovalStore.AppendAudit | ✅ |
| `cloud` | CloudStore.AppendAudit | ✅ |

## Hash Chain Algorithm

Each audit record is part of a SHA-256 hash chain within its scope:

```
previous_hash = latest record_hash in scope (empty for first record)
canonical = previous_hash + "|" + actor_id + "|" + action + "|" + scope_type + "|" + subject_id + "|" + timestamp
record_hash = SHA256(canonical)
```

Rules:
- Deterministic canonicalization
- No volatile fields (timestamps truncated to microsecond)
- No secret values in canonical fields
- Scoped chains (each scope has an independent chain)
- Transaction-safe: previous_hash is read within the same transaction or sequentially after commit

## Implementation

- `internal/adapters/repository/postgres/audit_chain.go` — shared `AppendHashChainedAudit` helper
- `compliance_audit_records` table (migration 000008) stores `previous_hash` and `record_hash`
- `governance_audit_logs` table (migration 000009) stores plain queryable audit entries
- Per-store audit tables store domain-specific audit details

## Verification

```bash
# API
curl "http://localhost:8080/api/v1/audit/verify?scopeType=pipeline"

# CLI
nivora audit verify --scope-type pipeline --server http://localhost:8080
```

The verify endpoint:
1. Reads all audit records for the scope ordered by created_at
2. Recomputes each record_hash from canonical fields
3. Validates previous_hash links between consecutive records
4. Returns `{valid: true}` or `{valid: false, firstBrokenId: "..."}`

## Limitations

- Hash chain verification requires PostgreSQL; not available in memory store mode.
- Per-store audit tables (`runtime_*_audit_logs`) are plain queryable tables without hash columns.
- The hash chain lives in `compliance_audit_records`, written alongside per-store audit inserts.
- Verification is not yet run in CI with real PostgreSQL (requires `NIVORA_RUN_POSTGRES_INTEGRATION=true`).
- This is a hardened beta-candidate feature; not a production compliance certification.
