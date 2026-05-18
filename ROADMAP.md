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

## Phase 2: GitOps and Production Release

- Add release and deployment workflows.
- Implement GitOps deployment mode.
- Add approval gates and release audit trails.
- Add deployment diff, health verification, and rollback foundation.

## Phase 3: Multi-Cloud and DevSecOps

- Add cloud provider adapters.
- Add artifact scanning and policy evaluation.
- Add secret backends and notification providers.
- Harden authn, authz, audit, and telemetry.

## Phase 4: Visualization Frontend

- Build visualization APIs and frontend surfaces.
- Add deployment topology, pipeline timelines, audit exploration, and operations dashboards.
