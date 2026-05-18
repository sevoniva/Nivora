# Roadmap

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

## Phase 1: Minimal Pipeline Execution

- Persist projects, repositories, pipelines, and pipeline runs.
- Schedule simple jobs to runners.
- Execute controlled shell steps.
- Stream basic logs and record run status.

## Phase 2: GitOps and Production Release

- Add release and deployment workflows.
- Implement GitOps deployment mode.
- Add approval gates and release audit trails.
- Strengthen runner registration and heartbeat behavior.

## Phase 3: Multi-Cloud and DevSecOps

- Add cloud provider adapters.
- Add artifact scanning and policy evaluation.
- Add secret backends and notification providers.
- Harden authn, authz, audit, and telemetry.

## Phase 4: Visualization Frontend

- Build visualization APIs and frontend surfaces.
- Add deployment topology, pipeline timelines, audit exploration, and operations dashboards.
