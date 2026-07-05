# Nivora Implementation Progress

This page tracks the current repository/workflow/MCP control-plane work. It is not a release approval document.

Current maturity remains **hardened beta-candidate foundation, not production-ready**.

## Completed In The Current Repository-Agnostic Actions Track

| Area | Status | Evidence | Notes |
|---|---|---|---|
| SCM provider port | foundation | `internal/ports/scm/scm.go` | Read-oriented repository operations are modeled behind a port. Provider writes are intentionally not part of the default path. |
| Generic/local SCM adapter | foundation | `internal/adapters/scm/generic` | Local read-only snapshotting works without external network access. Secret-like files are metadata-only and are not content-hashed. |
| Repository usecase model | foundation | `internal/usecase/repository` | RepositorySnapshot, RepositoryIntelligence, and DevOpsPlan models exist with memory storage for local mode and PostgreSQL storage for configured server/MCP mode. |
| Repository API | foundation | `/api/v1/repositories/{id}/snapshot`, `/snapshots`, `/intelligence`, `/analyze` | Backed by read-only local/generic inspection. Catalog metadata remains separate from snapshot storage, but both can use PostgreSQL when `database.runtime_store: postgres` is configured. |
| Repository CLI | foundation | `nivora repository inspect --path` | Local static inspection prints snapshot and intelligence metadata. |
| Nivora Workflow parser/planner | foundation | `internal/usecase/workflow` | Parser, validator, DAG planning, matrix expansion, unsupported `uses` warnings, and Pipeline definition conversion exist. |
| Workflow API | foundation | `/api/v1/workflows/validate`, `/api/v1/workflows/plan`, `/api/v1/workflows/run` | Validate/plan are plan-only. Run returns structured `not_implemented`. |
| Workflow CLI | foundation | `nivora workflow validate`, `nivora workflow plan` | Local authoring flow exists; it does not execute workflow steps. |
| MCP repository/workflow tools | foundation | `nivora_repository_inspect`, `nivora_workflow_validate`, `nivora_workflow_plan` | Plan-only tools return `mutated=false`; destructive MCP actions remain blocked. |
| Contract coverage | partial | OpenAPI paths, MCP permission matrix, targeted API/MCP/CLI tests | Route/path contract and MCP catalog coverage pass for the new surface. |
| Repository snapshot persistence | foundation | `000017_repository_workflow_persistence`, `internal/adapters/repository/postgres/repository_store.go`, runtime/server/MCP `NewRepositoryServiceWithConfig` wiring | RepositorySnapshot and RepositoryIntelligence survive service restart in optional PostgreSQL integration tests. |

## Still Open

| Area | Gap | Recommended Next Step |
|---|---|---|
| Workflow plan persistence | WorkflowDefinition and WorkflowPlan records are still plan-only/in-memory request outputs. | Add PostgreSQL WorkflowDefinition/WorkflowPlan storage only after the execution mapping to PipelineRun is finalized. |
| Workflow run lifecycle | Workflow execution is not modeled as a durable WorkflowRun; execution still belongs to PipelineRun. | Add WorkflowRun metadata only if it strengthens PipelineRun traceability without creating a second runtime. |
| External SCM providers | GitHub/GitLab/Gitea are not real integrations. | Keep them adapter skeletons until CredentialRef resolution, tenant policy, rate limits, and provider tests are designed. |
| MCP remote exposure | Repository/workflow MCP tools are safe locally but not proven for broad remote use. | Add tenant-scope tests and response caps for repository snapshots before remote exposure. |
| Web console | No repository/workflow pages are added in this track. | Add only after backend contracts settle. |

## Verification Notes

The current track adds focused tests for:

- generic SCM secret-like file handling
- repository snapshot/intelligence API behavior
- workflow validate/plan/run placeholder behavior
- MCP repository/workflow tools
- CLI repository/workflow local commands

Full production readiness still requires broader persistence, recovery, runner sandboxing, live restore drills, and external adapter hardening.
