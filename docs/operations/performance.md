# Performance, Scale, and Load Testing

Phase 8.4 adds a measurement-first performance foundation for Nivora. It does not claim production scale limits yet; it gives maintainers repeatable tools to measure current behavior before optimizing.

## Baseline Benchmarks

Run local benchmarks without external services:

```bash
make benchmark
```

The benchmark suite currently covers:

- PipelineRun creation in the in-memory runtime
- LogChunk append throughput
- Pipeline timeline query cost
- runner heartbeat processing
- job claim processing
- DeploymentRun resource inventory planning

Use `go test -bench=. -benchmem ./internal/usecase/pipeline ./internal/usecase/deployment` when you need raw Go benchmark controls.

## Safe Load Scripts

The load scripts expect a running local server and use only local HTTP APIs:

```bash
make run-server
NIVORA_LOAD_RUNS=50 make load-generate-runs
NIVORA_LOAD_RUNS=20 NIVORA_LOAD_LOG_BYTES=4096 make load-generate-logs
NIVORA_LOAD_RUNNERS=10 NIVORA_LOAD_HEARTBEATS=100 make load-simulate-runners
```

Environment variables:

- `NIVORA_URL`: server URL, default `http://127.0.0.1:8080`
- `NIVORA_LOAD_RUNS`: number of PipelineRuns to create
- `NIVORA_LOAD_LOG_BYTES`: bytes emitted by each log-generating run, capped at the API log chunk limit
- `NIVORA_LOAD_RUNNERS`: runners to register
- `NIVORA_LOAD_HEARTBEATS`: heartbeat requests to send

These scripts do not require Kubernetes, cloud services, registries, or external credentials.

## Pagination

List-like runtime endpoints accept optional `limit` and `offset` query parameters. When neither parameter is supplied, endpoints keep the legacy array response for compatibility. When either parameter is supplied, the response shape is:

```json
{
  "items": [],
  "pagination": {
    "limit": 100,
    "offset": 0,
    "total": 0
  }
}
```

Covered endpoints:

- `GET /api/v1/pipeline-runs`
- `GET /api/v1/pipeline-runs/{id}/logs`
- `GET /api/v1/pipeline-runs/{id}/events`
- `GET /api/v1/pipeline-runs/{id}/timeline`
- `GET /api/v1/deployments`
- `GET /api/v1/deployments/{id}/logs`
- `GET /api/v1/deployments/{id}/events`
- `GET /api/v1/deployments/{id}/timeline`
- `GET /api/v1/audit/search`

Limits:

- default page size: `100`
- maximum page size: `500`
- offset must be zero or greater

## API Limits

Current defensive limits:

- maximum HTTP request body: `4 MiB`
- maximum runner log chunk content: `64 KiB`
- maximum static manifest file size: `1 MiB`

These limits are intentionally conservative for the alpha/beta foundation. Increase them only after measuring memory, latency, and storage impact.

## Database Index Review

Migration `000006_performance_indexes` adds indexes for common read paths:

- run status plus creation time
- log chunk run plus sequence
- event subject plus creation time
- audit subject/actor plus creation time
- runner status plus heartbeat time
- runtime job runner/status lookup

The existing runtime repository already has indexes for queued/running runs, job leases, log ordering, event outbox retries, and runner heartbeat scans. Future database work should use `EXPLAIN (ANALYZE, BUFFERS)` against realistic row counts before adding more indexes.

## Measurement Guidance

Before optimizing, capture:

- benchmark output with `-benchmem`
- API latency from the load scripts
- `/metrics` output before and after the run
- database query plans for slow SQL
- CPU and memory profile if a benchmark regresses

Avoid adding queues, caches, or new infrastructure until the measurements show a specific bottleneck.

## Current Limitations

- Benchmarks mostly exercise in-memory runtime paths.
- Load scripts are local developer tools, not a distributed load test platform.
- PostgreSQL performance validation still needs a realistic integration environment.
- Metrics are process-local unless connected to external scraping/storage.
- Nivora remains a hardened beta-candidate foundation and is not production-ready.
