# MCP Enterprise Risk Register

Current maturity: **local beta foundation, remote no-go**.

| Priority | Risk | Impact | Evidence | Mitigation | Owner Area | Suggested Goal | Effort |
|---|---|---|---|---|---|---|---|
| P0 | Remote MCP without tenant filters could expose cross-project data | Confidential delivery metadata leak | `docs/security/MCP_TENANT_SCOPE_REVIEW.md` | Add resource ownership checks and fixture tests | MCP/Security | Tenant-Scoped MCP Resource Filtering | large |
| P0 | Remote MCP without auth would expose control-plane state | Unauthorized data access | remote MCP not implemented | Require OAuth/OIDC or service-account bearer | MCP/Security | Remote Read-only MCP Contract | medium |
| P0 | Runner tokens could become an AI access path if remote auth is careless | Runner credential escalation | local tests reject runner token | Keep runner-token deny tests for all transports | MCP/Runner | Runner Token Boundary Closure | medium |
| P0 | Action MCP could mutate delivery state without human governance | Unsafe apply/sync/rollback/approval | denied action tools | Keep action MCP no-go | MCP/Governance | MCP Action Threat Model | medium |
| P0 | Secret-bearing logs could leak through MCP | Credential exposure | redaction tests and scenarios | Transport-wide response scanner | MCP/Security | MCP Redaction Closure | medium |
| P1 | Prompt injection in logs/events could alter AI guidance | Unsafe operator advice | prompt injection scenarios | Expand adversarial corpus | MCP/Security | MCP Prompt Injection Corpus | medium |
| P1 | Plan-only output may be read as applied state | Operational confusion | plan tools return `mutated=false` | Stronger output labels and golden-answer linting | MCP/Product | MCP Answer Contract Hardening | small |
| P1 | Audit search may be too broad for remote exposure | Compliance data overexposure | audit.read resource exists | Scope and pagination audit search | MCP/Compliance | MCP Audit Scope Closure | medium |
| P1 | Response size limits are inconsistent | Resource exhaustion | log truncation exists, general cap missing | Add response cap helper before remote | MCP/SRE | MCP Transport Limits | medium |
| P1 | Rate limits are absent | Abuse and noisy neighbor risk | no remote transport | Add rate-limit design and tests | MCP/SRE | MCP Remote Abuse Controls | medium |
| P1 | Remote audit attribution proof is missing | Weak incident reconstruction across remote clients | local/compliance tests and Postgres MCP hash-chain proof exist | Add remote client identity and scope attribution tests before remote exposure | MCP/Compliance | MCP Remote Audit Contract | medium |
| P1 | New MCP tools may bypass scenario coverage | Drift | validator checks current corpus | Require permission matrix and scenario rows for new tools | MCP/QA | MCP Catalog Drift Gate | small |
| P1 | Golden answers may become generic or overconfident | Bad operator guidance | golden answers exist | Add style/safety linter | MCP/Product | MCP Golden Answer Quality Gate | small |
| P1 | Plugin capability data could reveal internal inventory remotely | Metadata leak | plugin resource exists | Scope plugin resource output | MCP/Plugin | MCP Metadata Scope Review | small |
| P1 | Runner summary may expose fleet topology | Operational metadata leak | runner summary resource exists | Add group/environment filters | MCP/Runner | Tenant-Scoped Runner Summary | medium |
| P2 | Unknown resource errors may disclose existence | Enumeration | unknown-ID coverage incomplete | Structured not-found without tenant hints | MCP/Security | MCP Unknown-ID Matrix | small |
| P2 | Local stdio trust assumptions may be misunderstood | Operator misuse | docs state local only | Keep README/status explicit | Docs | MCP Operator Docs Cleanup | small |
| P2 | Remote TLS/ingress profile is undefined | Deployment confusion | remote no-go | Document only after remote auth exists | Ops | MCP Remote Deployment RFC | medium |
| P2 | Scenario fixtures are not live production evidence | False confidence | scenario tests use fixtures | Add incident replay corpus later | QA | MCP Incident Replay Harness | large |
| P2 | Audit recorder failure behavior needs policy | Missing audit under failure | recorder tests focused on success/denial | Decide fail-open/fail-closed by surface | Compliance | MCP Audit Failure Policy | medium |
