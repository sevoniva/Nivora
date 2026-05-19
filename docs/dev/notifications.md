# Notifications

Phase 3.3 adds a `NotificationProvider` port and a noop local provider.

## Test Notification

```sh
nivora notification test --channel noop
```

Equivalent API:

```sh
curl -s http://localhost:8080/api/v1/notifications/test \
  -H 'content-type: application/json' \
  -d '{"type":"test","channel":"noop","subject":"Nivora test notification","recipients":["local"]}'
```

## Boundaries

No real email, Slack, Feishu, DingTalk, or webhook delivery is enabled by default. Future external adapters must keep webhook tokens and credentials in SecretRefs or CredentialRefs and must never log secret values.
