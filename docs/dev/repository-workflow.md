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

When the server or MCP process is configured with `database.runtime_store: postgres`, repository snapshots and repository intelligence are stored in PostgreSQL. The local `nivora repository inspect --path` command remains a local inspection command and does not require a database.

## Validate A Nivora Workflow

```bash
nivora workflow validate --file examples/workflows/go-ci.yaml
nivora workflow plan --file examples/workflows/go-ci.yaml
```

The planner builds a DAG and checks dependency cycles, missing `needs` targets, matrix size limits, unsupported `uses`, and secret-like environment values.

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
POST /api/v1/workflows/validate
POST /api/v1/workflows/plan
POST /api/v1/workflows/run
```

`/api/v1/workflows/run` is a guarded placeholder and returns `not_implemented`.

## MCP Surface

Local MCP tools:

```text
nivora_repository_inspect
nivora_workflow_validate
nivora_workflow_plan
```

Each tool is read-only or plan-only and returns `mutated=false`. MCP does not execute workflow steps.

## Known Limits

- RepositorySnapshot and RepositoryIntelligence are durable only in configured PostgreSQL server/MCP mode; local commands and default development mode still use in-memory state or direct local output.
- GitHub/GitLab/Gitea real network integrations are not implemented.
- WorkflowDefinition and WorkflowPlan persistence are not implemented.
- WorkflowRun persistence is not implemented.
- Workflow execution must continue through explicit PipelineRun work; direct workflow run remains future work.
- Shell execution is not a sandbox.
