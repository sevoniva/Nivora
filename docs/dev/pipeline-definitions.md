# Pipeline Definition Catalog

Nivora can store Pipeline definitions in the control-plane catalog and start a PipelineRun from the saved definition. This is a backend foundation feature, not a complete scheduler.

## Create A Definition

```bash
go run ./cmd/nivora pipeline definition create \
  --server http://localhost:8080 \
  --project-id project-a \
  --file examples/pipelines/simple-shell.yaml
```

In token auth mode, pass a token through an environment variable:

```bash
go run ./cmd/nivora pipeline definition list --server http://localhost:8080 --token-env NIVORA_AUTH_TOKEN
```

Do not put token values in command examples or committed files.

## Run A Saved Definition

```bash
go run ./cmd/nivora pipeline definition run <pipeline-id> --server http://localhost:8080 --token-env NIVORA_AUTH_TOKEN
```

The created PipelineRun records the catalog `pipelineId` and `pipelineVersionId` so later logs, events, audit entries, and timelines can be traced back to the saved definition.

To rerun a saved historical version, pass the version number:

```bash
go run ./cmd/nivora pipeline definition run <pipeline-id> --version 1 --server http://localhost:8080 --token-env NIVORA_AUTH_TOKEN
```

To attach the created PipelineRun to an environment, pass `--environment-id`:

```bash
go run ./cmd/nivora pipeline definition run <pipeline-id> --environment-id env-prod --server http://localhost:8080 --token-env NIVORA_AUTH_TOKEN
```

The same environment flag works when posting a local Pipeline YAML to the server:

```bash
go run ./cmd/nivora pipeline run --local=false --environment-id env-prod --server http://localhost:8080 --token-env NIVORA_AUTH_TOKEN examples/pipelines/simple-shell.yaml
```

Environment ownership is stored as PipelineRun metadata and labels. Environment-scoped runner tokens can only claim queued PipelineRuns with the matching `environmentId`; this is a control-plane metadata guardrail, not a runtime sandbox.

The shorthand command also supports catalog IDs when `--local=false`:

```bash
go run ./cmd/nivora pipeline run --local=false --server http://localhost:8080 --token-env NIVORA_AUTH_TOKEN <pipeline-id>
```

If the argument is an existing local file path, the command keeps the older server behavior and posts that YAML directly to `/api/v1/pipeline-runs`.

## Versions

```bash
go run ./cmd/nivora pipeline definition versions <pipeline-id> --server http://localhost:8080 --token-env NIVORA_AUTH_TOKEN
```

The endpoint returns persisted version metadata for versions created through catalog create/update operations. Each entry includes the version number, definition hash, and timestamps. `historyComplete` is true for the configured catalog store when it can enumerate the saved history.

This is a traceability and rerun feature, not a rollback or scheduling mechanism.

## Safety

- Disabled Pipeline definitions cannot be run.
- Project-scoped tokens can only list, read, update, disable, and run definitions in their own project scope.
- Running a saved definition does not change executor safety defaults. The shell executor remains a local foundation and is not a sandbox.
- PostgreSQL persistence is used only when `database.runtime_store: postgres` is configured.
