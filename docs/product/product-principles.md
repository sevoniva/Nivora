# Product Principles

## Backend Foundation First

Delivery state, APIs, architecture boundaries, and auditability come before frontend surfaces.

## Integrate Rather Than Replace

Nivora should integrate with mature tools through Ports and Adapters instead of replacing every Git provider, Artifact Registry, deployment tool, scanner, or cloud system.

## Immutable Artifacts

Releases should prefer immutable Artifact digests or signed versions. Mutable tags are not enough for audit.

## Auditable Delivery

PipelineRuns, Releases, DeploymentRuns, approvals, policy checks, runner activity, and rollback should produce durable audit context.

## Explicit Approvals and Policies

Approval and Policy are gates in the delivery lifecycle. They should be visible, enforceable, and recorded.

## Runner Isolation

The Execution Plane must be separate from the Control Plane. Runners execute work and report status; the server should not directly execute jobs.

## Open Extension Points

External systems should connect through stable Ports and focused Adapters.

## No Fake Production Readiness

Docs and code must distinguish current implementation from target architecture.

## Incremental Phases

Nivora should evolve through explicit phases. Phase creep makes architecture harder to review and maintain.

