# API Contracts

Nivora API contracts are verified through automated tests in `internal/api/http/routes/openapi_contract_test.go`. This document explains what is checked and what contract gaps remain.

## OpenAPI Verification

The OpenAPI spec (`api/openapi/openapi.yaml`) is validated against the registered HTTP router at test time:

| Check | Test Function | Status |
|---|---|---|
| Route-to-OpenAPI path/method coverage | `TestOpenAPIPathsMatchRegisteredRoutes` | ✅ Bidirectional |
| Placeholder route documentation | `TestOpenAPIPlaceholderRouteLabelsMatchRouter` | ✅ 13 groups verified |
| Placeholder route response format | `TestAllPlaceholderRoutesReturnStructuredNotImplemented` | ✅ Structured 501 |
| Error response schema consistency | `TestOpenAPIErrorResponseSchemaConsistency` | ✅ Audit-level check |
| Mutation route security schemes | `TestOpenAPIMutationRoutesHaveSecurity` | ✅ 64/64 protected |
| Request body schema presence | `TestOpenAPIRequestBodySchemaPresence` | ✅ Audit-level check |
| Route duplicate detection | `TestRouteDuplicateDetection` | ✅ |

## AsyncAPI Verification

The AsyncAPI spec (`api/asyncapi/asyncapi.yaml`) is validated for event documentation consistency:

| Check | Test Function | Status |
|---|---|---|
| File structure validation | `TestAsyncAPIEventDocumentation` | ✅ Channels/messages present |
| Key event type coverage | `TestAsyncAPIEventDocumentation` | ✅ 18 key events checked, including release cancel and release status updates |
| Future/reserved labeling | `TestAsyncAPIEventDocumentation` | ✅ Warns if missing |

## Security Scheme Coverage

All mutation routes (POST/PUT/PATCH/DELETE) are wrapped with `RequirePermission` middleware. The OpenAPI security schemes document `BearerAuth` and `RunnerTokenAuth`. Route-level permission requirements are documented in `docs/security/ROUTE_PERMISSION_MATRIX.md`.

### Public Routes (no auth required)
- `GET /healthz`, `GET /readyz`, `GET /metrics`
- `POST /api/v1/artifact-registries/validate`
- Runner protocol: heartbeat, claim, log append, status update (use `X-Nivora-Runner-Token`)

## Aggregate Runtime Queries

The control plane exposes read-only aggregate observability routes:

- `GET /api/v1/events`
- `GET /api/v1/logs`
- `GET /api/v1/timeline`
- `GET /api/v1/audit-logs`

`/api/v1/timeline` combines filtered runtime events and log summaries into one time-ordered response. It accepts the same lightweight filters used by events/logs, including `runId`, `pipelineRunId`, `deploymentRunId`, `releaseId`, `artifactId`, `securityScanId`, `jobRunId`, `stepRunId`, `contains`, `limit`, and `offset`. Secret-like values are redacted before they are returned.

## Schema Coverage

Key domain schemas are defined in OpenAPI `components/schemas`:

| Schema | Purpose |
|---|---|
| `ErrorResponse` | Standard error format (code, message, path, request_id) |
| `Version` | Server version info |
| `Pagination` / `PaginatedResponse` | List endpoint pagination |
| `ApprovalCreateRequest` | Approval request body |

Additional schemas for PipelineRun, DeploymentRun, AuditRecord, and other domain types are described structurally through path-level response examples rather than explicit schema references. Schema-level OpenAPI hardening is future work.

## Running Contract Tests

```bash
make verify-contracts
# or directly:
go test -v -run "TestOpenAPI|TestAsyncAPI|TestRouteDuplicate" ./internal/api/http/routes/
```

Contract tests are part of `make verify` and run in CI.

## Current Limitations

- Per-operation request/response schemas are not exhaustively defined in OpenAPI (response examples are present).
- Schema references (`$ref`) are used for ErrorResponse and Pagination but not for all domain types.
- AsyncAPI event types are not mechanically cross-referenced against emitted event constants from Go code.
- Security scheme coverage is verified at route level, not at OpenAPI operation level.
- OpenAPI `security` blocks use top-level `BearerAuth` for most operations rather than per-operation fine-grained permission documentation.
