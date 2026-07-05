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

## Validate A Nivora Workflow

```bash
nivora workflow validate --file examples/workflows/go-ci.yaml
nivora workflow plan --file examples/workflows/go-ci.yaml
nivora workflow run --file examples/workflows/go-ci.yaml --confirm --allow-pipeline-run --server http://localhost:8080
```

The planner builds a DAG and checks dependency cycles, missing `needs` targets, matrix size limits, unsupported `uses`, and secret-like environment values.
The run command is server-backed and queue-only. It creates WorkflowRun metadata plus a queued PipelineRun when the server authorizes the request; it does not execute workflow steps inside the CLI process.

## API Surface

Repository snapshot/intelligence:

```text
POST /api/v1/repositories/{id}/snapshot
GET  /api/v1/repositories/{id}/snapshots
GET  /api/v1/repositories/{id}/intelligence
POST /api/v1/repositories/{id}/analyze
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
POST /api/v1/workflows/run
```

`/api/v1/workflows/run` requires `confirm=true` and `allowPipelineRun=true`. It creates a WorkflowRun metadata record and queues a PipelineRun through the existing runtime; it does not execute shell steps in the HTTP handler.

`GET /api/v1/workflows/runs` and `GET /api/v1/workflows/runs/{id}` refresh WorkflowRun status from the linked PipelineRun when that PipelineRun is still available in the configured runtime store.

## MCP Surface

Local MCP tools and resources:

```text
nivora_repository_inspect
nivora_workflow_validate
nivora_workflow_plan
nivora://workflows
nivora://workflows/{id}/plan
```

Each tool is read-only or plan-only and returns `mutated=false`. MCP does not execute workflow steps.

## Known Limits

- RepositorySnapshot and RepositoryIntelligence are durable only in configured PostgreSQL server/MCP mode; local commands and default development mode still use in-memory state or direct local output.
- GitHub/GitLab/Gitea real network integrations are not implemented.
- WorkflowPlan record persistence exists in configured PostgreSQL server/MCP mode, but raw WorkflowDefinition YAML is not stored by that plan-record store.
- WorkflowRun metadata persistence exists in configured PostgreSQL server mode and points to the queued PipelineRun. Read APIs refresh the metadata status from the linked PipelineRun, but workflow-level retry/cancel semantics are still owned by the PipelineRun runtime.
- Workflow execution still belongs to PipelineRun, Runner, and Worker paths; MCP does not expose workflow action execution.
- Shell execution is not a sandbox.
