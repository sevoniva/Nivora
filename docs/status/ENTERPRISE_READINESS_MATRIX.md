# Nivora Enterprise Readiness Matrix

Current maturity: **hardened beta-candidate foundation, not production-ready**.

Maturity values used here:

- `production-candidate`: enough implementation, tests, and docs exist for a production-candidate review, but final release gates remain.
- `beta`: useful and tested, but not hardened for broad production operation.
- `foundation`: model, adapters, tests, or docs exist, but important behavior remains incomplete.
- `placeholder`: shape exists mainly to reserve API/model direction.
- `documented-only`: described but not materially implemented.
- `missing`: not present.
- `unsafe-for-production`: available only with strong warnings or explicit operator isolation.

No row in this matrix should be read as a GA production claim.

| Capability | Maturity | Evidence | Tests | Docs | Production Risk | Required Next Step | Owner Area |
|---|---|---|---|---|---|---|---|
| API server | beta | `cmd/nivora-server`, `internal/api/http` | `go test ./internal/api/http/...`, smoke API | README, API inventory | Broad route surface still maturing | Freeze route contracts and auth expectations | Platform |
| Worker | foundation | `cmd/nivora-worker`, runtime recovery paths | runtime smoke and Postgres integration | workflow/runtime docs | Long-running orchestration still simple | Add worker restart and reconcile e2e gates | Runtime |
| Runner binary | beta | `cmd/nivora-runner`, runner protocol routes | runner route and token tests | runner security docs | Runner executes untrusted work | Default isolated production runner profile | Runner |
| CLI | beta | `cmd/nivora` | smoke CLI, command tests | README examples | Some commands are local/foundation only | Mark local/noop/network behavior per command | DX |
| Config loader | production-candidate | `internal/infra/config` | config validation tests | operations config docs | Unsafe overrides can still be misused | Keep production unsafe matrix in CI | SRE |
| Health/readiness | beta | `/healthz`, `/readyz`, diagnostics | API smoke | operations docs | Dependency health depth varies | Add dependency-specific readiness proof | SRE |
| Diagnostics | beta | `/api/v1/system/diagnostics` | handler tests | troubleshooting docs | Must avoid sensitive metadata | Add redaction regression checks | SRE |
| Versioning | beta | `VERSION`, CLI/server version paths | CLI smoke | release docs | Release state still pre-GA | Stabilize version automation | Release |
| Graceful shutdown | foundation | app runtime wiring | limited tests | architecture docs | Long jobs may need stronger cancel handling | Add shutdown/reconcile integration test | Runtime |
| Request/correlation IDs | beta | observability middleware | middleware tests | observability docs | Cross-process propagation incomplete | Extend to runner/job/deployment events | Observability |
| Structured logging | beta | slog-based infra | redaction tests | logging/security docs | Operator fields may be inconsistent | Add log field contract test | Observability |
| PipelineRun | beta | pipeline usecase and Postgres store | local smoke, Postgres recovery | workflow docs | Not a heavy workflow engine | Worker recovery closure | Runtime |
| StageRun | beta | pipeline store models | pipeline store tests | workflow docs | Recovery tied to PipelineRun store | Add stage-level stale tests | Runtime |
| JobRun | beta | job claim and store methods | claim/recovery tests | runner protocol docs | Lease behavior needs more live proof | Multi-process runner recovery drill | Runner |
| StepRun | foundation | pipeline records and store | store tests | workflow docs | Fine-grained retry/recovery limited | Step timeout/cancel tests | Runtime |
| DeploymentRun | beta | deployment usecase/Postgres store | deployment recovery integration | deployment docs | Full CD semantics incomplete | Live fake-adapter recovery e2e | Deployment |
| DeploymentPlan | beta | deployment plan persistence | recovery integration | deployment model docs | Plan/live drift not fully proven | Persisted diff snapshot tests | Deployment |
| Release | beta | release store | release persistence tests | release docs | Approval/resume lifecycle incomplete | Release governance recovery test | Release |
| ReleaseArtifact | beta | release binding store | release recovery tests | artifact/release docs | External registry resolution optional | Registry credential integration proof | Artifact |
| ReleasePlan | beta | orchestration store | release plan recovery | release orchestration docs | Plan updates not deeply versioned | Immutable plan baseline checks | Release |
| ReleaseExecution | beta | orchestration store | non-terminal/stale recovery tests | release execution docs | Complex rollback orchestration incomplete | Multi-target failure recovery tests | Release |
| EventOutbox | beta | runtime outbox table/store | outbox recovery tests | runtime recovery docs | Publisher retry semantics simple | Worker outbox retry e2e | Runtime |
| LogChunk | beta | runtime log stores | log ordering/recovery tests | runner protocol docs | High-volume ingestion limits need proof | Load and pagination tests | Observability |
| AuditLog | beta | compliance store and runtime audits | hash-chain and store tests | audit/evidence docs | Not every audit path has same hash proof | Unified audit-chain coverage | Compliance |
| Runner heartbeat | beta | runner store/routes | heartbeat/offline tests | runner docs | Fleet-scale heartbeat load unproven | Heartbeat load benchmark | Runner |
| Runner claim | beta | claim/lease store logic | claim recovery tests | runner protocol docs | Scope matching needs broader fixtures | Environment/project scoped claim tests | Runner |
| Cancellation | foundation | run/job cancel fields | route/runtime tests | runtime docs | Executor cancellation best-effort | Reconcile cancel e2e | Runtime |
| Timeout | foundation | config and state checks | unit tests | runtime docs | Timeout reconcile coverage incomplete | Timeout recovery matrix | Runtime |
| Retry | foundation | outbox retry and pipeline attempts | focused tests | runtime docs | Execution retry semantics limited | Retry policy model hardening | Runtime |
| Stale run recovery | beta | stale query methods | Postgres recovery tests | runtime recovery docs | Full worker reconcile loop needs more proof | Multiprocess recovery gate | Runtime |
| Restart recovery | beta | optional Postgres integration tests | `NIVORA_RUN_POSTGRES_INTEGRATION=true` tests | runtime recovery docs | Optional locally, CI postgres job covers selected paths | Expand to full service bootstrap | Runtime |
| YAML deployment plan | beta | deployment parser/planner | CLI/API tests | deployment docs | Manifest edge cases remain | Manifest fuzz/fixture expansion | Deployment |
| Kubernetes dry-run | beta | guarded executor adapter | fake/local tests | Kubernetes ops docs | Cluster semantics not required in CI | Optional live cluster smoke | Deployment |
| Kubernetes apply | foundation | guarded apply paths | guard tests | Kubernetes ops docs | Real cluster safety not GA | Namespace/context policy hardening | Deployment |
| Rollout watch | foundation | workload health/watch model | fake tests | Kubernetes docs | CRD/custom health limited | Workload fake adapter expansion | Deployment |
| Resource inventory | beta | inventory extraction/persistence | extraction and recovery tests | resource model docs | Live inventory incomplete without cluster | Live query fallback tests | Deployment |
| Health evaluation | beta | lightweight evaluator | evaluator tests | health docs | Not a controller | Document unsupported CRDs clearly | Deployment |
| Rollback plan | beta | manifest snapshot/rollback plan | recovery tests | rollback docs | Execution guarded and limited | Restore strategy fake e2e | Deployment |
| Rollback execution | foundation | guarded execution surface | guard tests | rollback docs | No delete/prune by default | Confirmation/audit e2e | Deployment |
| GitOps plan | foundation | local GitOps adapter | local repo tests | GitOps docs | Push/sync guarded | Revision rollback proof | GitOps |
| Argo CD status | foundation | noop/HTTP skeleton | fake adapter tests | Argo docs | No production app lifecycle | Read-only status contract hardening | GitOps |
| Argo CD guarded sync | foundation | allow+confirm model | guard tests | Argo docs | Sync remains disabled by default | Remote credential/audit proof | GitOps |
| Host deployment | foundation | host model/noop/SSH skeleton | guard tests | host docs | Remote shell risk | Fake SSH batch/rollback proof | Host |
| Release orchestration | beta | release orchestration usecase | plan/deploy tests | release orchestration docs | Approval/resume incomplete | Governance integration e2e | Release |
| Auth | beta | dev/token/OIDC foundation | middleware tests | auth docs | Full enterprise SSO incomplete | OIDC provider lifecycle tests | Security |
| RBAC | beta | permission evaluator/middleware | route matrix tests | route matrix docs | Coverage broad but not every tenant path | Auto-generate route/permission drift reports | Security |
| Service account | foundation | auth model/store | focused tests | auth docs | Token lifecycle limited | Rotation/revocation coverage | Security |
| API token | foundation | token metadata/hash paths | token tests | auth docs | Rotation/use analytics limited | Expiry and audit proof | Security |
| Runner token | beta | token hash/runner routes | negative tests | runner security docs | Token storage and revocation need operational proof | Rotation/revoke e2e | Runner |
| Secret management | foundation | ports/providers/redaction | provider/redaction tests | secret docs | External providers not production-validated | Vault/KMS contract tests | Security |
| Credential management | foundation | metadata store/API | route/safety tests | credential docs | Validation adapter coverage limited | Provider-specific validation contracts | Security |
| Redaction | beta | redaction helpers/tests | API/MCP/secret tests | security docs | New fields can drift | Central sensitive-key corpus gate | Security |
| Route permission coverage | beta | route matrix + tests | `rbac_matrix_test.go` | matrix docs | Test logs some expected failures softly | Convert more soft logs to hard assertions | Security |
| Tenant isolation | foundation | tenant tests, scoped MCP artifact/security/runner filters | route tenant tests and MCP scope tests | tenancy docs | Runner ownership is still label-based and broad metadata surfaces need remote-safe profiles | First-class runner group ownership and remote-safe metadata profile | Security |
| Audit evidence | foundation | compliance store/evidence | hash-chain tests | audit docs | Enterprise export/retention incomplete | Evidence bundle release e2e | Compliance |
| Threat model | beta | security docs | review checklists | threat model docs | Needs recurring release review | Threat model release gate | Security |
| Policy gates | foundation | built-in rules | policy tests | policy docs | OPA/Kyverno not integrated | Policy decision persistence e2e | Security |
| Approval gates | foundation | approval usecase/routes | approval tests | approval docs | Resume workflow incomplete | Approval resume/reject e2e | Governance |
| Change windows | foundation | evaluator/routes | evaluator tests | change window docs | Timezone/holiday rules basic | Window boundary tests | Governance |
| PostgreSQL persistence | beta | Runtime/governance/catalog stores + migrations | Postgres integration | database docs | Full live recovery still expanding | Mandatory release gate for integration job | Database |
| Migrations | beta | migrations 000001-000011 | up/down tests | persistence docs | Backward compatibility not fully proven | Migration compatibility matrix | Database |
| Migration rollback | foundation | down files and tests | migration up/down test | database docs | Data-preserving rollback not guaranteed | Release-specific rollback guidance | Database |
| Backup | documented-only | backup scripts/docs | optional smoke | backup docs | Full backup automation not CI gate | Automated backup/restore drill | Ops |
| Restore | documented-only | restore docs/scripts | optional smoke | restore docs | Full restore drill limited | Restore job in integration profile | Ops |
| HA guidance | documented-only | HA/DR docs | docs checks | HA docs | Not live-tested | Multi-instance smoke | Ops |
| Disaster recovery | documented-only | DR docs/runbooks | limited smoke | DR docs | Dependency failure injection limited | Failure-mode tests | Ops |
| Helm production profile | beta | `values-production.yaml` | helm safety smoke | install docs | Live cluster install not mandatory | Optional kind production smoke | Packaging |
| Docker Compose production-like profile | beta | production compose example | compose smoke | install docs | Not a full production recipe | Live compose smoke gate | Packaging |
| Config validation | beta | config package tests | production unsafe tests | config docs | New unsafe flags can drift | Unsafe flag registry | SRE |
| Metrics | foundation | `/metrics` and counters | metrics tests | observability docs | Dashboard/export depth limited | SLO metric coverage | Observability |
| Tracing | placeholder | config placeholder | minimal tests | observability docs | No full trace export | OTel propagation proof | Observability |
| SLO | documented-only | SLO docs/runbooks | docs only | operations docs | Not enforced | Alert rule examples and tests | SRE |
| Runbooks | beta | operations runbooks/scripts | runbook smoke | operations docs | Not all failure modes scripted | Expand runbook checks | SRE |
| Troubleshooting | beta | troubleshooting docs | docs checks | operations docs | Needs real incident drills | Operator drill checklist | SRE |
| Local stdio MCP | beta | `cmd/nivora-mcp` | MCP tests/smoke | MCP docs | Local trust boundary only | Keep local-only warnings | MCP |
| Remote MCP readiness | foundation | RFC/readiness docs | matrix tests | remote MCP docs | No remote transport | Auth/scope/limit design first | MCP |
| MCP auth | foundation | local subject/RBAC | MCP auth tests | MCP security docs | Remote auth missing | Service-account bearer contract | MCP |
| MCP RBAC | beta | permission matrix/tests | MCP matrix tests | MCP permission docs | Remote auth and per-client policies missing | Resource ownership tests | MCP |
| MCP tenant scope | foundation | tenant scope review, scoped PipelineDefinition/Artifact/SecurityScan/Runner filtering | PipelineRun/PipelineDefinition/DeploymentRun/ReleaseExecution/Artifact/SecurityScan/Runner MCP scope tests | tenant scope docs | Remote IDOR risk remains for future resource families and unscoped historical records | First-class runner ownership, remote-safe metadata profile, and negative tests for new resources | MCP |
| MCP prompt injection defense | beta | prompt templates/scenarios | prompt tests | MCP security docs | Corpus can grow | Add adversarial scenarios | MCP |
| MCP redaction | beta | sanitizer/tests | server/scenario tests | MCP security docs | New fields can drift | Central corpus gate | MCP |
| MCP audit | foundation | compliance recorder and Postgres hash-chain proof | unit/compliance tests plus `TestPostgresIntegrationMCPAuditHashChain` | MCP audit docs | Remote client attribution and audit scope policy are still future work | Remote audit contract tests before remote MCP exposure | MCP |
| MCP scenario corpus | beta | scenario YAML | validator/tests | scenario README | Coverage must grow with tools | Require enterprise abuse cases | MCP |
| MCP golden answers | beta | golden markdown | validator/tests | scenario README | Style drift possible | Golden-answer linter | MCP |
| MCP rate limit | foundation | local stdio JSON-RPC request limit | MCP rate-limit tests | MCP security docs | Remote per-client rate limits missing | Add remote per-client limits before exposure | MCP |
| MCP request body cap | foundation | local stdio JSON-RPC request body cap | MCP request-size tests | MCP security docs | Remote body-size proof missing | Add remote transport body limits before exposure | MCP |
| MCP response size cap | beta | JSON-RPC transport cap plus log/resource truncation | transport cap and truncation tests | MCP docs | Remote pagination and body-size proof still missing | Add remote transport limits before exposure | MCP |
| MCP action denial | beta | denied tool registry | scenario tests | permission matrix | New action aliases could drift | Deny-list catalog test | MCP |
| MCP operator journey quality | beta | operator journeys and scenarios | scenario validator | AI operator docs | Still fixture-based | Real incident replay corpus | MCP |
