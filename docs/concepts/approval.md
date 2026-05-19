# Approval

An Approval is a human gate on a delivery subject.

## Concepts

- `ApprovalRequest`: pending request for a release, deployment, or pipeline decision.
- `ApprovalDecision`: approve, reject, or cancel action with approver, comment, and timestamp.
- `ApprovalPolicy`: future policy metadata that can require one or more participants.
- `ApprovalParticipant`: a user or role expected to review a request.

Approvals exist to make human governance auditable. They should not contain secrets, credentials, or external webhook tokens.

## Current Limitations

Phase 3.3 stores approvals in memory for local development and tests. It does not provide production-grade approval workflows, escalation, delegation, or ITSM integration.
