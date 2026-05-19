# Approvals

Phase 6.3 hardens the backend-only approval foundation.

## API

```sh
curl -s http://localhost:8080/api/v1/approvals \
  -H 'content-type: application/json' \
  -d '{"subjectType":"deployment","subjectId":"drun-local","environmentId":"prod","requestedBy":"local-user","reason":"production deployment"}'
```

Approve, reject, cancel, or expire:

```sh
curl -s http://localhost:8080/api/v1/approvals/<id>/approve \
  -H 'content-type: application/json' \
  -d '{"approver":"reviewer","comment":"approved for local test"}'
```

DeploymentRun and ReleaseExecution records that are waiting for approval can be resumed or stopped by posting the resulting approval decision:

```sh
curl -s http://localhost:8080/api/v1/deployments/<deployment-run-id>/resume \
  -H 'content-type: application/json' \
  -d '{"subjectType":"deployment","subjectId":"<deployment-run-id>","status":"Approved"}'

curl -s http://localhost:8080/api/v1/releases/executions/<execution-id>/resume \
  -H 'content-type: application/json' \
  -d '{"subjectType":"release","subjectId":"<execution-id>","status":"Rejected"}'
```

## CLI

```sh
nivora approvals list
nivora approvals approve <id> --comment "approved"
nivora approvals reject <id> --comment "not ready"
nivora approvals cancel <id> --comment "superseded"
nivora approvals expire <id> --comment "expired"
nivora deployment resume <deployment-run-id> --approval-status Approved
nivora release execution resume <execution-id> --approval-status Approved
```

## Notes

Approval decisions are audited, and approval events are emitted through the local EventBus. This is not production-ready approval orchestration.
