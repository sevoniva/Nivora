# Visualization APIs

Phase 4.0 exposes backend-only visualization APIs for future UI work. No frontend is implemented in this phase.

## PipelineRun Views

```sh
curl http://localhost:8080/api/v1/visualization/pipeline-runs/<pipeline-run-id>/dag
curl http://localhost:8080/api/v1/visualization/pipeline-runs/<pipeline-run-id>/timeline
curl http://localhost:8080/api/v1/visualization/pipeline-runs/<pipeline-run-id>/summary
```

The DAG endpoint returns `GraphNode` and `GraphEdge` DTOs derived from PipelineRun, StageRun, JobRun, and StepRun records.

## DeploymentRun Views

```sh
curl http://localhost:8080/api/v1/visualization/deployments/<deployment-run-id>/timeline
curl http://localhost:8080/api/v1/visualization/deployments/<deployment-run-id>/resources
curl http://localhost:8080/api/v1/visualization/deployments/<deployment-run-id>/diff
curl http://localhost:8080/api/v1/visualization/deployments/<deployment-run-id>/health
```

These endpoints reuse DeploymentRun timeline, resource inventory, diff, and health records. They do not query a live Kubernetes cluster by default.

## Release and Environment Views

```sh
curl http://localhost:8080/api/v1/visualization/releases/<release-id>/overview
curl http://localhost:8080/api/v1/visualization/releases/executions/<execution-id>/timeline
curl http://localhost:8080/api/v1/visualization/releases/executions/<execution-id>/targets
curl http://localhost:8080/api/v1/visualization/environments/<environment-id>/topology
```

The environment topology endpoint is derived from currently known deployment records. It is not a live discovery API.

## Dashboard Views

```sh
curl http://localhost:8080/api/v1/visualization/runners/summary
curl http://localhost:8080/api/v1/visualization/security/summary
curl http://localhost:8080/api/v1/visualization/audit/timeline
```

The audit timeline aggregates audit records already held by runtime services. It does not replace durable audit storage.

## Limitations

- No frontend is included.
- No charting or layout engine is included.
- No live environment discovery is performed.
- In-memory runtime mode only shows records created in the current process.
- Nivora remains a hardened beta-candidate foundation and is not production-ready.
