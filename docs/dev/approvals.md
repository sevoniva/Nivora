# Approvals

Phase 6.3 hardens the backend-only approval foundation.

## API

```sh
curl -s http://localhost:8080/api/v1/approvals \
  -H "Authorization: Bearer ${NIVORA_AUTH_TOKEN}" \
  -H 'content-type: application/json' \
  -d '{"subjectType":"deployment","subjectId":"drun-local","environmentId":"prod","requestedBy":"local-user","reason":"production deployment"}'
```

Approve, reject, cancel, or expire:

```sh
curl -s http://localhost:8080/api/v1/approvals/<id>/approve \
  -H "Authorization: Bearer ${NIVORA_AUTH_TOKEN}" \
  -H 'content-type: application/json' \
  -d '{"approver":"reviewer","comment":"approved for local test"}'
```

DeploymentRun and ReleaseExecution records that are waiting for approval can be resumed or stopped by posting the resulting approval decision:

```sh
curl -s http://localhost:8080/api/v1/deployments/<deployment-run-id>/resume \
  -H "Authorization: Bearer ${NIVORA_AUTH_TOKEN}" \
  -H 'content-type: application/json' \
  -d '{"subjectType":"deployment","subjectId":"<deployment-run-id>","status":"Approved"}'

curl -s http://localhost:8080/api/v1/releases/executions/<execution-id>/resume \
  -H "Authorization: Bearer ${NIVORA_AUTH_TOKEN}" \
  -H 'content-type: application/json' \
  -d '{"subjectType":"release","subjectId":"<execution-id>","status":"Rejected"}'
```

The approval service also exposes a subject resume helper. It reads the stored approval request by approval id and applies the terminal decision to the referenced DeploymentRun, ReleaseExecution, or PipelineRun:

```sh
curl -s -X POST http://localhost:8080/api/v1/approvals/<id>/resume-subject \
  -H "Authorization: Bearer ${NIVORA_AUTH_TOKEN}"
```

This helper rejects Pending approvals. For PipelineRun subjects, approved decisions move a `Paused` PipelineRun back to `Queued`; rejected or expired decisions fail the run; canceled decisions cancel it. The helper does not execute shell steps directly.

## CLI

```sh
nivora approvals create \
  --subject-type deployment \
  --subject-id drun-local \
  --env prod \
  --requested-by local-user \
  --reason "production deployment" \
  --token-env NIVORA_AUTH_TOKEN
nivora approvals list --token-env NIVORA_AUTH_TOKEN
nivora approvals get <id> --token-env NIVORA_AUTH_TOKEN
nivora approvals approve <id> --comment "approved" --token-env NIVORA_AUTH_TOKEN
nivora approvals reject <id> --comment "not ready" --token-env NIVORA_AUTH_TOKEN
nivora approvals cancel <id> --comment "superseded" --token-env NIVORA_AUTH_TOKEN
nivora approvals expire <id> --comment "expired" --token-env NIVORA_AUTH_TOKEN
nivora approvals resume <id> --token-env NIVORA_AUTH_TOKEN
nivora deployment resume <deployment-run-id> --approval-status Approved --token-env NIVORA_AUTH_TOKEN
nivora release execution resume <execution-id> --approval-status Approved --token-env NIVORA_AUTH_TOKEN
```

## Notes

Approval requests and terminal decisions are audited, emit local events, and create metadata-only notification records through the configured notification provider. Decision notifications do not copy the approver comment into the notification body. This is not production-ready approval orchestration.
