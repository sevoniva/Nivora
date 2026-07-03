# Stuck JobRun lease review

## Evidence Used

- Resources: `nivora://system/runtime`, `nivora://pipelines/runs/{id}/timeline`
- Tools: `nivora_status`, `nivora_get_pipeline_timeline`
- Prompt: `diagnose_pipeline_run`

## Facts

- Runtime summary can show recovery counters and warnings.
- PipelineRun timeline can show recorded state transitions.

## Inference

- A lease problem can be suggested only when runtime counters or timeline evidence point to expired work.

## Unknowns

- Exact database lease timestamps and runner process state may be unavailable through MCP.

## Blocked Actions

- Do not register runners or mutate runner leases through MCP.

## Safe Next Checks

- Read runner summary.
- Use normal runtime recovery diagnostics outside MCP.

## Permissions

- Requires `project.read`.

## Safety Notes

- Do not present a suspected stale lease as proven without direct state evidence.
