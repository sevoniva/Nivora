# Route Permission Matrix

Nivora is a hardened beta-candidate foundation, not production-ready. This matrix records the intended backend authorization contract for critical API groups. It is route-contract guidance for maintainers and should be kept in sync with `internal/api/http/routes/routes.go` and OpenAPI.

Auth modes:

- `BearerAuth`: control-plane API token, service-account token, static token, or future OIDC token.
- `RunnerTokenAuth`: `X-Nivora-Runner-Token`, valid only for runner heartbeat, job claim, job log append, and job status update endpoints.
- Dev auth may bypass these checks only in local development mode.

| Method | Path / Group | Required Permission | Scope Type | Subject Type | Auth Required? | Runner Token Allowed? | Service Account Allowed? | Notes |
|---|---|---|---|---|---|---|---|---|
| GET | `/healthz`, `/readyz`, `/metrics` | none | system | anonymous | no | no | no | Operational probes; no sensitive state should be returned. |
| GET | `/api/v1/version`, `/api/v1/system/info`, `/api/v1/system/runtime`, `/api/v1/system/diagnostics` | authenticated | system | user/service account | yes in auth-enabled modes | no | yes | Diagnostics must remain metadata-only. |
| GET/POST | `/api/v1/system/runtime/recovery`, `/api/v1/system/runtime/reconcile` | operator/admin review required | system | user/service account | yes | no | yes | Recovery routes are sensitive; keep restricted in production deployments. |
| GET | `/api/v1/auth/whoami`, `/api/v1/auth/permissions`, `/api/v1/auth/token-info` | authenticated | self | user/service account | yes | no | yes | Must not echo token values. |
| GET | `/api/v1/users`, `/api/v1/roles`, `/api/v1/permissions` | authenticated | org/global | user/service account | yes | no | yes | Read-only identity metadata. |
| GET/POST | `/api/v1/orgs/{id}/members` | `project.read` / `project.write` | org | user/service account | yes | no | yes | Current foundation maps org membership through project permissions. |
| GET/POST | `/api/v1/projects/{id}/members` | `project.read` / `project.write` | project | user/service account | yes | no | yes | Scoped tokens must match project scope. |
| GET/POST | `/api/v1/environments/{id}/members` | `environment.read` / `environment.write` | environment | user/service account | yes | no | yes | Scoped tokens must match environment scope. |
| GET | `/api/v1/orgs`, `/api/v1/orgs/{id}` | `project.read` | org | user/service account | yes | no | yes | Foundation organization catalog. |
| POST | `/api/v1/orgs` | `project.write` | org | user/service account | yes | no | yes | Creates an organization catalog record. |
| PATCH/DELETE | `/api/v1/orgs/{id}` | `project.write` | org | user/service account | yes | no | yes | Delete disables instead of hard-deleting. |
| GET | `/api/v1/projects`, `/api/v1/projects/{id}` | `project.read` | project | user/service account | yes | no | yes | Foundation project catalog. |
| POST | `/api/v1/projects` | `project.write` | project | user/service account | yes | no | yes | Parent org is validated. |
| PATCH/DELETE | `/api/v1/projects/{id}` | `project.write` | project | user/service account | yes | no | yes | Delete disables instead of hard-deleting. |
| GET | `/api/v1/applications`, `/api/v1/applications/{id}` | `application.read` | application/project | user/service account | yes | no | yes | Foundation application catalog. |
| POST | `/api/v1/applications` | `application.write` | application/project | user/service account | yes | no | yes | Parent project is validated. |
| PATCH/DELETE | `/api/v1/applications/{id}` | `application.write` | application/project | user/service account | yes | no | yes | Delete disables instead of hard-deleting. |
| GET | `/api/v1/environments`, `/api/v1/environments/{id}` | `environment.read` | environment/project | user/service account | yes | no | yes | Foundation environment catalog. |
| POST | `/api/v1/environments` | `environment.write` | environment/project | user/service account | yes | no | yes | Parent project is validated. |
| PATCH/DELETE | `/api/v1/environments/{id}` | `environment.write` | environment/project | user/service account | yes | no | yes | Delete disables instead of hard-deleting. |
| GET | `/api/v1/repositories`, `/api/v1/repositories/{id}` | `project.read` | project/repository | user/service account | yes | no | yes | Foundation SCM repository catalog; no provider network calls. |
| POST | `/api/v1/repositories` | `project.write` | project/repository | user/service account | yes | no | yes | Parent project is validated; CredentialRef id only. |
| PATCH/DELETE | `/api/v1/repositories/{id}` | `project.write` | project/repository | user/service account | yes | no | yes | Delete disables instead of hard-deleting. |
| GET/POST | `/api/v1/secrets`, `/api/v1/secrets/*` | `credential.manage` | org/project/environment/runner/global | user/service account | yes | no | yes | Responses return refs only, never secret values. |
| GET/POST/DELETE | `/api/v1/credentials*` | `credential.manage` | org/project/environment/runner/global | user/service account | yes | no | yes | Responses return metadata/SecretRef only. |
| POST | `/api/v1/artifacts/inspect`, `/api/v1/artifacts/resolve` | `release.create` | project/application | user/service account | yes | no | yes | Registry credentials must remain behind SecretRef/CredentialRef. |
| POST | `/api/v1/artifact-registries/validate` | `release.create` | project | user/service account | yes | no | yes | Validation must not echo credentials. |
| Placeholder | `/api/v1/artifact-registries` | placeholder | project | user/service account | yes | no | yes | Full registry CRUD is not implemented. |
| GET | `/api/v1/pipelines`, `/api/v1/pipelines/{id}` | `project.read` | project/pipeline | user/service account | yes | no | yes | Foundation pipeline definition catalog. |
| POST | `/api/v1/pipelines` | `project.write` | project/pipeline | user/service account | yes | no | yes | Creates a validated Pipeline definition record; does not execute it. |
| PATCH/DELETE | `/api/v1/pipelines/{id}` | `project.write` | project/pipeline | user/service account | yes | no | yes | Update can create a new definition version; delete disables instead of hard-deleting. |
| GET/POST | `/api/v1/pipeline-runs*` | `project.read` for list, `pipeline.run` for create/cancel | project | user/service account | yes | no | yes | Shell executor is not a sandbox. |
| GET | `/api/v1/runners`, `/api/v1/runners/{id}` | `runner.manage` | runner group/project | user/service account | yes | no | yes | Runner metadata includes token metadata only, never token hashes/raw values. |
| POST | `/api/v1/runners/register`, `/api/v1/runners/{id}/token/*`, `/api/v1/runners/offline-detect` | `runner.manage` | runner group/project | user/service account | yes | no | yes | Raw runner token returned only at registration/rotation. |
| POST | `/api/v1/runners/{id}/heartbeat` | runner token for same runner | runner | runner | yes | yes | no | Runner token scoped to URL runner id and validated by usecase. |
| POST | `/api/v1/runners/{id}/jobs/claim` | runner token for same runner | runner/job | runner | yes | yes | no | Claim must respect labels/capabilities/concurrency/lease. |
| POST | `/api/v1/runners/{id}/jobs/{job_id}/logs` | runner token and job ownership | runner/job | runner | yes | yes | no | Runner cannot append logs to unrelated jobs. |
| POST | `/api/v1/runners/{id}/jobs/{job_id}/status` | runner token and job ownership | runner/job | runner | yes | yes | no | Runner cannot update unrelated jobs. |
| POST | `/api/v1/jobs/{id}/logs`, `/api/v1/jobs/{id}/status` | `runner.manage` | runner/job | user/service account | yes | no | yes | Admin compatibility endpoints. |
| GET/POST | `/api/v1/releases*` | authenticated reads; `release.create` for create/plan/deploy | release/project | user/service account | yes | no | yes | ReleaseExecution remains foundation-level. |
| GET/POST | `/api/v1/deployments*` | authenticated reads; `deployment.create`, `deployment.cancel`, or `deployment.approve` for mutations | environment/target | user/service account | yes | no | yes | Apply/sync/rollback remain guarded and not default. |
| GET/POST | `/api/v1/host-groups`, `/api/v1/deployments/host/plan` | `environment.read`, `environment.write`, `deployment.create` | environment | user/service account | yes | no | yes | Remote host deploy is disabled by default. |
| GET/POST | `/api/v1/integrations/argocd/applications/*` | `deployment.create` | environment/target | user/service account | yes | no | yes | Argo sync is guarded; no production automation claim. |
| GET/POST | `/api/v1/approvals*` | authenticated reads; `deployment.approve` for decisions | environment/deployment | user/service account | yes | no | yes | Decisions must be audited. |
| GET/POST | `/api/v1/change-windows*`, `/api/v1/notifications*` | authenticated foundation routes | environment | user/service account | yes | no | yes | Real external notification delivery is not required in tests. |
| POST/GET | `/api/v1/security/scans*`, `/api/v1/policies/evaluate` | `policy.manage` for scan/evaluate; authenticated findings reads | policy/security | user/service account | yes | no | yes | Noop/fake scanners are foundation only. |
| GET/POST/PATCH/DELETE | `/api/v1/policies*` | `project.read` for list/get; `policy.manage` for create/update/disable | policy | user/service account | yes | no | yes | Policy catalog stores built-in gate definitions only; no OPA/Kyverno integration. |
| GET/POST | `/api/v1/cloud/*` | authenticated foundation routes | org/project | user/service account | yes | no | yes | Fake/skeleton inventory only; no cloud deployment. |
| GET | `/api/v1/audit/search`, `/api/v1/evidence/*`, `/api/v1/retention-policy` | `audit.read` | org/project/environment | user/service account | yes | no | yes | Auditor role can read audit/evidence. |
| POST | `/api/v1/retention-policy` | `policy.manage` | org/project/environment | user/service account | yes | no | yes | Retention mutation is policy-sensitive. |
| Placeholder | `/api/v1/audit-logs`, `/api/v1/events`, `/api/v1/logs`, `/api/v1/integrations` | placeholder | varies | user/service account | yes | no | yes | Structured `not_implemented`. |
| GET | `/api/v1/visualization/*` | authenticated; audit timeline requires `audit.read` | read model | user/service account | yes | no | yes | Backend visualization only, no frontend production claim. |
| GET/POST | `/api/v1/plugins*` | authenticated foundation routes | system | user/service account | yes | no | yes | Registry/capability metadata only; no unsafe dynamic loading. |
| GET/POST | `/api/v1/tenancy/*` | authenticated read; `project.write` for quota mutation | org/project | user/service account | yes | no | yes | Quotas are foundation-level. |

## Non-HTTP Control-Plane Surfaces

| Surface | Entry Point | Required Permission | Runner Token Allowed? | Notes |
|---|---|---|---|---|
| MCP stdio | `cmd/nivora-mcp`, `nivora mcp *` | existing RBAC by MCP resource/tool | no | MCP is not an HTTP route. It exposes read-only and plan-only operations; runner tokens and action tools are denied. |

## Required Negative Tests

The route suite must keep tests for these behaviors:

- unauthenticated requests are rejected in token/OIDC modes;
- viewers cannot mutate critical resources;
- developers can perform scoped deployment/pipeline operations they are granted;
- auditors can read audit routes but cannot mutate deployments;
- runner tokens can call only runner protocol endpoints and cannot access admin routes;
- secret values, credential values, runner token hashes, and API token hashes are not returned by normal responses.
