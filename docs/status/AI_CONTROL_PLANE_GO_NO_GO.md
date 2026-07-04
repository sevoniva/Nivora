# AI Control Plane Go / No-Go

Current maturity: **hardened beta-candidate foundation, not production-ready**.

## Decisions

| Area | Decision | Reason |
|---|---|---|
| Local stdio MCP | go for local maintainer use | read-only/plan-only resources and tools are tested; blocked actions are denied |
| Local AI operator demo | go with scripted local data | 29 validated scenarios and golden answers exist |
| Remote read-only MCP | experimental opt-in foundation | `POST /api/v1/mcp/rpc` exists with bearer/static-token tests, runner-token rejection, body/response caps, in-process per-subject rate limits, and blocked actions; broader OIDC/service-account, tenant, pagination, distributed rate-limit, and audit-attribution proof remains incomplete |
| Remote plan-only MCP | experimental opt-in foundation | plan tools share the same JSON-RPC path and return `mutated=false`; broader abuse controls and pagination still need hardening |
| Action MCP | no-go | apply, sync, rollback, approval, token, secret, runner, host, Git, prune, and delete actions remain blocked |
| Production use | no-go | broader platform hardening remains incomplete |

## Top 20 Risks

1. Remote MCP tenant filtering is incomplete for broad exposure.
2. Remote MCP OIDC/service-account scoped contract tests are incomplete.
3. Remote distributed rate limits do not exist; JSON-RPC request/response caps, request timeouts, and in-process per-subject request rate limits exist.
4. Audit search can expose broad metadata without future remote scope filters and pagination.
5. Remote MCP audit policy for client identity and per-client attribution is not complete.
6. Runner summary needs environment/group filtering before remote exposure.
7. AI can misread plan-only output as execution evidence.
8. Logs/events/manifests can contain prompt-injection text.
9. Shell executor is not an OS-level sandbox.
10. Kubernetes live state cannot be proven from plan-only summaries.
11. GitOps sync is not remotely safe.
12. Host deploy remains guarded and not remotely safe.
13. Artifact inspect is local parsing, not registry resolution.
14. Backup/restore posture is not proven by MCP.
15. Production install drills are outside MCP evidence.
16. External integrations remain foundation or experimental.
17. Cross-project ID access must be proven before remote exposure.
18. Golden answers can drift if scenarios are not maintained.
19. New tools could bypass mutated=false unless tests stay strict.
20. Documentation could overstate beta capabilities if not reviewed; tenant IDOR, cross-project audit, missing resource, massive log, and evidence-bundle scenarios must stay in the corpus.

Production configuration now also fails closed when MCP is enabled with `mcp.readonly=false`; this guards the current local foundation from accidental action-tier configuration.

## Top 20 Missing Tests

1. Expanded remote OIDC/service-account scoped MCP contract test.
2. Remote service-account scope test.
3. Remote runner-token denial test.
4. Tenant-filtered PipelineRun resource test.
5. Tenant-filtered DeploymentRun resource test.
6. Tenant-filtered ReleaseExecution resource test.
7. Tenant-filtered runner summary test.
8. Tenant-filtered audit search test.
9. Remote MCP pagination contract test for large result sets.
10. Distributed MCP rate-limit design and tests.
11. Remote MCP request timeout depth test.
12. Prompt-injection corpus expansion beyond current fixtures.
13. Golden answer drift check in CI.
14. Resource pagination test for logs.
15. Resource pagination test for audit.
16. Service-account no-role negative test for every tool class.
17. Unknown resource structured error test for every URI class.
18. Remote transport no-mutation proof for plan-only tool aliases; local alias non-mutation is covered by `TestMCPPlanOnlyToolsReturnMutatedFalseAndDoNotCreateDeploymentRuns`.
19. Remote TLS/deployment profile smoke test.
20. Remote MCP per-client audit attribution test.

## Next 3 Goals

1. **Remote Read-Only MCP Contract Hardening**
   - Scope: auth model, service-account tokens, runner-token denial, response cap enforcement, request timeout enforcement, and stronger multi-process rate limits.
   - Non-goals: action tools, secret retrieval, remote plan mutation.
   - Acceptance: remote read-only remains disabled by default and all expanded remote contract tests pass.

2. **Tenant-Scoped MCP Resource Filtering**
   - Scope: project/environment ownership checks for PipelineRun, DeploymentRun, ReleaseExecution, runner summary, audit search, security summary.
   - Non-goals: multi-tenant UI, SSO provider implementation.
   - Acceptance: cross-tenant fixture tests fail closed.

3. **MCP Remote Audit Contract Hardening**
   - Scope: remote-style actor/client metadata, tenant scope, pagination, and audit attribution tests on top of the existing Postgres MCP hash-chain proof.
   - Non-goals: new audit product features or action-tier MCP.
   - Acceptance: remote MCP remains disabled by default and audit contract tests fail closed for missing identity or scope.

## Next 30 Backlog Items

| Rank | Task | Type | Dependency |
|---:|---|---|---|
| 1 | Expand remote MCP OIDC/service-account auth contract tests | security | current bearer route tests |
| 2 | Add tenant fixture model for MCP resources | test | auth scopes |
| 3 | Add remote MCP audit attribution tests | test | remote auth contract |
| 4 | Add audit search pagination | code | API/store support |
| 5 | Add log resource pagination | code | log store support |
| 6 | Add MCP pagination contract for remote-scale event/log/audit reads | code | current response cap |
| 7 | Add deeper remote MCP request timeout contract tests | test | current JSON-RPC timeout |
| 8 | Add rate-limit design doc | docs | remote RFC |
| 9 | Add service-account examples without secrets | docs | auth docs |
| 10 | Add runner summary scope filters | code | tenancy model |
| 11 | Add PipelineRun ownership checks | code | tenancy model |
| 12 | Add DeploymentRun ownership checks | code | tenancy model |
| 13 | Add ReleaseExecution ownership checks | code | tenancy model |
| 14 | Add security summary scope filters | code | tenancy model |
| 15 | Expand prompt-injection scenarios to 40 cases | test | current scenarios |
| 16 | Add golden answer style linter | test | validator |
| 17 | Add docs link checker for MCP docs | docs | docs tooling |
| 18 | Add OpenAPI note for MCP status endpoint if added later | API | route decision |
| 19 | Add smoke test for `nivora-mcp` stdio denial path | test | current CLI |
| 20 | Add MCP resource unknown-ID matrix tests | test | fixtures |
| 21 | Extend plan-only alias no-mutation proof to any future remote transport | test | remote contract |
| 22 | Add compliance recorder failure behavior test | test | audit recorder |
| 23 | Add operator runbook for local MCP incident review | docs | current docs |
| 24 | Add external-client threat model update | security | remote RFC |
| 25 | Add disabled-by-default remote config validation | code | remote config |
| 26 | Add TLS/ingress deployment guidance | docs | remote design |
| 27 | Add tenant-safe capability document view | code | tenant policy |
| 28 | Add MCP scenario for missing resource by type | test | scenario validator |
| 29 | Add MCP scenario for malformed local plan input | test | tool validation |
| 30 | Add maintainer release checklist item for MCP docs drift | docs | release process |

## Release Recommendation

Use local stdio MCP in demos with scripted data and clear limits. Remote read-only MCP may be tested only as an explicitly enabled foundation; continue hardening before broad remote exposure or action MCP.
