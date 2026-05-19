# AGENTS.md

Canonical AI coding instructions for Nivora.

Nivora is an open-source DevOps delivery control plane for CI/CD, GitOps, multi-cloud deployment, artifact orchestration, policy gates, runners, release audit, and future visualization APIs.

This repository is backend-first. Do not build frontend code unless explicitly requested.

## 1. Project Priorities

When making changes, optimize in this order:

1. Correct architecture boundaries
2. Simplicity
3. Maintainability
4. Testability
5. Contributor readability
6. Runtime performance

Do not optimize for cleverness, premature abstraction, or speculative future features.

## 2. Current Architecture

Nivora uses a modular monolith architecture with multiple binaries:

- `nivora-server`
- `nivora-worker`
- `nivora-runner`
- `nivora` CLI

The architecture must allow future extraction into separate services, but do not split into microservices unless explicitly requested.

## 3. Hard Architecture Rules

The domain layer must not depend on:

- HTTP frameworks
- database packages
- queue clients
- object storage clients
- Kubernetes clients
- cloud provider SDKs
- Argo CD clients
- GitHub/GitLab clients
- logging frameworks
- telemetry frameworks

Ports define capabilities.
Adapters implement ports.
Use cases orchestrate domain behavior.
Infrastructure code must not leak into domain models.

## 4. Directory Rules

Expected high-level structure:

- `cmd/`: binary entrypoints only
- `internal/app/`: application bootstrapping
- `internal/domain/`: pure domain types and invariants
- `internal/usecase/`: business use cases
- `internal/ports/`: interfaces for external capabilities
- `internal/adapters/`: external system implementations
- `internal/infra/`: technical infrastructure
- `internal/api/`: HTTP/gRPC API layer
- `api/`: OpenAPI, AsyncAPI, proto specs
- `configs/`: example configuration
- `deployments/`: deployment manifests
- `docs/`: architecture and engineering documentation
- `examples/`: sample pipelines and deployment specs
- `scripts/`: local development scripts
- `test/`: integration and e2e tests

Do not place business logic in `cmd/`.
Do not place database logic in `internal/domain/`.
Do not place HTTP handlers in `internal/usecase/`.
Do not place cloud-specific logic in `internal/usecase/`.
Do not create new top-level directories unless the reason is clear.

## 5. Phase Control

Current phase is set by the maintainer's explicit task. If no phase is specified, default to alpha-safe maintenance: documentation, tests, examples, guardrails, and small bug fixes only.

Allowed in foundation and alpha-hardening work:

- repository skeleton
- binaries
- config loading
- logging setup
- health endpoints
- placeholder API routes
- domain structs
- port interfaces
- placeholder adapters
- docs
- migrations
- Dockerfile
- docker-compose
- Makefile
- CI workflow
- minimal tests
- architecture validation and guardrail polish
- public planning, concept, roadmap, community, and RFC documentation

Not allowed unless explicitly requested by the current phase task:

- real AWS integration
- real Aliyun integration
- real Tencent Cloud integration
- production Kubernetes deployment engine
- production Argo CD automation
- real GitHub/GitLab write integration
- real Harbor/Nexus/JFrog integration
- frontend beyond explicitly requested frontend foundation work
- production readiness claims
- complex workflow engine
- multi-service decomposition

## 6. Dependency Rules

Before adding a production dependency:

1. Check whether the standard library is enough.
2. Check whether an existing dependency already solves it.
3. Explain why the dependency is needed.
4. Prefer small, widely used, actively maintained packages.
5. Do not add dependencies for trivial helpers.

Preferred dependencies:

- HTTP router: `go-chi/chi`
- CLI: `spf13/cobra`
- Config: `spf13/viper` or a small internal loader
- Database driver: `pgx`
- Logging: `log/slog`
- Migrations: `goose` or `golang-migrate`
- Testing: standard Go testing package

Avoid heavy frameworks unless explicitly requested.

## 7. Coding Style

Use simple, explicit Go code.

Do:

- keep packages small
- use clear names
- return errors explicitly
- keep interfaces minimal
- write tests for behavior
- prefer composition over inheritance-like structures
- keep public APIs documented when exported

Do not:

- create abstractions for single-use code
- introduce reflection-heavy designs
- hide behavior behind magic registration
- create global mutable state
- add broad helper packages without clear purpose
- use real credentials or realistic-looking secrets
- add generated massive files unless requested

## 8. Error Handling

Errors should be explicit and useful.

Do not silently ignore errors.
Do not panic in normal application flow.
Do not return vague errors like `failed` without context.
Do not leak secrets in errors.

## 9. Logging

Use structured logs.
Prefer `slog`.
Do not log secrets, tokens, passwords, private keys, kubeconfigs, cloud credentials, or full authorization headers.

## 10. Security Rules

Never commit secrets.

Never include:

- real tokens
- real private keys
- real cloud credentials
- real kubeconfigs
- real host passwords
- realistic fake credentials that could be mistaken for real ones

Secret values must not be returned by normal APIs.
Secret usage may be audited, but secret values must not be logged.

## 11. Testing Rules

For code changes, add or update tests when the behavior changes.

Minimum checks before considering a task done:

```bash
gofmt -w .
go test ./...
go build ./cmd/nivora-server
go build ./cmd/nivora-worker
go build ./cmd/nivora-runner
go build ./cmd/nivora
```

If a command cannot be run, state why.

## 12. API Rules

HTTP responses should be JSON.
Unimplemented endpoints should return a clear `not implemented` response.
Do not create undocumented public API behavior.
Keep OpenAPI in sync when API behavior changes.

## 13. Migration Rules

Database migrations must be reversible when practical.

For each migration:

- create an up migration
- create a down migration
- avoid destructive changes unless explicitly requested
- do not over-index early
- keep schemas readable

## 14. Documentation Rules

When changing architecture, update docs.

Relevant docs:

- `docs/architecture/architecture-contract.md`
- `docs/architecture/module-boundaries.md`
- `docs/architecture/executor-design.md`
- `docs/architecture/runner-design.md`
- `docs/engineering/dependency-policy.md`
- `docs/engineering/testing-policy.md`

Do not let docs claim features are production-ready before they exist.

## 15. Change Discipline

Make surgical changes.

Do not:

- rewrite unrelated files
- reformat unrelated code
- rename packages without need
- change public interfaces casually
- refactor adjacent code just because it looks imperfect
- continue into the next phase without explicit instruction

Every changed line should be traceable to the requested task.

## 16. When Ambiguous

If the task is ambiguous, do not guess silently.

For interactive work:

- ask a concise clarification question.

For autonomous goal-based work:

- state the assumption
- choose the smallest reversible implementation
- document the limitation in the final summary

## 17. Final Response Expectations

At the end of a task, summarize:

- what changed
- what commands were run
- whether checks passed
- known limitations
- recommended next step

Do not claim that unimplemented integrations are complete.
