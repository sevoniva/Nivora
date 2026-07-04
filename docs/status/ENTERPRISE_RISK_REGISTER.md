# Enterprise Risk Register

Current maturity: **hardened beta-candidate foundation, not production-ready**.

| Priority | Risk | Impact | Evidence | Mitigation | Owner Area | Suggested Goal | Effort |
|---|---|---|---|---|---|---|---|
| P0 | Runner shell jobs are not OS-sandboxed | Host compromise or data exposure | `docs/security/runner-trust-boundary.md` | Default isolated runner profile and operator hardening checklist | Runner | Runner Sandbox and Fleet Hardening | large |
| P0 | Remote MCP lacks auth/scope/limits | Cross-tenant metadata exposure | `docs/status/REMOTE_MCP_READINESS_AUDIT.md` | Keep remote disabled; add remote contract tests | MCP | MCP Remote Read-only Readiness | large |
| P0 | Tenant isolation is not proven for every read model | Project data leakage | tenant tests and MCP tenant scope review | Add ownership tests for API and MCP resources | Security | Tenant Isolation and Quota Hardening | large |
| P0 | Live restore drill is not a release gate | Recovery failure during incident | backup docs and optional scripts | Automate restore drill in integration profile | Ops | Production Install and DR Drill | large |
| P0 | External adapters remain foundation-level | Misleading production use | capability status | Keep labels explicit; harden adapter by adapter | Product | Adapter Production Boundary Closure | large |
| P1 | Audit hash-chain proof is not uniform | Weak compliance evidence | compliance store tests | Add end-to-end audit-chain verification | Compliance | Audit Evidence and Compliance Closure | medium |
| P1 | Upgrade compatibility is not proven | Failed customer upgrades | migration up/down tests only | Add release-to-release migration tests | Database | Migration Compatibility Closure | large |
| P1 | Runner token lifecycle needs operational proof | Stale runner credentials | runner token tests | Add rotate/revoke/claim e2e | Runner | Secure Runner Token Lifecycle | medium |
| P1 | Secret provider lifecycle is incomplete | Secret retrieval/rotation gaps | provider skeletons | Add provider validation contracts | Security | Secret Provider Production Contracts | large |
| P1 | OIDC lifecycle is incomplete | Enterprise auth gaps | auth foundation docs | Add provider integration and token validation tests | Security | Enterprise Auth Closure | large |
| P1 | SLOs are not enforced | Poor production operations | observability docs | Add metrics/alert tests and runbook checks | SRE | Observability and SLO Closure | medium |
| P1 | Performance limits are not characterized | Capacity planning risk | benchmarks/load scripts limited | Add load baselines for logs/runners/API | Performance | Performance and Load Readiness | medium |
| P1 | Kubernetes apply is guarded but not production CD | Unsafe cluster changes if misused | guarded apply docs/tests | Add namespace/context policy and live optional smoke | Deployment | Kubernetes CD Safety Closure | large |
| P1 | Argo sync remains foundation-level | Unsafe GitOps automation if overused | guarded sync docs/tests | Keep disabled; harden status/read-only path first | GitOps | GitOps Safety Closure | medium |
| P1 | Host deploy uses remote shell risk | Host disruption | host docs/tests | Fake SSH batch/rollback tests and isolation docs | Host | Host Deployment Safety Closure | medium |
| P1 | Audit search needs scope proof | Compliance data over-read | audit routes and tests | Scope audit search by tenant/project | Compliance | Audit Scope Enforcement | medium |
| P1 | Response redaction can drift | Secret leakage | redaction tests | Central sensitive-key corpus gate | Security | Redaction Drift Gate | small |
| P1 | API contract is broad | Client compatibility risk | OpenAPI route tests | Freeze semantic versioning and error contracts | API | API Contract Stabilization | medium |
| P1 | Web UI is experimental | User confusion | web docs/build | Keep experimental label and no production claims | Product | Web Console Boundary Cleanup | small |
| P1 | Backup docs are not proof | False recovery confidence | docs and optional scripts | Make restore drill mandatory for release candidate | Ops | DR Drill Release Gate | medium |
| P2 | Stage/step recovery is less proven than run recovery | Debug gaps after restart | pipeline recovery tests | Add stage/step stale tests | Runtime | Runtime Recovery Closure | medium |
| P2 | Event outbox publisher behavior is simple | Missed or duplicated events | outbox recovery tests | Add retry/backoff/idempotency e2e | Runtime | Event Outbox Closure | medium |
| P2 | Cancellation is best-effort | Long-running jobs may continue | runner docs | Add cancel observe tests per executor | Runtime | Cancellation Reconcile Closure | medium |
| P2 | Timeout behavior needs matrix coverage | Stuck runs | timeout tests/docs | Add Pipeline/Deployment/Release timeout matrix | Runtime | Timeout Reconcile Closure | medium |
| P2 | Approval resume flow is incomplete | Governance workflow gaps | approval foundation | Add approval resume/reject e2e | Governance | Approval Lifecycle Closure | medium |
| P2 | Change windows are simple | Scheduling policy gaps | change-window tests | Add timezone/boundary/holiday coverage | Governance | Change Window Hardening | small |
| P2 | Notifications are noop/log/placeholder | Operators may expect real delivery | notification docs | Keep labels; add webhook contract if scoped later | Governance | Notification Boundary Closure | small |
| P2 | OCI registry resolution is optional | Mutable release risk | artifact tests/docs | Enforce requireDigest in release flows where configured | Artifact | Artifact Immutability Closure | medium |
| P2 | Harbor is OCI-compatible only | Product integration confusion | artifact docs | Keep Harbor admin API out of scope | Artifact | Registry Documentation Cleanup | small |
| P2 | Cloud inventory is skeleton/fake | Misread as cloud platform | cloud docs | Keep placeholder labels | Cloud | Cloud Provider Boundary Cleanup | small |
| P2 | Plugin protocol is skeleton | Adapter contributor risk | plugin docs/proto | Add compatibility checks | Plugin | Plugin API Stabilization | medium |
| P2 | Metrics coverage incomplete | Blind spots | metrics docs/tests | Add queue/claim/policy metrics | Observability | Metrics Coverage Closure | medium |
| P2 | Trace propagation incomplete | Hard distributed debugging | tracing placeholder | Add OTel propagation proof | Observability | Tracing Foundation Closure | medium |
| P2 | Runbook checks partial | Operator drift | runbook scripts | Add checks for DB/outbox/runner/deployment incidents | SRE | Runbook Validation | small |
| P2 | Helm live install not mandatory | Install drift | helm template checks | Optional kind install smoke | Packaging | Helm Live Smoke | medium |
| P2 | Compose production profile is example-only | Operator confusion | compose smoke | Stronger docs and live compose smoke | Packaging | Compose Profile Closure | small |
| P2 | Config examples can drift | Unsafe operator copy/paste | config docs/tests | Add config example validation | SRE | Config Example Gate | small |
| P2 | AsyncAPI may document future events | Event contract ambiguity | AsyncAPI test logs | Mark future/reserved channels clearly | API | Event Contract Stabilization | small |
| P2 | Golden MCP answers can become generic | Lower operator trust | validator checks sections | Add style and overclaim checks | MCP | MCP Golden Answer Quality Gate | small |
| P2 | MCP unknown-ID behavior incomplete | Enumeration risk | backlog item | Add structured not-found tests | MCP | MCP Unknown Resource Closure | small |
| P2 | MCP audit failure policy unclear | Missing evidence under failure | audit recorder tests | Decide fail-open vs fail-closed by operation | MCP | MCP Audit Failure Policy | medium |
| P2 | Large log/audit pagination incomplete | Memory and leakage risk | log truncation tests | Add resource-wide pagination/caps | MCP/SRE | MCP Response Limits | medium |
| P2 | Service-account examples lack production flow | Integration friction | docs | Add safe examples without secrets | Auth | Service Account Operator Guide | small |
| P3 | Docs volume is high | Maintainer navigation burden | docs tree | Add landing pages and pruning pass | Docs | Documentation Information Architecture | medium |
| P3 | Examples may be stale | Confusing demos | example validation | Add example ownership and freshness check | QA | Example Stewardship | small |
| P3 | Release docs mention future versions | Maturity confusion | release docs | Keep future checklist wording | Release | Release Docs Cleanup | small |
| P3 | Local dev proxy/network setup is undocumented | Contributor friction | user reports | Add troubleshooting note if needed | DX | Contributor Network Notes | small |
| P3 | Frontend dependency drift can break CI | CI failures | `make verify-web` | Keep lockfile and build check | Web | Web Build Stability | small |
| P3 | Benchmarks are not release-gated | Performance regressions | benchmark targets | Add baseline dashboard later | Performance | Benchmark Baseline | medium |
| P3 | Optional smoke scripts may be skipped too often | Weak confidence | Make targets skip safely | Record skip reasons in release checklist | QA | Smoke Result Reporting | small |
| P3 | Contributor adapter templates are basic | Integration quality varies | plugin docs | Add adapter authoring checklist | Plugin | Adapter Authoring Guide | small |
| P3 | Object store backup is mostly documented | Artifact evidence restore gaps | backup docs | Add object store fake restore test | Ops | Object Store Restore Proof | medium |
| P3 | Production terminology may drift again | Public trust risk | status docs | Add enterprise docs validator | Docs | Maturity Claim Guard | small |
