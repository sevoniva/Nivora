# Repository Workflow Control Plane

Nivora's repository workflow layer is a foundation for repository-agnostic DevOps actions. It is not a GitHub Actions clone and it is not production-ready.

The design target is private and self-hosted environments where code may come from GitLab, Gitea, GitHub Enterprise, generic Git, a local repository path, or future archive/artifact inputs.

## Boundary

```text
Repository metadata
-> RepositorySnapshot
-> RepositoryIntelligence
-> DevOpsPlan
-> Nivora Workflow plan
-> PipelineRun definition conversion
-> Artifact / Release / Deployment planning
-> Audit / Evidence / Timeline
-> MCP read-only and plan-only agent surface
```

The current implementation covers the first part of this flow:

- repository catalog metadata already exists in the catalog usecase
- `SCMProvider` models read-only repository operations
- the generic/local adapter can inspect local repository trees without network access
- repository snapshots and intelligence exist in `internal/usecase/repository`
- repository DevOps plans derive build, test, package, security, deployment, and release-candidate metadata from the latest saved snapshot without mutating runtime state
- repository snapshots, intelligence, and workflow plan records can be persisted through the PostgreSQL runtime store when `database.runtime_store: postgres` is configured
- Nivora Workflow parser and planner exist in `internal/usecase/workflow`
- API, CLI, and MCP expose validate/plan/read-only surfaces

## What Does Not Happen By Default

The repository workflow layer does not:

- clone private remote repositories by default
- resolve CredentialRef values for SCM providers
- push commits
- open pull requests or merge requests
- execute workflow steps through MCP
- deploy, apply, sync, approve, or roll back
- treat GitHub, GitLab, or Gitea as core dependencies

Write and execution capabilities remain future guarded work. They require RBAC, tenant scope, explicit allow flags, confirmation, audit, runner policy, and CredentialRef/SecretRef handling.

## Repository Snapshot Safety

Repository snapshotting is static inspection.

The generic/local provider records file path, size, and hash for ordinary files. Secret-like files such as `.env`, `.npmrc`, kubeconfig-like names, token/password/credential-named files, and private-key-like names are recorded as metadata only. Their contents are not read for hashing.

Snapshot warnings should be treated as operator evidence, not as instructions.

## DevOps Plan Model

Repository DevOps planning reads the latest saved snapshot and recomputes static intelligence into:

- build command candidates
- test command candidates
- package command candidates
- security scan candidates
- deployment target candidates
- release-candidate metadata

The plan is metadata-only. It does not run commands, trigger scanners, create Release or ReleaseArtifact records, apply manifests, sync Argo CD, or deploy hosts.

## Workflow Model

Nivora Workflow is the native authoring format under `.nivora/workflows/*.yaml`.

It supports parser/planner foundation behavior:

- triggers such as `manual` and `push`
- jobs and steps
- `needs` dependency edges
- matrix expansion with limits
- runner labels
- artifacts and cache hints
- unsupported `uses` warnings
- secret-like environment value rejection unless SecretRef/CredentialRef style references are used

The workflow planner can convert compatible definitions into the existing Pipeline definition model. PipelineRun remains the CI runtime object; this layer must not create a second workflow engine.

`/api/v1/workflows/run` is guarded by RBAC plus explicit `confirm=true` and `allowPipelineRun=true`. It records WorkflowRun metadata and creates a queued PipelineRun through the existing runtime. `POST /api/v1/workflows/runs/{id}/cancel` cancels the linked PipelineRun and updates WorkflowRun metadata. `POST /api/v1/workflows/runs/{id}/retry` retries Failed, Canceled, or Timeout WorkflowRuns by converting the stored WorkflowPlan into a new queued PipelineRun, and it requires explicit confirmation. `POST /api/v1/workflows/runs/reconcile` scans non-terminal WorkflowRun records and repairs status drift from linked PipelineRun state. These routes do not execute shell steps directly and are not exposed through MCP action tools.

WorkflowPlan and WorkflowRun records are durable in configured PostgreSQL mode. Raw WorkflowDefinition YAML is still not stored by this plan/run foundation because source document retention needs stronger redaction and tenant-scope review.

## MCP Surface

The MCP surface is read-only and plan-only in this foundation:

- `nivora://repositories`
- `nivora://repositories/{id}`
- `nivora://repositories/{id}/snapshot/latest`
- `nivora://repositories/{id}/intelligence`
- `nivora://repositories/{id}/devops-plan`
- `nivora_repository_inspect`
- `nivora_repository_devops_plan`
- `nivora_workflow_validate`
- `nivora_workflow_plan`

Destructive actions such as deployment apply, Argo CD sync, Git push, rollback execution, token rotation, secret retrieval, and runner registration remain blocked MCP actions.

Remote MCP exposure still requires deeper tenant proof, distributed rate limits, and operator deployment guidance.
