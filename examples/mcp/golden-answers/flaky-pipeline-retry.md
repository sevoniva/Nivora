# Flaky PipelineRun retry review

## Evidence Used

- Resources: `nivora://pipelines/runs/{id}`, `nivora://pipelines/runs/{id}/timeline`, `nivora://pipelines/runs/{id}/logs`
- Tool: `nivora_explain_pipeline_failure`
- Prompt: `diagnose_pipeline_run`

## Facts

- MCP can explain the current failed run from Nivora state and redacted logs.
- MCP does not expose a rerun action.

## Inference

- Flakiness is only a hypothesis unless multiple historical runs or dependency signals support it.

## Unknowns

- Historical pass/fail rate and external service health are not proven by this scenario.

## Blocked Actions

- Do not rerun, register runners, or rotate tokens through MCP.

## Safe Next Checks

- Compare previous PipelineRuns through normal read paths.
- Check runner fleet health.

## Permissions

- Requires `project.read`; plan-only explanation requires `deployment.create`.

## Safety Notes

- Keep retry advice as a recommendation for a guarded control-plane path, not an MCP mutation.
