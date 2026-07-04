# AI Control Plane Go / No-Go

Current maturity: **hardened beta-candidate foundation, not production-ready**.

## Decisions

| Area | Decision | Reason |
|---|---|---|
| Local stdio MCP | go for local maintainer use | read-only/plan-only resources and tools are tested; blocked actions are denied |
| Local AI operator demo | go with scripted local data | 29 validated scenarios and golden answers exist |
| Remote read-only MCP | no-go until next hardening phase | auth, tenant filters, remote per-client rate limits, pagination, and remote audit tests are missing; local request/response caps, request timeouts, and stdio request rate limits exist but are not remote transport proof |
| Remote plan-only MCP | no-go | plan tools need remote abuse controls |
| Action MCP | no-go | apply, sync, rollback, approval, token, secret, runner, host, Git, prune, and delete actions remain blocked |
| Production use | no-go | broader platform hardening remains incomplete |

## Top 20 Risks

1. Remote MCP tenant filtering is incomplete.
2. Remote MCP auth/OAuth contract tests do not exist.
3. Remote per-client rate limits do not exist; local MCP request/response caps, request timeouts, and stdio request rate limits exist but remote transport limits are not proven.
4. Dedicated Postgres MCP audit-chain test is not complete.
5. Audit search can expose broad metadata without future scope filters.
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
20. Documentation could overstate beta capabilities if not reviewed.
21. Tenant IDOR, cross-project audit, missing resource, massive log, and evidence-bundle scenarios must stay in the corpus.

## Top 20 Missing Tests

1. Remote bearer auth MCP contract test.
2. Remote service-account scope test.
3. Remote runner-token denial test.
4. Tenant-filtered PipelineRun resource test.
5. Tenant-filtered DeploymentRun resource test.
6. Tenant-filtered ReleaseExecution resource test.
7. Tenant-filtered runner summary test.
8. Tenant-filtered audit search test.
9. Remote MCP response-size cap contract test.
10. MCP per-client rate-limit test.
11. Remote MCP request timeout contract test.
12. Postgres MCP audit-chain integration test.
13. Prompt-injection corpus expansion beyond current fixtures.
14. Golden answer drift check in CI.
15. Resource pagination test for logs.
16. Resource pagination test for audit.
17. Service-account no-role negative test for every tool class.
18. Unknown resource structured error test for every URI class.
19. No-mutation test for every plan-only tool alias.
20. Remote TLS/deployment profile smoke test.

## Next 3 Goals

1. **Remote Read-Only MCP Contract Hardening**
   - Scope: auth model, service-account tokens, runner-token denial, remote transport proposal tests, response cap enforcement, request timeout enforcement, rate limits.
   - Non-goals: action tools, secret retrieval, remote plan mutation.
   - Acceptance: remote read-only remains disabled by default and all remote contract tests pass.

2. **Tenant-Scoped MCP Resource Filtering**
   - Scope: project/environment ownership checks for PipelineRun, DeploymentRun, ReleaseExecution, runner summary, audit search, security summary.
   - Non-goals: multi-tenant UI, SSO provider implementation.
   - Acceptance: cross-tenant fixture tests fail closed.

3. **MCP Audit Durability Integration Proof**
   - Scope: Postgres-backed compliance recorder test for MCP resource/tool/prompt/denied events, hash-chain verification, redaction proof.
   - Non-goals: new audit product features.
   - Acceptance: optional Postgres integration target proves persisted MCP audit entries.

## Next 30 Backlog Items

| Rank | Task | Type | Dependency |
|---:|---|---|---|
| 1 | Add remote MCP auth contract tests | security | none |
| 2 | Add tenant fixture model for MCP resources | test | auth scopes |
| 3 | Add Postgres MCP audit-chain test | test | Postgres harness |
| 4 | Add audit search pagination | code | API/store support |
| 5 | Add log resource pagination | code | log store support |
| 6 | Extend local response-size caps to any future remote transport contract | code | transport design |
| 7 | Add remote MCP request timeout contract tests | test | transport design |
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
| 21 | Add plan-only alias no-mutation tests | test | current aliases |
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

Use local stdio MCP in demos only with scripted data and clear limits. Continue hardening before remote MCP or new feature expansion.
