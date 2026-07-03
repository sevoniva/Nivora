# MCP Example: Diagnose PipelineRun

Safe local workflow:

1. Ask the MCP client to list resources with `resources/list`.
2. Read `nivora://pipelines/runs/<pipeline-run-id>`.
3. Read `nivora://pipelines/runs/<pipeline-run-id>/timeline`.
4. Read `nivora://pipelines/runs/<pipeline-run-id>/logs`.
5. Use the `diagnose_pipeline_run` prompt.

Expected AI behavior:

- Cite the resource URIs it used.
- Separate facts from inference.
- List unknowns.
- Recommend only read-only checks or plan-only follow-up.
- Do not rerun the pipeline through MCP.

This example is credential-free and does not require external services.
