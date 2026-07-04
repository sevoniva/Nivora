# MCP Tenant Scope Review

Current status: **partially proven for local MCP RBAC, not proven for complete remote multi-tenant exposure**.

## Scope Summary

| MCP Surface | Current Permission | Current Scope Behavior | Future Tenant Filter | Risk |
|---|---|---|---|---|
| capability/runtime/API inventory | `project.read` | global metadata summary | optional org/project redaction | medium |
| PipelineRun resources/tools | `project.read` | explicit ID reads now check stored project scope when present; HTTP-created project-scoped runs persist project scope | broader application/environment ownership checks for pipeline-derived resources | medium-high for remote |
| DeploymentRun resources/tools | `project.read` or `deployment.create` | explicit ID reads now check stored project/environment scope when present; HTTP-created project-scoped runs persist project scope | broader project/environment/target ownership checks for all deployment-derived resources | medium-high for remote |
| ReleaseExecution resources/tools | `project.read` or `deployment.create` | explicit ID reads and aggregate event searches check stored project/environment scope when present; HTTP plan/deploy project scope is copied to release targets and child DeploymentRuns | release/project/environment ownership check across all execution records | medium-high for remote |
| runner summary | `project.read` | fleet summary | runner group/environment filter | high for remote |
| security summary | `project.read` | service summary | project/environment filter | medium |
| audit search | `audit.read` | filter arguments supported | mandatory scope filter and caps | high |
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
- Project-scoped service account can read its own scoped DeploymentRun record, health, and diff.
- Project-scoped service account is denied when reading another project's scoped DeploymentRun.
- Aggregate MCP event and log searches filter out scoped DeploymentRun records outside the subject project.
- Project-scoped service account can read its own scoped ReleaseExecution record and timeline.
- Project-scoped service account is denied when reading another project's scoped ReleaseExecution.
- Aggregate MCP event searches filter out scoped ReleaseExecution records outside the subject project.

Evidence:

- `internal/api/mcp/server.go`
- `internal/api/mcp/server_test.go`
- `internal/api/mcp/scenario_test.go`
- `docs/security/MCP_PERMISSION_MATRIX.md`

## Gaps

| Gap | Impact | Required Work |
|---|---|---|
| Resource ID ownership not checked for every MCP read | A remote subject could request another tenant's ID if underlying stores do not filter | Extend the scoped PipelineRun/DeploymentRun/ReleaseExecution guard pattern to PipelineRun definitions, artifact bindings, security summaries, and evidence bundles. |
| Audit search can be broad | Sensitive operational metadata could cross scopes | Enforce scope defaults and pagination before remote exposure. |
| Runner summary is global | Runner fleet metadata can reveal other environments | Filter by runner group/project/environment. |
| Capability/runtime documents are broad | Metadata can reveal unsupported or experimental areas | Decide what is safe for remote subjects. |
| Plan-only tool input scope is local | Remote plan-only tools need body limits and policy checks | Add input size and subject-scope validation. |

## Recommendation

Keep local stdio MCP available for maintainer workflows. Do not expose remote MCP until every resource and tool has explicit tenant-scope tests and the audit path includes remote request metadata.
