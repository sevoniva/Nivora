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

The current backend also includes a foundation Policy catalog at `/api/v1/policies` and `nivora policy list/create/get/update/disable`. It stores built-in gate definitions such as digest requirements and finding thresholds, but it is not a full enterprise policy platform and is memory-backed in current runtime wiring.

## Common Confusion

Policy is not just documentation. A Policy should be evaluated and its result should affect workflow state.
