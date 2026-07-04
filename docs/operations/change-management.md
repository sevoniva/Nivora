# Change Management

Phase 6.3 makes governance usable for beta-direction backend validation. It is not a full ITSM workflow engine and Nivora is not GA production-ready.

## Gates

Delivery flows can use three governance gates:

- Policy result: allow, warn, deny, or require approval.
- Change window: environment-scoped timezone/day/time evaluation.
- Approval request: pending, approved, rejected, expired, or canceled.

Release and deployment flows may stop in `WaitingApproval` when approval is required. Approved requests can resume the waiting execution. Rejection and expiration fail the waiting release/deployment; cancellation cancels it. All decisions are auditable and produce local metadata-only notification records.

## Notifications

Notification delivery is behind a `NotificationProvider` port. The default behavior is safe local recording. External providers must be explicitly configured and must use `SecretRef` or `CredentialRef` for sensitive values.

Approval request and terminal decision notifications include approval id, subject, status, and scope metadata. They do not copy approver comments into notification bodies, which keeps the local notification catalog from becoming an extra place for sensitive review text.

The guarded webhook adapter refuses to send unless `AllowSend=true`. Slack, Feishu, DingTalk, email, and ITSM integrations remain future work.

## CLI

```sh
nivora approvals create \
  --subject-type deployment \
  --subject-id drun-prod \
  --env prod \
  --reason "manual production gate" \
  --token-env NIVORA_AUTH_TOKEN
nivora approvals list --token-env NIVORA_AUTH_TOKEN
nivora approvals get <approval-id> --token-env NIVORA_AUTH_TOKEN
nivora approvals approve <id> --comment "approved for current window" --token-env NIVORA_AUTH_TOKEN
nivora approvals reject <id> --comment "policy exception not accepted" --token-env NIVORA_AUTH_TOKEN
nivora approvals cancel <id> --comment "superseded" --token-env NIVORA_AUTH_TOKEN
nivora approvals expire <id> --comment "window expired" --token-env NIVORA_AUTH_TOKEN
nivora deployment resume <deployment-run-id> --approval-status Approved --token-env NIVORA_AUTH_TOKEN
nivora release execution resume <execution-id> --approval-status Approved --token-env NIVORA_AUTH_TOKEN
nivora change-window create --file examples/change-windows/prod-window.yaml --token-env NIVORA_AUTH_TOKEN
nivora change-window list --token-env NIVORA_AUTH_TOKEN
nivora change-window get <change-window-id> --token-env NIVORA_AUTH_TOKEN
nivora change-window evaluate --env prod --at 2026-05-18T02:00:00Z --token-env NIVORA_AUTH_TOKEN
nivora notification list --token-env NIVORA_AUTH_TOKEN
nivora notification test --channel noop --token-env NIVORA_AUTH_TOKEN
```

## Safety

- Do not place webhook secrets in config files or examples.
- Do not log notification payloads that may contain secrets.
- Use placeholders only in examples.
- Local/noop governance is suitable for development and tests, not production approval compliance.
