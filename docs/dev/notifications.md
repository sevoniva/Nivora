# Notifications

Phase 6.3 keeps notifications pluggable and safe by default.

Available foundation adapters:

- `noop`: records the notification without external delivery.
- `log`: writes metadata-only notification records to structured logs.
- `webhook`: posts JSON only when explicitly configured with `AllowSend=true`.

## Test Notification

```sh
nivora notification list
nivora notification test --channel noop
```

Equivalent API:

```sh
curl -s http://localhost:8080/api/v1/notifications
curl -s http://localhost:8080/api/v1/notifications/test \
  -H 'content-type: application/json' \
  -d '{"type":"test","channel":"noop","subject":"Nivora test notification","recipients":["local"]}'
```

## Boundaries

No real email, Slack, Feishu, DingTalk, or webhook delivery is enabled by default. Future external adapters must keep webhook tokens and credentials in SecretRefs or CredentialRefs and must never log secret values.
