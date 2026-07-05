# Tenant Isolation

Nivora provides project/environment-level tenant isolation through scoped service accounts, RBAC permissions, and route-level authorization. Full cross-tenant data isolation is a hardening work in progress.

## Current Isolation Coverage

### Tested and Verified

| Resource | Actor | Cross-Scope Prevention | Status |
|---|---|---|---|
| Credentials | Developer (project-A) | Permission-gated (credential.manage) | ✅ |
| Deployments | Developer (project-A) | Permission-gated (deployment.create) | ✅ |
| Releases | Developer (project-A) | Permission-gated (release.create) | ✅ |
| PipelineRuns | Developer (project-A) | Permission-gated (pipeline.run) | ✅ |
| Audit | Auditor (project-A) | Permission-gated (audit.read) | ✅ |
| Approvals | Maintainer (project-A) | Permission-gated (deployment.approve) | ✅ |
| Cloud Accounts | Admin (project-A) | Permission-gated (credential.manage) | ✅ |
| Secrets | Admin (project-A) | Permission-gated (credential.manage) | ✅ |
| Visualization | Viewer (project-A) | Read-only access | ✅ |
| Runner Admin | Developer (project-A) | Permission-gated (runner.manage) | ✅ |
| Runner job claim | Scoped runner token | Project/environment labels and RunnerGroup constraints | ✅ |
| Runner admin job mutation | Admin (project-A or environment-prod) | `/api/v1/jobs/{id}/logs` and `/api/v1/jobs/{id}/status` check the owning PipelineRun project/environment scope | ✅ |
| Policies | Admin (project-A) | Permission-gated (policy.manage) | ✅ |

### Not Yet Exhaustively Tested

| Area | Gap | Risk |
|---|---|---|
| Cross-tenant deployment data | List endpoints may return all tenants | Medium |
| Cross-tenant credential list | List not scope-filtered | Medium |
| Older records without scope metadata | Some historical/foundation records may not carry project/environment ownership | Medium |
| Audit search scope filtering | Search may return cross-tenant records | Low |
| Visualization data aggregation | Summary may include cross-tenant data | Low |

## How Isolation Works

1. **Service Accounts** are created with a scope (org/project/environment).
2. **RBAC permissions** gate route access based on role.
3. **Route middleware** checks Bearer token, resolves subject, validates permission.
4. **Tenancy usecase** provides `CanAccessScope()` and `CheckQuota()` methods.

## Testing Tenant Isolation

```bash
# Run the tenant isolation test matrix
go test -v -run TestTenantIsolation ./internal/api/http/routes/

# Run the full RBAC test suite (100+ sub-tests including cross-tenant)
go test -v -run "TestRBAC|TestCrossTenant|TestTenantIsolation" ./internal/api/http/routes/
```

## Current Limitations

- Cross-tenant data filtering is not enforced at the store/repository level.
- List endpoints return all records across tenants.
- Visualization summaries are not tenant-filtered.
- Audit search is not scope-filtered by default.
- Runner job claims are scope-checked through runner labels, RunnerGroup constraints, and queued PipelineRun ownership. This is still a control-plane metadata guardrail, not a runtime sandbox.
- Admin compatibility job mutation endpoints are scope-checked against the owning PipelineRun project or environment before log append or status update.
- These are documented gaps for future hardening.
