# MCP Tenant Scope Review

Current status: **partially proven for local MCP RBAC and scoped read models, including repository/workflow cross-project guards; not proven for complete remote multi-tenant exposure**.

## Scope Summary

| MCP Surface | Current Permission | Current Scope Behavior | Future Tenant Filter | Risk |
|---|---|---|---|---|
| capability/runtime/API inventory | `project.read` | global maturity metadata with scoped warning; not tenant inventory | optional remote-safe redaction profile | medium |
| PipelineRun resources/tools | `project.read` | explicit ID reads now check stored project scope when present; HTTP-created project-scoped runs persist project scope; HTTP detail/log/event/timeline and visualization reads are guarded by stored scope | broader application/environment ownership checks for pipeline-derived resources | medium-high for remote |
| DeploymentRun resources/tools | `project.read` or `deployment.create` | explicit ID reads now check stored project/environment scope when present; HTTP-created project-scoped runs persist project scope; HTTP detail/log/event/timeline/resource and visualization reads are guarded by stored scope | broader project/environment/target ownership checks for all deployment-derived resources | medium-high for remote |
| ReleaseExecution resources/tools | `project.read` or `deployment.create` | explicit ID reads and aggregate event searches check stored project/environment scope when present; HTTP plan/deploy project scope is copied to release targets and child DeploymentRuns; HTTP execution detail/timeline/target and visualization reads are guarded by stored scope | release/project/environment ownership check across all execution records | medium-high for remote |
| Artifact resources/tools | `project.read` | tracked artifacts and release-bound artifacts filter by stored project/environment metadata when present; HTTP scoped artifact creation persists project scope; MCP list/get/release binding and artifact event searches are scoped | broader registry/project ownership checks for registry-derived records | medium |
| runner summary | `project.read` | filters runners by `projectId`/`environmentId`/`orgId` labels when the subject is scoped; unscoped runners are hidden from scoped subjects; RunnerGroup metadata constrains registration and claim in the runtime path | remote-safe runner metadata profile | medium for remote |
| security summary | `project.read` | security scans now persist project/environment scope; MCP security findings, policy summaries, and security events filter by stored scan scope when present | broader subject ownership checks for historical unscoped records and evidence joins | medium |
| audit search | `audit.read` | HTTP scoped subjects are constrained to their own audit scope; MCP audit resource requires `audit.read` | mandatory remote scope filter and caps | high |
| evidence bundles | `audit.read` | HTTP evidence bundles for PipelineRun, DeploymentRun, and ReleaseExecution subjects are checked against stored project/environment scope and persisted with scope metadata when verifiable | add first-class scope metadata to release/artifact/security evidence | medium-high for remote |
| plugin capabilities | `project.read` | built-in metadata | usually global | low |
| plan-only local tools | `deployment.create` | local input only | input size and scope policy | medium |
| denied action tools | none | denied before permission grant | never expose action tier | low if kept denied |

## Tested Boundaries

- Anonymous subject is rejected.
- Runner token subject is rejected before normal RBAC.
- Viewer can read ordinary resources but cannot run plan-only tools.
- Developer can run plan-only tools.
- Registered MCP resources, tools, prompts, and blocked action names must appear in `docs/security/MCP_PERMISSION_MATRIX.md`; `TestMCPPermissionMatrixCoversCatalogEntries` fails on catalog drift.
- Plan-only tools and aliases return `mutated=false`; `TestMCPPlanOnlyToolsReturnMutatedFalseAndDoNotCreateDeploymentRuns` also checks they do not create DeploymentRuns.
- Auditor can read audit but cannot run plan-only tools.
- Service-account-like developer subject can use explicit plan permissions.
- Service-account-like viewer subject cannot use plan permissions.
- Audit resource requires `audit.read`.
- Project-scoped service account can list only same-project repository catalog records through MCP.
- Project-scoped service account is denied when reading another project's repository detail, latest snapshot metadata, repository intelligence, or DevOps plan resource through MCP.
- Project-scoped service account is denied when invoking repository snapshot, repository intelligence, repository DevOps plan, DevOps readiness review, or workflow draft tools against another project's repository.
- Project-scoped service account can list only same-project WorkflowPlan summaries and is denied when reading another project's stored WorkflowPlan or workflow detail through MCP.
- List-like MCP resource URIs support `limit` and `offset` query parameters for events, logs, pipeline definitions, releases, artifacts, security findings, policy results, audit search, and evidence bundles; response content returns the base resource URI so query strings cannot leak token-like values into resource content.
- Invalid MCP resource pagination query values return structured `mcp_invalid_arguments` errors.
- Project-scoped service account can read its own scoped PipelineRun record, logs, and timeline.
- Project-scoped service account is denied when reading another project's scoped PipelineRun.
- Aggregate MCP event and log searches filter out scoped PipelineRun records outside the subject project.
- Project-scoped service account can list and read only same-project PipelineDefinition records through MCP resources/tools; requested cross-project list filters are ignored in favor of the subject project.
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
- HTTP evidence bundle generation and direct bundle reads are scope-checked for PipelineRun, DeploymentRun, and ReleaseExecution subjects.
- HTTP project-scoped auditors cannot read another project's evidence bundle by subject path, bundle id, or evidence list.
- Project-scoped service account can read its own scoped SecurityScan findings and policy summary through MCP.
- Project-scoped service account does not see another project's SecurityScan findings, policy result summary, or security events through MCP aggregate resources/tools.
- HTTP project-scoped service account creates SecurityScan records with project scope and cannot list, fetch, or fetch findings for another project's SecurityScan.
- Project-scoped service account can read its own scoped Artifact catalog records and release bindings through MCP.
- Project-scoped service account is denied when directly reading another project's Artifact by ID through MCP.
- MCP aggregate event search filters release/artifact events by stored release project scope.
- HTTP project-scoped service account creates Artifact records with project scope and cannot list, fetch, or fetch release bindings for another project's Artifact.
- Project-scoped service account sees only same-project runner records in MCP runner summary resource/tool output.
- Project-scoped HTTP subject sees only same-project runners in `/api/v1/runners` and `/api/v1/visualization/runners/summary`.
- Project-scoped HTTP subject is forbidden from reading, rotating, revoking, or globally marking offline another project's runner.
- Project-scoped MCP capability status includes an explicit scope object and warning that capability status is global maturity metadata, not tenant inventory.
- Remote HTTP MCP audit attribution records request id, correlation id, MCP client id, transport, and remote address when available, while redacting secret-like values.
- Remote HTTP MCP returns structured JSON-RPC errors for unknown methods, resources, and tools.
- Remote HTTP MCP enforces configured per-subject request rate limits and transport response caps in route-level tests.

Evidence:

- `internal/api/mcp/server.go`
- `internal/api/mcp/server_test.go`
- `internal/api/mcp/scenario_test.go`
- `internal/api/http/handlers/security.go`
- `internal/api/http/handlers/deployments.go`
- `internal/api/http/handlers/release_orchestration.go`
- `internal/api/http/routes/security_test.go`
- `internal/api/http/routes/tenant_isolation_test.go`
- `docs/security/MCP_PERMISSION_MATRIX.md`

## Gaps

| Gap | Impact | Required Work |
|---|---|---|
| Resource ID ownership not checked for every future MCP read | A remote subject could request another tenant's ID if new resources skip scoped guards | Extend the scoped guard pattern to every future resource/tool and keep route/resource contract tests as MCP grows. |
| Audit and observability scope is not complete for every historical or unscoped record family | Older unscoped records can only be hidden or treated as global, which limits confidence for remote multi-tenant exposure | Add first-class scope metadata to every persisted evidence/audit/resource family and keep negative tests for historical unscoped records. |
| Runner ownership is still metadata-based | RunnerGroup records now constrain runtime registration and claim, but MCP runner summaries still expose only scoped metadata and are not a remote fleet-management contract | Add remote-safe runner metadata profile before exposing remote MCP runner fleet views. |
| Capability/runtime documents are broad | Metadata can reveal unsupported or experimental areas | Define a remote-safe capability summary profile before exposing remote MCP. |
| Plan-only tool input scope is local | Repository-backed workflow draft and DevOps planning tools enforce repository scope, but raw local workflow/manifests still need stronger subject-scope hints before broad remote use | Add plan-input ownership hints and policy checks before broader remote exposure. |

## Recommendation

Keep local stdio MCP available for maintainer workflows. Do not broadly expose remote MCP until every future resource/tool keeps explicit tenant-scope tests, historical unscoped records are handled consistently, distributed rate limiting is designed for multi-replica deployments, and operator guidance covers the remote audit, rate-limit, and deployment model.
