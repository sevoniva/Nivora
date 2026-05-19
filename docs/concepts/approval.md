# Approval

An Approval is a human gate on a delivery subject.

## Concepts

- `ApprovalRequest`: pending request for a release, deployment, or pipeline decision.
- `ApprovalDecision`: approve, reject, cancel, or expire action with approver, comment, and timestamp.
- `ApprovalPolicy`: policy metadata that can require participants by environment, target, severity, or policy result.
- `ApprovalParticipant`: a user or role expected to review a request.

Approvals exist to make human governance auditable. They should not contain secrets, credentials, or external webhook tokens.

## Current Limitations

Phase 6.3 keeps approvals suitable for local/backend validation. It does not provide production-grade approval workflows, escalation, delegation, frontend review queues, or ITSM integration.
