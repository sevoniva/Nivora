# API Inventory

Phase 9.0 beta freeze inventory. This document summarizes the public HTTP API surface and whether each group is implemented, partial, or placeholder. OpenAPI remains the schema source of truth at `api/openapi/openapi.yaml`, and route/path coverage is checked by `internal/api/http/routes/openapi_contract_test.go`.

## Implemented Foundation

| Group | Representative Routes | Notes |
|---|---|---|
| Health / readiness / version | `GET /healthz`, `GET /readyz`, `GET /api/v1/version` | local operational checks |
| System diagnostics | `GET /api/v1/system/info`, `/runtime`, `/diagnostics`, `/runtime/recovery`, `POST /runtime/reconcile` | diagnostics and recovery summaries |
| Metrics | `GET /metrics` | process-local Prometheus text format |
| PipelineRun / Pipeline definitions | `POST /api/v1/pipeline-runs`, `GET /api/v1/pipeline-runs`, `GET /api/v1/pipeline-runs/{id}`, logs/events/timeline/cancel, `GET/POST/PATCH/DELETE /api/v1/pipelines`, `/api/v1/pipelines/{id}/versions`, `/api/v1/pipelines/{id}/runs` | shell runtime foundation, optional pagination on list/log/event/timeline reads; Pipeline definitions can be cataloged, versioned, and run from the current or a saved historical version |
| Nivora Workflow planning and guarded run | `GET /api/v1/workflows`, `POST /api/v1/workflows/validate`, `POST /api/v1/workflows/plan`, `GET /api/v1/workflows/{id}/plan`, `GET /api/v1/workflows/plans`, `GET /api/v1/workflows/plans/{id}`, `GET /api/v1/workflows/runs`, `GET /api/v1/workflows/runs/{id}`, `POST /api/v1/workflows/run` | parser/validator/DAG planner foundation; workflow summaries and latest-plan reads are derived from stored plan records. Plan records include content hashes and redacted plan output and can be persisted in configured PostgreSQL runtime mode. Guarded run requires `confirm=true` and `allowPipelineRun=true`, records WorkflowRun metadata, and queues a PipelineRun through the existing runtime. Raw workflow YAML is not persisted by this foundation |
| Runner protocol | runner groups, register, heartbeat, claim, append logs, update status, offline detect, token rotate/revoke | runner mutation uses runner tokens or RBAC where applicable; runner groups provide project/environment/executor/concurrency claim guardrails |
| DeploymentRun | `POST /api/v1/deployments`, plan/apply, get, resources, health, diff, snapshot, rollback plan, logs/events/timeline/cancel/resume | apply and rollback remain guarded; GitOps plans can resolve `target.repositoryId` from the repository catalog without contacting SCM providers |
| Release orchestration | releases, release artifacts, release inventory filters, release cancel, plan/deploy, executions, targets, timeline, cancel/resume, release evidence | sequential local orchestration foundation plus server-backed Release ID plan/deploy for safe noop/webhook targets; Release inventory can be filtered by project, environment, application, and status; Release status now follows plan/deploy/approval/cancel outcomes and emits `devops.release.status.updated`; release cancel marks the Release, cancels non-terminal ReleaseExecutions, and asks linked non-terminal DeploymentRuns to cancel; it does not run rollback or delete resources |
| Release target catalog | `GET/POST /api/v1/release-targets`, get/update/disable/validate | metadata only; unsafe flags default false |
| SCM repository catalog | `GET/POST /api/v1/repositories`, get/update/disable/validate | metadata-only repository inventory; validation and GitOps plan metadata resolution do not contact SCM providers or resolve CredentialRef values |
| Artifact / release binding | inspect, resolve, artifact create/list/get/release bindings, release create/list/get/artifacts, inline registry validate, saved registry validate, explicit saved registry repository artifact listing | OCI-compatible foundation; known artifact references can be tracked as standalone catalog records, release-bound artifacts are also indexed, server-backed Release creation can attach explicit project ownership metadata, direct and saved registry metadata validation share endpoint safety rules, inline endpoint credentials are rejected in favor of CredentialRef metadata, explicit OCI repository tags can be listed through saved registry metadata, and vendor management APIs or registry crawling are not implemented |
| Security / policy | scans, findings, stored scan/finding list/detail queries, policy catalog, policy attachments, policy evaluate, stored policy result list/detail queries, release/deployment security | noop/fake scanners and built-in policy rules; policy result catalog is scoped and persisted when configured for PostgreSQL; no external policy distribution |
| Auth / RBAC | whoami, permissions, token info, users, roles, permissions, memberships, service accounts, API tokens | local/token/OIDC-foundation only |
| Secrets / credentials | secrets, secret refs, provider validate, rotate/delete, credentials CRUD/validate | values are not returned by normal APIs |
| Approval / change windows / notifications | approvals, approval subject resume, change-window evaluate, notifications test/list | backend governance foundation; `/api/v1/approvals/{id}/resume-subject` applies terminal approval decisions to waiting DeploymentRun, ReleaseExecution, or Paused PipelineRun subjects |
| Cloud inventory | providers, account metadata create/list/get/validate, regions, clusters, hosts, registries, inventory | fake/provider skeleton inventory only; no cloud deployment |
| Host deployment | host groups, host deployment plan, server-backed dry-run/noop via `POST /api/v1/deployments`, deployment hosts, rollback plan | dry-run/noop and guarded SSH surface; server-backed CLI host run rejects remote apply inputs |
| Compliance | audit search, filtered audit-log reads, evidence bundle list/read/export, retention policy get/set/run | evidence bundles include redacted subject summaries, release execution/deployment references, policy/security/approval references, events/audits/log references, and deterministic digests; retention runs can preview candidates and, with explicit confirmation, delete expired evidence bundles only. Audit remains immutable; log/event cleanup jobs remain future work |
| Plugins | list, inspect, capabilities, validate | built-in registry and manifest validation |
| MCP remote JSON-RPC | `POST /api/v1/mcp/rpc` | experimental opt-in read-only/plan-only MCP endpoint; disabled unless configured, bearer/service-account/OIDC auth required, runner tokens and action tools rejected |
| Visualization | `/api/v1/visualization` index, pipeline/deployment/release visualization, environment topology, runner/security/audit summaries | backend read models for future UI |
| Tenancy | quota, usage | scope and quota foundation |

## Non-HTTP Control-Plane Surfaces

| Surface | Entry Point | Notes |
|---|---|---|
| MCP stdio foundation | `cmd/nivora-mcp`, `nivora mcp serve --stdio` | Local read-only and plan-only MCP resources/tools/prompts over stdio JSON-RPC, including runtime recovery, aggregate event/log search, catalog summary, repository catalog/snapshot/intelligence, stored workflow plan records, pipeline definition, deployment, release, release-bound artifact inventory, security finding, policy result, audit/evidence, and capability resources. It records compliance-backed audit, rejects runner tokens, and does not expose action tools. |

## Partial Or Guarded

| Group | Routes | Reason |
|---|---|---|
| Kubernetes apply / rollback | `POST /api/v1/deployments/apply`, `POST /api/v1/deployments/{id}/rollback` | explicit confirmation required; no default destructive behavior |
| Argo CD sync | integration and deployment sync routes | sync requires explicit allow and confirmation; production automation is future work |
| GitOps commit / rollback | `POST /api/v1/deployments/gitops/commit`, `/rollback` | local working tree foundation; push is guarded |
| Workflow execution | `POST /api/v1/workflows/run` | guarded queue-only foundation; shell execution still happens only through PipelineRun/Runner/Worker paths |
| External providers | cloud, registry, secret, notification, scanner routes | adapters are skeletal or fake unless explicitly configured |
| Pagination and filters | selected list/log/event/timeline/audit routes | optional `limit`/`offset`; aggregate events/logs/timeline/audit support lightweight run, subject, scope, and content filters |

## Placeholder / Not Implemented

Several capabilities remain foundation-only, skeleton, noop, fake, or experimental even when their read/write metadata APIs are implemented. Any new placeholder route must return structured `not_implemented` and be labeled in OpenAPI.

## API Freeze Notes

- New routes during beta freeze require an explicit rationale and OpenAPI updates.
- Unimplemented routes must keep structured `not_implemented` responses.
- Existing response compatibility is preferred; pagination is opt-in to avoid breaking legacy array clients.
- No route should return secrets, token hashes, kubeconfigs, private keys, or realistic credentials.
- API behavior remains beta-level and not GA stable.
