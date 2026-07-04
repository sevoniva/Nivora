# MCP Tenant Scope Review

Current status: **partially proven for local MCP RBAC, not proven for complete remote multi-tenant exposure**.

## Scope Summary

| MCP Surface | Current Permission | Current Scope Behavior | Future Tenant Filter | Risk |
|---|---|---|---|---|
| capability/runtime/API inventory | `project.read` | global metadata summary | optional org/project redaction | medium |
| PipelineRun resources/tools | `project.read` | explicit ID reads now check stored project scope when present; HTTP-created project-scoped runs persist project scope; HTTP detail/log/event/timeline and visualization reads are guarded by stored scope | broader application/environment ownership checks for pipeline-derived resources | medium-high for remote |
| DeploymentRun resources/tools | `project.read` or `deployment.create` | explicit ID reads now check stored project/environment scope when present; HTTP-created project-scoped runs persist project scope; HTTP detail/log/event/timeline/resource and visualization reads are guarded by stored scope | broader project/environment/target ownership checks for all deployment-derived resources | medium-high for remote |
| ReleaseExecution resources/tools | `project.read` or `deployment.create` | explicit ID reads and aggregate event searches check stored project/environment scope when present; HTTP plan/deploy project scope is copied to release targets and child DeploymentRuns; HTTP execution detail/timeline/target and visualization reads are guarded by stored scope | release/project/environment ownership check across all execution records | medium-high for remote |
| runner summary | `project.read` | fleet summary | runner group/environment filter | high for remote |
| security summary | `project.read` | service summary | project/environment filter | medium |
| audit search | `audit.read` | HTTP scoped subjects are constrained to their own audit scope; MCP audit resource requires `audit.read` | mandatory remote scope filter and caps | high |
| plugin capabilities | `project.read` | built-in metadata | usually global | low |
| plan-only local tools | `deployment.create` | local input only | input size and scope policy | medium |
| denied action tools | none | denied before permission grant | never expose action tier | low if kept denied |

## Tested Boundaries

- Anonymous subject is rejected.
- Runner token subject is rejected before normal RBAC.
- Viewer can read ordinary resources but cannot run plan-only tools.
- Developer can run plan-only tools.
- Auditor can read audit but cannot run plan-only tools.
- Service-account-like developer subject can use explicit plan permissions.
- Service-account-like viewer subject cannot use plan permissions.
- Audit resource requires `audit.read`.
- Project-scoped service account can read its own scoped PipelineRun record, logs, and timeline.
- Project-scoped service account is denied when reading another project's scoped PipelineRun.
- Aggregate MCP event and log searches filter out scoped PipelineRun records outside the subject project.
- HTTP project-scoped service account can read its own PipelineRun detail, logs, events, timeline, and visualization DAG/timeline/summary.
- HTTP project-scoped service account is denied when directly reading another project's PipelineRun detail and visualization endpoints by ID.
- HTTP aggregate `/api/v1/events`, `/api/v1/logs`, and visualization audit timeline are scoped for PipelineRun records.
- Project-scoped service account can read its own scoped DeploymentRun record, health, and diff.
- Project-scoped service account is denied when reading another project's scoped DeploymentRun.
- Aggregate MCP event and log searches filter out scoped DeploymentRun records outside the subject project.
- HTTP project-scoped service account can read its own DeploymentRun detail, plan, resources, health, diff, snapshot, rollback plan, logs, events, timeline, and security summary.
- HTTP project-scoped service account is denied when directly reading another project's DeploymentRun detail and visualization endpoints by ID.
- HTTP aggregate `/api/v1/events`, `/api/v1/logs`, and visualization audit timeline use scoped DeploymentRun lists.
- Project-scoped service account can read its own scoped ReleaseExecution record and timeline.
- Project-scoped service account is denied when reading another project's scoped ReleaseExecution.
- Aggregate MCP event searches filter out scoped ReleaseExecution records outside the subject project.
- HTTP project-scoped service account can read its own ReleaseExecution detail, timeline, and target list.
- HTTP project-scoped service account is denied when directly reading another project's ReleaseExecution detail and visualization endpoints by ID.
- HTTP aggregate `/api/v1/events` and visualization audit timeline filter ReleaseExecution records by stored target project/environment scope.

Evidence:

- `internal/api/mcp/server.go`
- `internal/api/mcp/server_test.go`
- `internal/api/mcp/scenario_test.go`
- `internal/api/http/handlers/deployments.go`
- `internal/api/http/handlers/release_orchestration.go`
- `internal/api/http/routes/tenant_isolation_test.go`
- `docs/security/MCP_PERMISSION_MATRIX.md`

## Gaps

| Gap | Impact | Required Work |
|---|---|---|
| Resource ID ownership not checked for every MCP read | A remote subject could request another tenant's ID if underlying stores do not filter | Extend the scoped PipelineRun/DeploymentRun/ReleaseExecution guard pattern to PipelineRun definitions, artifact bindings, security summaries, and evidence bundles. |
| Audit and observability scope is not complete for every resource family | Sensitive operational metadata from artifact/security/evidence records could cross scopes if exposed remotely | Extend project/environment ownership metadata and tests to artifacts, security scans, and evidence bundles before remote MCP or broad tenant exposure. |
| Runner summary is global | Runner fleet metadata can reveal other environments | Filter by runner group/project/environment. |
| Capability/runtime documents are broad | Metadata can reveal unsupported or experimental areas | Decide what is safe for remote subjects. |
| Plan-only tool input scope is local | Remote plan-only tools need body limits and policy checks | Add input size and subject-scope validation. |

## Recommendation

Keep local stdio MCP available for maintainer workflows. Do not expose remote MCP until every resource and tool has explicit tenant-scope tests and the audit path includes remote request metadata.
