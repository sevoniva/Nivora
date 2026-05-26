# Store Persistence Matrix

Current maturity: **hardened beta-candidate (0.9.0-rc.1), not production-ready**. All 11 runtime stores have PostgreSQL implementations. Audit hash chaining is wired through the audit-producing store paths. Runtime store audit paths (pipeline/deployment/release) also keep their per-store audit tables.

| Service | Store Interface | Memory Store | Postgres Store | Runtime Wiring | Migration | Tests | Production Risk |
|---|---|---|---|---|---|---|---|
| Pipeline | `usecase/pipeline.Store` | `pipeline.NewMemoryStore()` | `postgres.NewPipelineStore(pool)` | `runtime.WithConfig` | 000001-000006 | pipeline_store_test.go, runtime_recovery_integration_test.go | Medium |
| Deployment | `usecase/deployment.Store` | `deployment.NewMemoryStore()` | `postgres.NewDeploymentStore(pool)` | `runtime.WithConfig` | 000007 | deployment_release_store_test.go, runtime_recovery_integration_test.go | Medium |
| Release / Artifact | `usecase/artifact.Store` | `artifact.NewMemoryStore()` | `postgres.NewReleaseStore(pool)` | `runtime.WithConfig` | 000007 | deployment_release_store_test.go, runtime_recovery_integration_test.go | Medium |
| ReleaseOrchestration | `usecase/releaseorchestration.Store` | `releaseorchestration.NewMemoryStore()` | `postgres.NewReleaseOrchestrationStore(pool)` | `runtime.WithConfig` | 000007 | deployment_release_store_test.go, runtime_recovery_integration_test.go | Medium |
| Compliance | `usecase/compliance.Store` | `compliance.NewMemoryStore()` | `postgres.NewComplianceStore(pool)` | `runtime.WithConfig` | 000008 | compliance_store_integration_test.go, runtime_recovery_integration_test.go | Medium |
| Auth | `usecase/auth.Store` | `auth.NewMemoryStore()` | `postgres.NewAuthStore(pool)` | `runtime.WithConfig` | 000009 | auth service tests, runtime test | Medium |
| Credential | `usecase/credential.Store` | `credential.NewMemoryStore()` | `postgres.NewCredentialStore(pool)` | `runtime.WithConfig` | 000009 | credential service tests, runtime test | Medium |
| Security | `usecase/security.Store` | `security.NewMemoryStore()` | `postgres.NewSecurityStore(pool)` | `runtime.WithConfig` | 000009 | security service tests, runtime test | Medium |
| Approval | `usecase/approval.Store` | `approval.NewMemoryStore()` | `postgres.NewApprovalStore(pool)` | `runtime.WithConfig` | 000009 | approval service tests, runtime test | Medium |
| Cloud | `usecase/cloud.Store` | `cloud.NewMemoryStore()` | `postgres.NewCloudStore(pool)` | `runtime.WithConfig` | 000009 | cloud service tests, runtime test | Medium |
| Tenancy | `usecase/tenancy.Store` | `tenancy.NewMemoryStore()` | `postgres.NewTenancyStore(pool)` | `runtime.WithConfig` | 000009 | tenancy service tests | Low |

## Postgres Integration Tests

- **Local opt-in**: `NIVORA_RUN_POSTGRES_INTEGRATION=true make test-postgres-integration`
- **CI**: `postgres-integration` job in `.github/workflows/ci.yaml` with PostgreSQL 16 service container
- **Not in `make verify` baseline**: Requires a running PostgreSQL instance; kept separate by design

## Audit Hash Chain Coverage

| Store Group | Hash-Chained | Method |
|---|---|---|
| Governance (auth, credential, security, approval, cloud) | ✅ | `AppendHashChainedAudit` writes to `compliance_audit_records` |
| Runtime (pipeline, deployment, release, release orchestration) | ✅ | `AppendHashChainedAudit` called after per-store audit table insert |
| Compliance | ✅ | `AppendAuditRecord` with SHA-256 chaining |

All 9 audit-producing stores now produce hash-chained `compliance_audit_records` entries. The verify API (`GET /api/v1/audit/verify`) can validate the chain for any scope.

## Key Risks

1. **Multi-process recovery not proven in CI**: Postgres integration tests exist but require a live database; CI job runs separately from `make verify`.
2. **Runner sandboxing is config-level, not OS-level**: Workspace isolation, env blocklist, and process group cleanup are enforced, but no container/VM sandbox.
3. **Production install smoke test not automated**: Helm templates validated statically (11/11 checks) but no automated install+healthcheck test exists.
