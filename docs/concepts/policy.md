# Policy

A Policy is an enforceable gate in the delivery lifecycle.

## Why It Exists

Policies help decide whether a PipelineRun, Release, or DeploymentRun may proceed. They can represent security checks, environment rules, approval requirements, artifact requirements, or operational constraints.

## Relationships

- Produces PolicyResults.
- May apply before build, before release, before deployment, or during verification.
- Should be recorded in audit context.

## Current Implementation

Phase 2.1 calls a PolicyEngine during DeploymentRun pre-check. The default local runtime uses an allow-all placeholder so the workflow is explicit without pretending a production policy engine exists. Real policy engines, admission checks, and security scanner integrations remain future work.

## Common Confusion

Policy is not just documentation. A Policy should be evaluated and its result should affect workflow state.
