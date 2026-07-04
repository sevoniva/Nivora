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

Runtime state is in memory. If the server or CLI process restarts, previous PipelineRuns are gone. Durable runtime repositories are future work.

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
