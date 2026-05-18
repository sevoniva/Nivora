# Testing Policy

## Minimum Local Verification

For normal backend changes:

```bash
gofmt -w .
go test ./...
go build ./cmd/nivora-server
go build ./cmd/nivora-worker
go build ./cmd/nivora-runner
go build ./cmd/nivora
```

## Unit Tests

Use unit tests for:

- domain statuses
- config loading
- route behavior
- executor behavior
- event envelope validation
- usecase state transitions

## Integration Tests

Use integration tests for:

- database migrations
- repository persistence
- event bus behavior
- object store behavior
- runner-server interaction

Do not make tests depend on real cloud services.

## E2E Tests

E2E tests should run locally with docker-compose.

Do not require AWS, Aliyun, Tencent Cloud, Kubernetes, Argo CD, Harbor, or GitLab for baseline CI.

## Test Data

Test data must not contain real credentials or realistic-looking fake credentials.
