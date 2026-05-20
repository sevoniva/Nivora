# Production-Like Runtime Soak Testing

The soak test harness runs Nivora server, worker, runner, and PostgreSQL for a configurable duration and verifies runtime stability under repeated workload creation.

## Quick Start

```bash
# Short soak (default 60 seconds)
DATABASE_URL="postgres://nivora:nivora@localhost:5432/nivora?sslmode=disable" make soak-runtime-postgres

# Quick smoke (10 seconds, 2 runs)
DATABASE_URL="..." make smoke-soak-runtime

# Overnight mode (6 hours)
DATABASE_URL="..." NIVORA_SOAK_DURATION_SECONDS=21600 make soak-runtime-postgres

# Skip if no PostgreSQL
SKIP_SOAK_RUNTIME=1 make soak-runtime-postgres
```

## Duration Controls

| Variable | Default | Description |
|---|---|---|
| `NIVORA_SOAK_DURATION_SECONDS` | 60 | Total soak duration |
| `NIVORA_SOAK_INTERVAL_SECONDS` | 5 | Time between workload loops |
| `NIVORA_SOAK_RUNS` | 0 | Exact loop count (0 = use duration) |
| `NIVORA_SOAK_RESTART_WORKER` | 1 | Periodically restart worker |
| `NIVORA_SOAK_RESTART_WORKER_EVERY` | 3 | Restart worker every N loops |

## What The Soak Does

1. Starts server, worker, and runner with `runtime_store: postgres`
2. Every loop (every `INTERVAL` seconds):
   - Health-checks the server
   - Creates a shell PipelineRun, verifies it succeeds
   - Fetches logs and timeline for the run
   - Every 3rd loop: creates a DeploymentRun dry-run
   - Every 5th loop: verifies state (audit chain, stuck runs)
   - Periodically restarts the worker to simulate ops events
3. At the end:
   - Creates a final PipelineRun
   - Stops worker and runner
   - Restarts them
   - Verifies the final PipelineRun survived the restart

## Failure Detection

The soak tracks these failure conditions:

| Condition | Counter | Severity |
|---|---|---|
| PipelineRun creation failed | `PIPELINE_FAIL` | High — indicates runtime instability |
| DeploymentRun creation failed | `DEPLOY_FAIL` | Medium — may be expected in noop mode |
| API timeout | `API_TIMEOUTS` | High — server unreachable |
| Heartbeat lost | `HEARTBEAT_LOST` | High — server process may be dead |
| Stuck queued/running runs | `STUCK_RUNS` | Medium — reconciliation may be needed |
| Worker restart failure | `WORKER_FAILURES` | Medium |
| Runner restart failure | `RUNNER_FAILURES` | Low — runner may not be needed for all tests |

Any non-zero failure count causes the soak to exit with code 1.

## What Is Not Covered

- High-concurrency workloads (sequential PipelineRuns only)
- Real Kubernetes/Argo CD deployment targets
- Real cloud provider operations
- Real artifact registry resolution
- Network partition or resource exhaustion scenarios
- Large payload or long-running step scenarios
- Production-scale data volumes

## Interpreting Failures

- **PipelineRun failures**: Check server logs. May indicate executor issues or Postgres connectivity problems.
- **API timeouts**: Server may be overloaded or crashed. Check process logs.
- **Stuck runs**: Worker reconciliation may not be running. Verify worker process.
- **Heartbeat lost**: Server process may have crashed. Check if it needs restart.

## CI Integration

The soak test is NOT part of `make verify` (it requires PostgreSQL and runs for minutes/hours). It is designed for manual or scheduled CI runs.

To add to a scheduled CI job:
```yaml
soak:
  runs-on: ubuntu-latest
  services:
    postgres:
      image: postgres:16-alpine
      ...
  steps:
    - run: |
        DATABASE_URL="postgres://..." NIVORA_SOAK_DURATION_SECONDS=600 make soak-runtime-postgres
```
