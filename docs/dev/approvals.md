# Approvals

Phase 3.3 adds a backend-only approval foundation.

## API

```sh
curl -s http://localhost:8080/api/v1/approvals \
  -H 'content-type: application/json' \
  -d '{"subjectType":"deployment","subjectId":"drun-local","environmentId":"prod","requestedBy":"local-user","reason":"production deployment"}'
```

Approve or reject:

```sh
curl -s http://localhost:8080/api/v1/approvals/<id>/approve \
  -H 'content-type: application/json' \
  -d '{"approver":"reviewer","comment":"approved for local test"}'
```

## CLI

```sh
nivora approvals list
nivora approvals approve <id> --comment "approved"
nivora approvals reject <id> --comment "not ready"
```

## Notes

Approval decisions are audited, and approval events are emitted through the local EventBus. This is not production-ready approval orchestration.
