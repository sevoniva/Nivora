# Repository and Workflow Development

This guide covers the current local/foundation repository workflow path. It is safe for local development and CI tests because it does not require external SCM credentials or network access.

## Inspect A Local Repository

```bash
nivora repository inspect --path . --name nivora-local
```

The command prints a repository snapshot and static intelligence:

- detected languages
- detected build/test/package command candidates
- detected deployment/workflow files
- recommended Nivora Workflow draft
- warnings

Detected commands are suggestions only. They are not executed by repository intelligence.

When the server or MCP process is configured with `database.runtime_store: postgres`, repository snapshots, repository intelligence, and workflow plan records are stored in PostgreSQL. The local `nivora repository inspect --path` and `nivora workflow plan --file` commands remain local commands and do not require a database.

## Plan Repository DevOps Candidates

After a repository snapshot exists, Nivora can derive a plan-only DevOps summary:

```bash
nivora repository devops-plan <repository-id> --server http://localhost:8080
```

The summary includes build, test, package, security scan, deployment target, and release-candidate metadata from the latest saved snapshot. It does not execute detected commands, run scanners, create releases, bind artifacts, deploy, or mutate runtime state. The response includes `mutated=false` for API, CLI, and MCP callers.

## Validate A Nivora Workflow

```bash
nivora workflow validate --file examples/workflows/go-ci.yaml
nivora workflow plan --file examples/workflows/go-ci.yaml
nivora workflow run --file examples/workflows/go-ci.yaml --confirm --allow-pipeline-run --server http://localhost:8080
nivora workflow cancel <workflow-run-id> --server http://localhost:8080
```

The planner builds a DAG and checks dependency cycles, missing `needs` targets, matrix size limits, unsupported `uses`, artifact/cache declarations, plan-only security/release/deployment intent, and secret-like environment or intent values.
The run command is server-backed and queue-only. It creates WorkflowRun metadata plus a queued PipelineRun when the server authorizes the request; it does not execute workflow steps inside the CLI process.

Workflow-level `artifacts` and `cache` entries are recorded as PipelineRun metadata when a guarded WorkflowRun queues a PipelineRun. The control plane records names, paths, cache keys, restore keys, retention hints, and metadata. It does not read artifact files, upload cache blobs, or store large content in the database.

Workflow-level `security`, `release`, and `deployment` sections are plan-only intent summaries. They can show scanner, release, digest, target, apply, or sync intent in a plan, but the workflow planner does not run scanners, create releases, bind artifacts, apply Kubernetes manifests, sync Argo CD, or deploy hosts.

## API Surface

Repository snapshot/intelligence:

```text
POST /api/v1/repositories/{id}/snapshot
GET  /api/v1/repositories/{id}/snapshots
GET  /api/v1/repositories/{id}/intelligence
POST /api/v1/repositories/{id}/analyze
POST /api/v1/devops/plan
```

Workflow plan-only endpoints:

```text
GET  /api/v1/workflows
POST /api/v1/workflows/validate
POST /api/v1/workflows/plan
GET  /api/v1/workflows/{id}/plan
GET  /api/v1/workflows/plans
GET  /api/v1/workflows/plans/{id}
GET  /api/v1/workflows/runs
GET  /api/v1/workflows/runs/{id}
POST /api/v1/workflows/runs/{id}/cancel
POST /api/v1/workflows/run
```

`/api/v1/workflows/run` requires `confirm=true` and `allowPipelineRun=true`. It creates a WorkflowRun metadata record and queues a PipelineRun through the existing runtime; it does not execute shell steps in the HTTP handler.

`GET /api/v1/workflows/runs` and `GET /api/v1/workflows/runs/{id}` refresh WorkflowRun status from the linked PipelineRun when that PipelineRun is still available in the configured runtime store.

`POST /api/v1/workflows/runs/{id}/cancel` cancels the linked PipelineRun through the existing PipelineRun runtime and updates WorkflowRun metadata. It does not retry the workflow, roll back deployments, delete resources, or execute workflow steps directly.

PipelineRun metadata read endpoints:

```text
GET /api/v1/pipeline-runs/{id}/artifacts
GET /api/v1/pipeline-runs/{id}/caches
GET /api/v1/pipeline-runs/{id}/annotations
GET /api/v1/pipeline-runs/{id}/summary
```

These endpoints return control-plane metadata only. Artifact and cache blobs are not returned by the API. Large step summaries should use a storage reference instead of inline content.

## MCP Surface

Local MCP tools and resources:

```text
nivora_repository_inspect
nivora_repository_devops_plan
nivora_workflow_validate
nivora_workflow_plan
nivora://repositories/{id}/devops-plan
nivora://workflows
nivora://workflows/{id}/plan
nivora://pipelines/runs/{id}/artifacts
nivora://pipelines/runs/{id}/caches
nivora://pipelines/runs/{id}/annotations
nivora://pipelines/runs/{id}/summary
```

Each tool is read-only or plan-only and returns `mutated=false`. MCP does not execute workflow steps, repository commands, scanner runs, release creation, or deployment actions.

## Known Limits

- RepositorySnapshot and RepositoryIntelligence are durable only in configured PostgreSQL server/MCP mode; local commands and default development mode still use in-memory state or direct local output.
- GitHub/GitLab/Gitea real network integrations are not implemented.
- WorkflowPlan record persistence exists in configured PostgreSQL server/MCP mode, but raw WorkflowDefinition YAML is not stored by that plan-record store.
- Repository DevOps plans depend on the latest saved snapshot. They are not generated from live remote SCM state and do not create Release, DeploymentRun, SecurityScan, or PipelineRun records.
- WorkflowRun metadata persistence exists in configured PostgreSQL server mode and points to the queued PipelineRun. Read and cancel APIs synchronize through the linked PipelineRun. Workflow-level retry semantics remain future work.
- Workflow security/release/deployment intent is stored in redacted plan records and remains plan-only. It is not release orchestration, deployment execution, or scanner execution.
- Pipeline artifact/cache/annotation/summary metadata persistence exists in memory and configured PostgreSQL runtime stores. Blob storage is not implemented by this metadata foundation; use storage references for content outside the control plane.
- Workflow execution still belongs to PipelineRun, Runner, and Worker paths; MCP does not expose workflow action execution.
- Shell execution is not a sandbox.
