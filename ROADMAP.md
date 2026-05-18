# Roadmap

This is the concise project roadmap. Detailed phase docs live in [docs/roadmap/overview.md](docs/roadmap/overview.md).

## Phase 0: Skeleton

- Establish Go module structure.
- Define control plane, worker, runner, and CLI binaries.
- Add domain models, ports, placeholder adapters, API specs, migrations, docs, CI, and local development setup.

## Phase 0.5: Guardrails and Validation

- Harden AI and contributor guardrails.
- Keep GOPROXY configurable for global contributors and China-based development.
- Enforce formatting, tidy, vet, tests, builds, architecture checks, and secret checks in CI.
- Polish local development verification scripts and Makefile targets.
- Keep placeholder APIs honest and structured without adding business logic.

## Phase 0.6: Planning and Collaboration Docs

- Add public planning docs, project charter, product vision, architecture blueprint, concept docs, roadmap docs, community docs, and RFC template.
- Keep current implementation clearly separated from target architecture.
- Improve contribution paths without adding Phase 1 business logic.

## Phase 1: Minimal Pipeline Execution

- Parse minimal Pipeline definitions and create PipelineRuns.
- Execute controlled shell steps.
- Capture logs, events, audit records, and run status.

## Phase 1.5: Durable Runtime Foundation

- Add explicit status transition helpers for PipelineRun, StageRun, JobRun, and StepRun.
- Add in-memory runtime repositories, ordered LogChunks, timeline APIs, minimal cancellation, retry, timeout, runner selection, and runner heartbeat.
- Keep runtime shell-only and avoid Phase 2 deployment integrations.

## Phase 1.6: Runtime Acceptance and Developer Experience

- Add runtime acceptance docs, smoke scripts, safer examples, CLI/API polish, and developer troubleshooting docs.
- Keep all tests self-contained without Kubernetes, Argo CD, cloud, Git provider, or registry dependencies.

## Phase 2: Release and Deployment Foundation

- Add release and deployment workflows.
- Phase 2.0 adds YAML deployment planning and non-destructive dry-run foundation.
- Phase 2.1 adds controlled Kubernetes YAML dry-run/apply runtime foundation with explicit local apply, resource inventory, rollout result modeling, and rollback baseline.
- Future Phase 2 work adds GitOps deployment mode, approval gates, release audit trails, live deployment diff, production health verification, and rollback execution.
- Keep production Kubernetes apply semantics, Helm, Kustomize, and Argo CD implementation out of Phase 2.1.

## Phase 3: Multi-Cloud and DevSecOps

- Add cloud provider adapters.
- Add artifact scanning and policy evaluation.
- Add secret backends and notification providers.
- Harden authn, authz, audit, and telemetry.

## Phase 4: Visualization Frontend

- Build visualization APIs and frontend surfaces.
- Add deployment topology, pipeline timelines, audit exploration, and operations dashboards.
