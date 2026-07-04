# AI Control Plane Beta Readiness

Current MCP maturity: local stdio foundation, beta-readiness hardening in progress. Nivora is not production-ready.

Current scenario evidence:

- 29 validated MCP operator scenarios in `examples/mcp/scenarios/`
- 29 matching golden answers in `examples/mcp/golden-answers/`
- fixture-backed scenario tests in `internal/api/mcp/scenario_test.go`
- validation script: `scripts/validate-mcp-scenarios.sh`
- broader status: `docs/status/AI_CONTROL_PLANE_DEEP_AUDIT.md`, `docs/status/AI_OPERATOR_JOURNEYS.md`, `docs/status/AI_CONTROL_PLANE_GO_NO_GO.md`

## What AI Can Safely Do

- Read current capability status.
- Read runtime posture.
- Read PipelineRun, DeploymentRun, ReleaseExecution, runner, security, plugin, and audit summaries when RBAC permits.
- Use prompts that require facts vs inference, unknowns, safe checks, and no production-ready claims.
- Run local plan-only helpers that return `mutated=false`.
- Inspect artifact references locally without registry network access.
- Evaluate local policy inputs without persisting policy results.

## What AI Cannot Do

- Apply Kubernetes manifests.
- Sync Argo CD.
- Execute rollback.
- Approve or reject governance requests.
- Retrieve secrets.
- Rotate or reveal tokens.
- Register runners.
- Push Git changes.
- Deploy to remote hosts.
- Delete or prune Kubernetes resources.
- Prove production readiness from MCP state alone.

## Go / No-Go

| Area | Decision | Reason |
|---|---|---|
| Local stdio MCP | go for maintainer/local use | read-only and plan-only, denied action tools, runner-token rejection, redaction, response caps, request timeouts, and compliance audit exist |
| Remote read-only MCP | conditional next phase | design is documented, but OAuth/OIDC, tenant filters, rate limits, pagination, remote timeout/cap proof, and remote audit tests are required |
| Remote action MCP | no-go | apply/sync/rollback/approval/token/secret/runner/host/Git/Kubernetes delete actions are intentionally blocked |

## Evidence

- MCP implementation: `cmd/nivora-mcp`, `internal/api/mcp`, `internal/app/mcp`
- Permission matrix: `docs/security/MCP_PERMISSION_MATRIX.md`
- Threat model: `docs/security/mcp-threat-model.md`
- Remote read-only design: `docs/rfcs/remote-mcp-read-only.md`
- Operator review: `docs/status/AI_CONTROL_PLANE_PRODUCT_REVIEW.md`
- Golden scenarios: `examples/mcp/scenarios/`
- Golden answers: `examples/mcp/golden-answers/`
- Scenario tests: `internal/api/mcp/scenario_test.go`
- Catalog drift and plan-only non-mutation tests: `TestMCPPermissionMatrixCoversCatalogEntries`, `TestMCPPlanOnlyToolsReturnMutatedFalseAndDoNotCreateDeploymentRuns`
- Smoke script: `scripts/smoke-mcp-local.sh`
- Postgres MCP audit hash-chain proof: `TestPostgresIntegrationMCPAuditHashChain` and CI `MCP audit hash chain verification`
- Production configuration validation rejects enabled MCP when `mcp.readonly=false`

## Remaining Top Risks

1. Remote MCP has an opt-in bearer-gated read-only JSON-RPC foundation, but broad remote identity lifecycle and tenant filtering are not fully proven.
2. Remote MCP audit attribution has basic actor/operation coverage, but richer remote client metadata and request/correlation attribution are still future work.
3. Prompt-injection resilience is mostly prompt guidance and scenario coverage, not a full adversarial test corpus.
4. AI can still misinterpret unknown live state if prompts are ignored.
5. Runner shell execution is hardened but not an OS-level sandbox.
6. Production install/restore drills remain broader platform blockers.

## Recommended Next Phase

Proceed with **Remote Read-Only MCP Design Validation** before any action-tier work:

- add OAuth/OIDC or service-account scoped remote auth contract tests
- add tenant-filtered MCP resource tests
- add remote response-size, request-timeout, and rate-limit contract tests
- add richer remote MCP client/request attribution and scope contract tests on top of the existing Postgres hash-chain proof
- expand adversarial log/prompt-injection fixtures
