# Change Management

Phase 6.3 makes governance usable for beta-direction backend validation. It is not a full ITSM workflow engine and Nivora is not GA production-ready.

## Gates

Delivery flows can use three governance gates:

- Policy result: allow, warn, deny, or require approval.
- Change window: environment-scoped timezone/day/time evaluation.
- Approval request: pending, approved, rejected, expired, or canceled.

Release and deployment flows may stop in `WaitingApproval` when approval is required. Approved requests can resume the waiting execution. Rejection and expiration fail the waiting release/deployment; cancellation cancels it. All decisions are auditable.

## Notifications

Notification delivery is behind a `NotificationProvider` port. The default behavior is safe local recording. External providers must be explicitly configured and must use `SecretRef` or `CredentialRef` for sensitive values.

The guarded webhook adapter refuses to send unless `AllowSend=true`. Slack, Feishu, DingTalk, email, and ITSM integrations remain future work.

## CLI

```sh
nivora approvals create \
  --subject-type deployment \
  --subject-id drun-prod \
  --env prod \
  --reason "manual production gate"
nivora approvals list
nivora approvals get <approval-id>
nivora approvals approve <id> --comment "approved for current window"
nivora approvals reject <id> --comment "policy exception not accepted"
nivora approvals cancel <id> --comment "superseded"
nivora approvals expire <id> --comment "window expired"
nivora deployment resume <deployment-run-id> --approval-status Approved
nivora release execution resume <execution-id> --approval-status Approved
nivora change-window create --file examples/change-windows/prod-window.yaml
nivora change-window list
nivora change-window get <change-window-id>
nivora change-window evaluate --env prod --at 2026-05-18T02:00:00Z
nivora notification list
nivora notification test --channel noop
```

## Safety

- Do not place webhook secrets in config files or examples.
- Do not log notification payloads that may contain secrets.
- Use placeholders only in examples.
- Local/noop governance is suitable for development and tests, not production approval compliance.
