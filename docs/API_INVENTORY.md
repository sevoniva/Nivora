# API Inventory

Phase 9.0 beta freeze inventory. This document summarizes the public HTTP API surface and whether each group is implemented, partial, or placeholder. OpenAPI remains the schema source of truth at `api/openapi/openapi.yaml`, and route/path coverage is checked by `internal/api/http/routes/openapi_contract_test.go`.

## Implemented Foundation

| Group | Representative Routes | Notes |
|---|---|---|
| Health / readiness / version | `GET /healthz`, `GET /readyz`, `GET /api/v1/version` | local operational checks |
| System diagnostics | `GET /api/v1/system/info`, `/runtime`, `/diagnostics`, `/runtime/recovery`, `POST /runtime/reconcile` | diagnostics and recovery summaries |
| Metrics | `GET /metrics` | process-local Prometheus text format |
| PipelineRun | `POST /api/v1/pipeline-runs`, `GET /api/v1/pipeline-runs`, `GET /api/v1/pipeline-runs/{id}`, logs/events/timeline/cancel | shell runtime foundation, optional pagination on list/log/event/timeline reads |
| Runner protocol | register, heartbeat, claim, append logs, update status, offline detect, token rotate/revoke | runner mutation uses runner tokens or RBAC where applicable |
| DeploymentRun | `POST /api/v1/deployments`, plan/apply, get, resources, health, diff, snapshot, rollback plan, logs/events/timeline/cancel/resume | apply and rollback remain guarded |
| Release orchestration | releases, release artifacts, plan/deploy, executions, targets, timeline, cancel/resume, release evidence | sequential local orchestration foundation |
| Release target catalog | `GET/POST /api/v1/release-targets`, get/update/disable/validate | metadata only; unsafe flags default false |
| Artifact / release binding | inspect, resolve, release create/list/get/artifacts, registry validate | OCI-compatible foundation; vendor management APIs are not implemented |
| Security / policy | scans, findings, stored scan/finding queries, policy catalog, policy attachments, policy evaluate, release/deployment security | noop/fake scanners and built-in policy rules; no external policy distribution |
| Auth / RBAC | whoami, permissions, token info, users, roles, permissions, memberships, service accounts, API tokens | local/token/OIDC-foundation only |
| Secrets / credentials | secrets, secret refs, provider validate, rotate/delete, credentials CRUD/validate | values are not returned by normal APIs |
| Approval / change windows / notifications | approvals, change-window evaluate, notifications test/list | backend governance foundation |
| Cloud inventory | providers, accounts, validate, regions, clusters, hosts, registries, inventory | fake/provider skeleton inventory only |
| Host deployment | host groups, host deployment plan, deployment hosts, rollback plan | dry-run/noop and guarded SSH surface |
| Compliance | audit search, evidence bundle, retention policy | retention enforcement jobs remain future work |
| Plugins | list, inspect, capabilities, validate | built-in registry and manifest validation |
| Visualization | `/api/v1/visualization` index, pipeline/deployment/release visualization, environment topology, runner/security/audit summaries | backend read models for future UI |
| Tenancy | quota, usage | scope and quota foundation |

## Non-HTTP Control-Plane Surfaces

| Surface | Entry Point | Notes |
|---|---|---|
| MCP stdio foundation | `cmd/nivora-mcp`, `nivora mcp serve --stdio` | Local read-only and plan-only MCP resources/tools/prompts over stdio JSON-RPC. It is not an HTTP API, records compliance-backed audit, rejects runner tokens, and does not expose action tools. |

## Partial Or Guarded

| Group | Routes | Reason |
|---|---|---|
| Kubernetes apply / rollback | `POST /api/v1/deployments/apply`, `POST /api/v1/deployments/{id}/rollback` | explicit confirmation required; no default destructive behavior |
| Argo CD sync | integration and deployment sync routes | sync requires explicit allow and confirmation; production automation is future work |
| GitOps commit / rollback | `POST /api/v1/deployments/gitops/commit`, `/rollback` | local working tree foundation; push is guarded |
| External providers | cloud, registry, secret, notification, scanner routes | adapters are skeletal or fake unless explicitly configured |
| Pagination | selected list/log/event/timeline/audit routes | optional `limit`/`offset`; legacy array responses remain when omitted |

## Placeholder / Not Implemented

There are currently no root-level HTTP route groups registered solely to return structured `not_implemented`. Several capabilities remain foundation-only, skeleton, noop, fake, or experimental even when their read/write metadata APIs are implemented. If a placeholder route group is reintroduced, it must return structured `not_implemented` and be labeled in OpenAPI.

## API Freeze Notes

- New routes during beta freeze require an explicit rationale and OpenAPI updates.
- Unimplemented routes must keep structured `not_implemented` responses.
- Existing response compatibility is preferred; pagination is opt-in to avoid breaking legacy array clients.
- No route should return secrets, token hashes, kubeconfigs, private keys, or realistic credentials.
- API behavior remains beta-level and not GA stable.
