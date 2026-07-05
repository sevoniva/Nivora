# Repository and Workflow Development

This guide covers the current local/foundation repository workflow path. It is safe for local development and CI tests because it does not require external SCM credentials or network access.

## Create A Repository Catalog Record

Repository catalog records can be created from a file so private and offline environments can keep repository metadata under review:

```bash
nivora repository create --file examples/repositories/generic-git.yaml --server http://localhost:8080
```

The file format uses `kind: Repository` with metadata and spec fields. Supported catalog provider values in this foundation path are `generic`, `generic_git` (normalized to `generic`), `github`, `gitlab`, `gitea`, `local`, and `archive`. `github`, `gitlab`, and `gitea` are provider metadata and adapter-skeleton labels in this path; Nivora does not call their product APIs, push commits, open pull requests, or resolve CredentialRef secret values during catalog create.

Example files live in `examples/repositories/`. They use placeholder URLs and CredentialRef names only. Do not put tokens, SSH keys, passwords, or inline `https://user:password@...` URLs in repository definitions.

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

## Snapshot And Analyze Through The Server

For cataloged repositories, the CLI can call the server-backed read-only snapshot and analysis APIs:

```bash
nivora repository snapshot <repository-id> --local-path . --ref HEAD --server http://localhost:8080
nivora repository analyze <repository-id> --server http://localhost:8080
```

The snapshot command stores repository metadata for the configured local/generic provider. The analyze command refreshes static intelligence from the latest saved snapshot. These commands do not clone remote providers, resolve CredentialRef values, execute repository scripts, run scanners, create releases, or deploy.

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
nivora workflow run --file examples/workflows/go-ci.yaml --repository-snapshot-id snap-dev --confirm --allow-pipeline-run --server http://localhost:8080
nivora workflow cancel <workflow-run-id> --server http://localhost:8080
nivora workflow reconcile --repository-id repo-dev --server http://localhost:8080
nivora workflow retry <workflow-run-id> --confirm --allow-pipeline-run --server http://localhost:8080
```

The planner builds a DAG and checks dependency cycles, missing `needs` targets, matrix size limits, unsupported `uses`, artifact/cache declarations, workflow `permissions`, plan-only security/release/deployment intent, and secret-like environment or intent values.
The run command is server-backed and queue-only. It creates WorkflowRun metadata plus a queued PipelineRun when the server authorizes the request; it does not execute workflow steps inside the CLI process.
The cancel command asks the server to cancel the linked PipelineRun. The reconcile command scans non-terminal WorkflowRun metadata and refreshes status from linked PipelineRun state. The retry command requires explicit confirmation and queues a replacement PipelineRun from the stored WorkflowPlan for Failed, Canceled, or Timeout WorkflowRuns. These commands do not roll back deployments, delete resources, or execute workflow steps directly.

Workflow-level `permissions` entries are recorded as plan-only permission requests. They make workflow intent visible to API, CLI, and MCP callers, but they do not grant runtime access by themselves. Write, run, admin, and `id-token` style requests produce security warnings and remain subject to Nivora RBAC, runner policy, explicit confirmation, and configured unsafe-operation gates.

Workflow job `labels` are preserved in WorkflowPlan records and in the generated PipelineRun definition. Runner claim paths in both memory and PostgreSQL stores require a runner to carry matching job labels before it can claim the job. These labels are scheduling metadata only; they do not sandbox shell execution, and secret-like label keys or values are rejected.

Workflow source metadata is also preserved for traceability. When a guarded WorkflowRun queues a PipelineRun, the PipelineRun stores `workflowId`, `workflowPlanId`, `workflowRunId`, `repositoryId`, and `repositorySnapshotId`. Generated JobRun and StepRun records store `workflowJobId` and `workflowStepId`. These fields help audit a runtime record back to a repository snapshot and workflow plan; they do not grant permissions, execute code, or replace runner policy checks.

## Events And Audit

Repository create, catalog validation, snapshot, intelligence refresh, DevOps plan, and readiness-review paths record metadata-only events and audit entries through the configured store. In PostgreSQL mode, audit entries are written through the shared hash-chained audit writer used by the rest of the control plane.

Workflow validate, plan, run, cancel, retry, and reconcile paths also record metadata-only events and audit entries. Validation does not persist a WorkflowPlan record or raw workflow YAML; it records that a definition was validated and returns the planned view. Workflow plan/run lifecycle events are keyed by the workflow plan ID, workflow run ID, or workflow ID so timeline and audit views can connect repository intelligence, workflow planning, and PipelineRun metadata without storing secret values.

MCP repository inspection/planning and workflow planning calls record separate MCP audit actions (`devops.mcp.repository.inspected`, `devops.mcp.workflow.planned`) in addition to the generic MCP tool audit. These audit entries identify the safe operation that was performed but do not store raw workflow content, local file contents, CredentialRef values, or secret values.

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
POST /api/v1/workflows/runs/{id}/retry
POST /api/v1/workflows/runs/reconcile
POST /api/v1/workflows/run
```

`/api/v1/workflows/run` requires `confirm=true` and `allowPipelineRun=true`. It creates a WorkflowRun metadata record and queues a PipelineRun through the existing runtime; it does not execute shell steps in the HTTP handler.

`GET /api/v1/workflows/runs` and `GET /api/v1/workflows/runs/{id}` refresh WorkflowRun status from the linked PipelineRun when that PipelineRun is still available in the configured runtime store.

`POST /api/v1/workflows/runs/{id}/cancel` cancels the linked PipelineRun through the existing PipelineRun runtime and updates WorkflowRun metadata. It does not retry the workflow, roll back deployments, delete resources, or execute workflow steps directly.

`POST /api/v1/workflows/runs/{id}/retry` requires `confirm=true` and `allowPipelineRun=true`. It retries only Failed, Canceled, or Timeout WorkflowRuns by creating a new queued PipelineRun from the stored WorkflowPlan.

`POST /api/v1/workflows/runs/reconcile` scans non-terminal WorkflowRun records and repairs status drift from linked PipelineRun state. It is a metadata reconciliation path only.

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

- `make verify-workflow` runs the workflow planner unit tests and validates/plans the checked-in workflow examples without external SCM, registry, Kubernetes, or runner services.
- RepositorySnapshot and RepositoryIntelligence are durable only in configured PostgreSQL server/MCP mode; local commands and default development mode still use in-memory state or direct local output.
- GitHub/GitLab/Gitea real network integrations are not implemented.
- WorkflowPlan record persistence exists in configured PostgreSQL server/MCP mode, but raw WorkflowDefinition YAML is not stored by that plan-record store.
- Repository DevOps plans depend on the latest saved snapshot. They are not generated from live remote SCM state and do not create Release, DeploymentRun, SecurityScan, or PipelineRun records.
- WorkflowRun metadata persistence exists in configured PostgreSQL server mode and points to the queued PipelineRun. Workflow and repository source IDs are copied into the linked PipelineRun/JobRun/StepRun records, including `repositorySnapshotId` when supplied. Read, cancel, retry, and reconcile APIs synchronize through the linked PipelineRun. Background retry policy controls and automatic workflow reconciliation remain future work.
- Workflow security/release/deployment intent is stored in redacted plan records and remains plan-only. It is not release orchestration, deployment execution, or scanner execution.
- Pipeline artifact/cache/annotation/summary metadata persistence exists in memory and configured PostgreSQL runtime stores. Blob storage is not implemented by this metadata foundation; use storage references for content outside the control plane.
- Workflow execution still belongs to PipelineRun, Runner, and Worker paths; MCP does not expose workflow action execution.
- Shell execution is not a sandbox.
