# Runtime Troubleshooting

This page covers common Phase 1 / 1.5 development failures.

## Pipeline Spec Fails Validation

Check for:

- `metadata.name`
- at least one stage
- at least one job per stage
- at least one step per job
- `executor: shell`
- non-empty `run`
- non-negative `retries`
- non-negative `timeoutSeconds`
- duplicate stage, job, or named step entries

## PipelineRun Fails

Inspect logs:

```sh
go run ./cmd/nivora pipeline logs <pipeline-run-id> --server http://localhost:8080 --token-env NIVORA_AUTH_TOKEN
```

Shell commands return a non-zero exit code as a failed JobRun. Timeout failures usually show a deadline error.

## PipelineRun Is Not Found

Local development often still uses the in-memory runtime store. If the server or CLI process restarts in memory mode, previous PipelineRuns are gone. Production-like recovery work should use `database.runtime_store: postgres`; the PostgreSQL-backed runtime stores and recovery checks are covered by the optional Postgres integration profile and CI jobs, but this still does not make Nivora production-ready.

## Runner Is Not Found

The default local runtime registers `local-runner` in memory. If you started a new process, register or heartbeat the runner again:

```sh
go run ./cmd/nivora runner heartbeat --name local-runner --server http://localhost:8080
```

## API Smoke Test Fails On Port

`make smoke-api` uses port `18080` by default. Override it if that port is occupied:

```sh
NIVORA_SMOKE_PORT=18081 make smoke-api
```

## Local Cluster Services Are Unavailable

Phase 1 / 1.5 runtime tests do not require Kubernetes, Argo CD, Harbor, Nexus, GitLab, or Gitea. Optional local environment docs are for future manual validation only.

## Secret Safety

Do not paste secrets into examples, tests, logs, or audit records. Use placeholder values or environment variable names in docs.
