# Enterprise Next Goals

Current maturity: **hardened beta-candidate foundation, not production-ready**.

These are large goals for future work. They are ordered by production-candidate impact, not by roadmap phase number.

## 1. Runtime Recovery Closure

Why now: runtime durability is the core production-candidate blocker.

Scope:
- multiprocess worker/runner recovery with PostgreSQL;
- stale PipelineRun, DeploymentRun, ReleaseExecution, and JobRun reconciliation;
- event outbox retry and idempotency;
- cancellation and timeout reconciliation.

Non-goals:
- Temporal/Tekton/Argo Workflows;
- cloud provider integrations.

Acceptance:
- Postgres integration and multiprocess smoke prove restart recovery;
- recovery docs list tested and untested paths;
- `make verify-runtime-recovery` is a release gate.

## 2. Enterprise Security Closure

Why now: security boundaries must be mechanical, not aspirational.

Scope:
- hard route/RBAC assertions for critical routes;
- tenant isolation fixtures;
- secret/token response scanner;
- production unsafe config tests;
- audit of denied sensitive actions.

Non-goals:
- full Keycloak product integration;
- frontend login UI.

Acceptance:
- route permission matrix, tests, and OpenAPI security schemes align;
- runner token cannot access non-runner surfaces;
- tenant cross-access tests fail closed.

## 3. Runner Sandbox and Fleet Hardening

Why now: runner execution is the highest-risk runtime boundary.

Scope:
- default isolated runner guidance;
- token rotation/revocation e2e;
- label/capability/concurrency claim tests;
- offline detection and reclaim proof;
- shell executor safety limits.

Non-goals:
- autoscaling;
- privileged execution by default.

Acceptance:
- runner fleet tests cover token misuse and unrelated jobs;
- docs state shell is not a sandbox;
- production profile disables unsafe executor options.

## 4. MCP Remote Read-only Readiness

Why now: Nivora is a strong AI control-plane candidate, but remote exposure is risky.

Scope:
- remote auth contract design;
- service-account bearer model;
- tenant scope filters;
- rate limit, timeout, response cap, and pagination;
- remote audit metadata and denial audit.

Non-goals:
- action MCP;
- secret retrieval;
- apply/sync/rollback/approve actions.

Acceptance:
- remote MCP remains disabled by default;
- contract tests prove runner tokens and cross-tenant reads are denied;
- go/no-go doc moves from no-go to conditional only with evidence.

## 5. Tenant Isolation and Quota Hardening

Why now: enterprise customers need predictable org/project/environment boundaries.

Scope:
- resource ownership checks for APIs and MCP resources;
- quota enforcement for concurrent runs, runners, logs, and artifacts;
- rate limits for API tokens and runner protocol.

Non-goals:
- billing;
- marketplace tenancy.

Acceptance:
- cross-project access tests cover read and mutation paths;
- quota violations return structured errors and audit entries.

## 6. Audit Evidence and Compliance Closure

Why now: audit is a core control-plane value.

Scope:
- unified audit hash-chain coverage;
- evidence bundle generation and persistence;
- approval, policy, artifact, deployment, and release evidence links;
- JSON and markdown export.

Non-goals:
- external GRC integrations.

Acceptance:
- Postgres audit-chain verification covers critical runtime and governance paths;
- evidence exports contain no secrets;
- retention policies are tested.

## 7. Production Install and DR Drill

Why now: production-candidate needs repeatable install and recovery.

Scope:
- Helm production profile smoke;
- Compose production-like profile smoke;
- migration job validation;
- backup and restore drill;
- documented skip reasons for optional tools.

Non-goals:
- Kubernetes operator;
- cloud-managed database automation.

Acceptance:
- install smoke scripts pass or skip clearly;
- restore drill is documented and automated where practical.

## 8. API Contract Stabilization

Why now: broad API surface needs compatibility discipline.

Scope:
- route/OpenAPI coverage;
- security scheme checks;
- placeholder behavior;
- error response consistency;
- AsyncAPI event accuracy.

Non-goals:
- new API groups.

Acceptance:
- contract tests are hard gates;
- docs distinguish implemented, placeholder, and future routes.

## 9. Observability and SLO Closure

Why now: operators need system behavior they can observe.

Scope:
- metrics for run counts, failures, duration, queue time, runner heartbeat, claim latency, policy denials;
- alert examples;
- diagnostics dependency checks;
- runbook smoke tests.

Non-goals:
- hosted dashboards;
- mandatory Prometheus deployment.

Acceptance:
- metrics registration tests pass;
- runbook scripts cover stuck run, failed deployment, offline runner, DB unavailable, and outbox stuck.

## 10. Performance and Load Readiness

Why now: capacity limits should be measured before broad beta.

Scope:
- benchmarks for run creation, log append, timeline query, heartbeat, claim, inventory;
- load scripts for runs, logs, runners;
- pagination and request-size limits;
- index review.

Non-goals:
- large infrastructure requirements;
- speculative optimization without measurements.

Acceptance:
- benchmark suite runs locally;
- load scripts are deterministic and documented;
- high-volume APIs enforce limits.
