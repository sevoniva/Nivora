# API Contract Checks

Nivora keeps the HTTP router and OpenAPI document aligned through automated tests, not manual review alone.

The primary implementation is `internal/api/http/routes/openapi_contract_test.go`.

## What Is Checked

- registered routes are present in `api/openapi/openapi.yaml`
- OpenAPI paths map back to registered routes
- placeholder routes are documented as placeholder or not implemented
- placeholder routes return structured `not_implemented` responses
- implemented routes are not documented as placeholder routes
- mutation routes are checked for security documentation
- request body and error response coverage are audited

Run:

```bash
go test ./internal/api/http/routes -run TestOpenAPI
make verify-contracts
make verify
```

`make verify` runs these checks in CI. The checks do not contact Kubernetes, Argo CD, cloud providers, registries, scanners, or external systems.

## Current Limits

OpenAPI still does not define exhaustive request and response schemas for every domain object. The route contract check prevents route drift; schema-level hardening remains a separate API quality task.

See also: `docs/dev/api-contracts.md`.
