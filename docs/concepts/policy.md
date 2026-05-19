# Policy

A Policy is an enforceable gate in the delivery lifecycle.

## Why It Exists

Policies help decide whether a PipelineRun, Release, or DeploymentRun may proceed. They can represent security checks, environment rules, approval requirements, artifact requirements, or operational constraints.

## Relationships

- Produces PolicyResults.
- May apply before build, before release, before deployment, or during verification.
- Should be recorded in audit context.

## Current Implementation

Phase 2.1 calls a PolicyEngine during DeploymentRun pre-check. The default local runtime uses an allow-all placeholder so the workflow is explicit without pretending a production policy engine exists.

Phase 3.0 adds security policy gates backed by SecurityScan and SecurityFinding records. The local implementation supports noop/fake scanners and minimal built-in rules for critical findings, high finding warnings, mutable artifact tags, digest requirements, and simple manifest risks. Real Trivy, Cosign, SBOM, OPA, Kyverno, Gatekeeper, and enterprise policy integrations remain future work.

## Common Confusion

Policy is not just documentation. A Policy should be evaluated and its result should affect workflow state.
