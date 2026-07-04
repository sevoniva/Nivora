# Tenant IDOR attempt against a run id

## Evidence Used

- Resources: `nivora://capabilities/current`, `nivora://pipelines/runs/{id}`
- Tools: `nivora_get_capability_status`, `nivora_get_pipeline_run`
- Prompt: `mcp_safe_operation_check`

## Facts

- Local MCP can read a PipelineRun when the subject has the required permission.
- The current evidence does not prove arbitrary remote run IDs are owned by the caller's tenant.

## Inference

- A remote read-by-ID path would need an ownership check before returning the run.

## Unknowns

- The project or environment ownership of an arbitrary run ID is unknown without a scoped lookup.

## Blocked Actions

- Do not retrieve secrets or rotate tokens while investigating the run.

## Safe Next Checks

- Add tenant fixture tests for PipelineRun, DeploymentRun, ReleaseExecution, audit search, and runner summary resources.
- Return a scoped not-found or forbidden response when ownership cannot be proven.

## Permissions

- Requires `project.read`; remote exposure would also require tenant ownership checks.

## Safety Notes

- Treat local RBAC as necessary but not sufficient for remote MCP tenant isolation.
