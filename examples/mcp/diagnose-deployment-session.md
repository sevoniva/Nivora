# MCP Example: Diagnose DeploymentRun

Safe local workflow:

1. Read `nivora://deployments/<deployment-run-id>`.
2. Read `nivora://deployments/<deployment-run-id>/timeline`.
3. Read `nivora://deployments/<deployment-run-id>/resources`.
4. Read `nivora://deployments/<deployment-run-id>/health`.
5. Read `nivora://deployments/<deployment-run-id>/diff`.
6. Use the `diagnose_deployment_run` prompt.

Expected AI behavior:

- Treat apply, sync, rollback, prune, and host deploy as blocked actions.
- Explain health and diff using Nivora evidence.
- List missing live cluster evidence as unknowns.
- Do not claim production readiness.

This example is safe for local validation and does not require Kubernetes.
