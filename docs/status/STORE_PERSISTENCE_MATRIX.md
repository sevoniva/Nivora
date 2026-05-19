# Store Persistence Matrix

Current maturity: **beta-candidate foundation**.

| Service | Store Interface | Memory Store | Postgres Store | Runtime Wiring | Migration | Tests | Production Risk |
|---|---|---|---|---|---|---|---|
| Pipeline | `usecase/pipeline.Store` | `pipeline.NewMemoryStore()` | `postgres.NewPipelineStore(pool)` | `runtime.WithConfig` selects Postgres when `runtime_store: postgres` | 000001-000006 | pipeline_store_test.go, pipeline service tests, runtime_recovery_integration_test.go | Medium |
| Deployment | `usecase/deployment.Store` | `deployment.NewMemoryStore()` | `postgres.NewDeploymentStore(pool)` | `runtime.WithConfig` selects Postgres when `runtime_store: postgres` | 000007 | deployment_release_store_test.go, runtime_recovery_integration_test.go | Medium |
| Release / Artifact | `usecase/artifact.Store` | `artifact.NewMemoryStore()` | `postgres.NewReleaseStore(pool)` | `runtime.WithConfig` selects Postgres when `runtime_store: postgres` | 000007 | deployment_release_store_test.go, runtime_recovery_integration_test.go | Medium |
| ReleaseOrchestration | `usecase/releaseorchestration.Store` | `releaseorchestration.NewMemoryStore()` | `postgres.NewReleaseOrchestrationStore(pool)` | `runtime.WithConfig` selects Postgres when `runtime_store: postgres` | 000007 | deployment_release_store_test.go, runtime_recovery_integration_test.go | Medium |
| Compliance | `usecase/compliance.Store` | `compliance.NewMemoryStore()` | `postgres.NewComplianceStore(pool)` | `runtime.WithConfig` selects Postgres when `runtime_store: postgres` | 000008 | compliance_store_integration_test.go, runtime_recovery_integration_test.go | Medium |
| **Auth** | `usecase/auth.Store` | `auth.NewMemoryStore()` | **MISSING** | **Always MemoryStore** | **MISSING** | auth service tests only | **High** |
| **Credential** | `usecase/credential.Store` | `credential.NewMemoryStore()` | **MISSING** | **Always MemoryStore** | **MISSING** | credential service tests only | **High** |
| **Security** | `usecase/security.Store` | `security.NewMemoryStore()` | **MISSING** | **Always MemoryStore** | **MISSING** | security service tests only | **High** |
| **Approval** | `usecase/approval.Store` | `approval.NewMemoryStore()` | **MISSING** | **Always MemoryStore** | **MISSING** | approval service tests only | **High** |
| **Cloud** | `usecase/cloud.Store` | `cloud.NewMemoryStore()` | **MISSING** | **Always MemoryStore** | **MISSING** | cloud service tests only | **High** |
| Tenancy | `usecase/tenancy.Store` | `tenancy.NewMemoryStore()` | `postgres.NewTenancyStore(pool)` | `runtime.WithConfig` selects Postgres when `runtime_store: postgres` | 000009 | tenancy service tests | Low |
| Plugin | N/A (built-in registry) | N/A | N/A | `pluginusecase.NewDefaultRegistry()` | N/A | plugin registry tests | Low |

## Key Risks

1. **Auth/Credential Store loss on restart**: All users, tokens, service accounts, credentials, memberships, and permissions are lost when server restarts in Postgres mode. This means auth enforcement is completely reset.
2. **Security Scan loss**: All scan records and findings disappear on restart.
3. **Approval/ChangeWindow loss**: Pending approvals, decisions, and change windows are lost on restart.
4. **Cloud Account loss**: Cloud account metadata and inventory snapshots are lost on restart.

## Remediation Plan

The missing Postgres stores are added in migration `000009_governance_persistence` and implemented in `internal/adapters/repository/postgres/`:

- `auth_store.go` — AuthStore Postgres implementation
- `credential_store.go` — CredentialStore Postgres implementation
- `security_store.go` — SecurityStore Postgres implementation
- `approval_store.go` — ApprovalStore Postgres implementation
- `cloud_store.go` — CloudStore Postgres implementation

Runtime wiring in `internal/app/runtime/runtime.go` is updated to select Postgres stores when `database.runtime_store: postgres` is configured.
