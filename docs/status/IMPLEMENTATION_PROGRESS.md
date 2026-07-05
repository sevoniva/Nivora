# Nivora Implementation Progress

This page tracks the current repository/workflow/MCP control-plane work. It is not a release approval document.

Current maturity remains **hardened beta-candidate foundation, not production-ready**.

## Completed In The Current Repository-Agnostic Actions Track

| Area | Status | Evidence | Notes |
|---|---|---|---|
| SCM provider port | foundation | `internal/ports/scm/scm.go` | Read-oriented repository operations are modeled behind a port. Provider writes are intentionally not part of the default path. |
| Generic/local SCM adapter | foundation | `internal/adapters/scm/generic` | Local read-only snapshotting works without external network access. Secret-like files are metadata-only and are not content-hashed. |
| Repository usecase model | foundation | `internal/usecase/repository` | RepositorySnapshot, RepositoryIntelligence, and DevOpsPlan models exist with memory storage for local mode and PostgreSQL storage for configured server/MCP mode. |
| Repository API | foundation | `/api/v1/repositories/{id}/snapshot`, `/snapshots`, `/intelligence`, `/analyze`, `/api/v1/devops/plan` | Backed by read-only local/generic inspection. Catalog metadata remains separate from snapshot storage, but both can use PostgreSQL when `database.runtime_store: postgres` is configured. DevOps plan output is metadata-only and does not execute commands, create releases, trigger scans, or deploy. |
| Repository CLI | foundation | `nivora repository inspect --path`, `nivora repository devops-plan <repository-id>` | Local static inspection prints snapshot and intelligence metadata. Server-backed DevOps planning reads the latest saved snapshot and returns build/test/package/security/deployment/release-candidate metadata with `mutated=false`. |
| Nivora Workflow parser/planner | foundation | `internal/usecase/workflow` | Parser, validator, DAG planning, matrix expansion, unsupported `uses` warnings, Pipeline definition conversion, and stored plan-record metadata exist. |
| Workflow API | foundation | `/api/v1/workflows`, `/api/v1/workflows/validate`, `/api/v1/workflows/plan`, `/api/v1/workflows/{id}/plan`, `/api/v1/workflows/plans`, `/api/v1/workflows/plans/{id}`, `/api/v1/workflows/runs`, `/api/v1/workflows/runs/{id}`, `/api/v1/workflows/runs/{id}/cancel`, `/api/v1/workflows/runs/{id}/retry`, `/api/v1/workflows/runs/reconcile`, `/api/v1/workflows/run` | Validate is parser-only. Plan stores redacted plan records. Workflow list/latest-plan reads are derived from stored plans. Run requires explicit confirmation and creates a WorkflowRun record plus queued PipelineRun. WorkflowRun read APIs synchronize status from the linked PipelineRun when available. Cancel requests cancel the linked PipelineRun and update WorkflowRun metadata without rollback/delete behavior. Retry requests require explicit confirmation and create a new queued PipelineRun from the stored WorkflowPlan for Failed/Canceled/Timeout WorkflowRuns. Reconcile requests scan non-terminal WorkflowRun metadata and repair status drift from linked PipelineRun state. |
| Workflow CLI | foundation | `nivora workflow validate`, `nivora workflow plan`, `nivora workflow run`, `nivora workflow cancel`, `nivora workflow retry`, `nivora workflow reconcile` | Local authoring flow exists; server-backed run/cancel/retry/reconcile paths queue, cancel, retry, or repair linked WorkflowRun/PipelineRun status metadata and do not execute workflow steps in the CLI process. |
| MCP repository/workflow tools | foundation | `nivora_repository_inspect`, `nivora_repository_devops_plan`, `nivora_workflow_validate`, `nivora_workflow_plan`, `nivora://repositories/{id}/devops-plan` | Plan-only tools and resources return `mutated=false`; destructive MCP actions remain blocked. |
| Contract coverage | partial | OpenAPI paths, MCP permission matrix, targeted API/MCP/CLI tests | Route/path contract and MCP catalog coverage pass for the new surface. |
| Repository snapshot persistence | foundation | `000017_repository_workflow_persistence`, `internal/adapters/repository/postgres/repository_store.go`, runtime/server/MCP `NewRepositoryServiceWithConfig` wiring | RepositorySnapshot and RepositoryIntelligence survive service restart in optional PostgreSQL integration tests. |
| Workflow plan/run persistence | foundation | `000018_workflow_plan_persistence`, `000019_workflow_run_persistence`, `internal/adapters/repository/postgres/workflow_store.go`, runtime/server/MCP `NewWorkflowServiceWithConfig` wiring | WorkflowPlan and WorkflowRun records survive service restart in optional PostgreSQL integration tests. Raw workflow YAML is not stored by the plan/run store. |

## Still Open

| Area | Gap | Recommended Next Step |
|---|---|---|
| Workflow definition catalog | WorkflowDefinition source documents are not stored as raw YAML in the workflow plan store. | Add a source-definition catalog only after redaction, tenant scope, and PipelineRun conversion semantics are finalized. |
| Workflow run lifecycle | WorkflowRun metadata exists, queues PipelineRun records, read APIs synchronize WorkflowRun status from linked PipelineRun state, cancel requests cancel linked PipelineRun records, guarded retry queues a replacement PipelineRun for Failed/Canceled/Timeout runs, and manual reconcile repairs non-terminal WorkflowRun status drift. | Add background/event-driven reconciliation, retry policy controls, and richer aggregate workflow events before calling it beta-grade. |
| External SCM providers | GitHub/GitLab/Gitea are not real integrations. | Keep them adapter skeletons until CredentialRef resolution, tenant policy, rate limits, and provider tests are designed. |
| MCP remote exposure | Repository/workflow MCP tools are safe locally but not proven for broad remote use. | Add tenant-scope tests and response caps for repository snapshots before remote exposure. |
| Web console | No repository/workflow pages are added in this track. | Add only after backend contracts settle. |

## Verification Notes

The current track adds focused tests for:

- generic SCM secret-like file handling
- repository snapshot/intelligence API behavior
- workflow validate/plan/guarded-run/cancel/retry/reconcile behavior
- MCP repository/workflow tools
- CLI repository/workflow local commands

Full production readiness still requires broader persistence, recovery, runner sandboxing, live restore drills, and external adapter hardening.
