# Policy

A Policy is an enforceable gate in the delivery lifecycle.

## Why It Exists

Policies help decide whether a PipelineRun, Release, or DeploymentRun may proceed. They can represent security checks, environment rules, approval requirements, artifact requirements, or operational constraints.

## Relationships

- Produces PolicyResults.
- May apply before build, before release, before deployment, or during verification.
- Should be recorded in audit context.

## Current Implementation

Phase 2.1 calls a PolicyEngine during DeploymentRun pre-check. Early runtimes may still use an allow-all engine for local planning, so operators should not treat that path as production enforcement.

Phase 3.0 adds security policy gates backed by SecurityScan and SecurityFinding records. The local implementation supports noop/fake scanners and minimal built-in rules for critical findings, high finding warnings, mutable artifact tags, digest requirements, and simple manifest risks. Real Trivy, Cosign, SBOM, OPA, Kyverno, Gatekeeper, and enterprise policy integrations remain future work.

The current backend also includes a foundation Policy catalog at `/api/v1/policies` and `nivora policy list/create/get/update/disable --token-env NIVORA_AUTH_TOKEN`. It stores built-in gate definitions such as digest requirements and finding thresholds. Saved policies can be evaluated through `/api/v1/policies/{id}/evaluate` and `nivora policy evaluate <policy-id> --subject ...`; disabled policies are rejected. Policies can be attached to `global`, `org`, `project`, `application`, `environment`, `target`, `release`, or `deployment` scopes through `/api/v1/policies/{id}/attachments` and `nivora policy attach --token-env NIVORA_AUTH_TOKEN`.

Security scans can apply a saved policy explicitly with `policyId`. If a scan omits both `policyId` and an ad hoc `policy`, the server resolves an enabled security policy attachment for the request `environmentId`, then `projectId`, then `global`, and records the applied `policyId` in the PolicyResult.

DeploymentRun and ReleaseExecution security pre-checks also resolve saved security policy attachments when the server runtime is wired with the policy catalog. A deny decision stops execution; a require-approval decision enters the existing approval workflow instead of continuing as a warning.

Policy attachments are control-plane metadata. They declare where a built-in policy is intended to apply, but they are not an external policy distribution system. The catalog and attachments can use PostgreSQL when the runtime store is configured for Postgres; local development can still use in-memory stores.

## Common Confusion

Policy is not just documentation. A Policy should be evaluated and its result should affect workflow state.
