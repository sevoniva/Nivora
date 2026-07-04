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
NIVORA_AUTH_TOKEN=... go run ./cmd/nivora pipeline definition list --server http://localhost:8080
```

Do not put token values in command examples or committed files.

## Run A Saved Definition

```bash
go run ./cmd/nivora pipeline definition run <pipeline-id> --server http://localhost:8080
```

The created PipelineRun records the catalog `pipelineId` and `pipelineVersionId` so later logs, events, audit entries, and timelines can be traced back to the saved definition.

The shorthand command also supports catalog IDs when `--local=false`:

```bash
go run ./cmd/nivora pipeline run --local=false --server http://localhost:8080 <pipeline-id>
```

If the argument is an existing local file path, the command keeps the older server behavior and posts that YAML directly to `/api/v1/pipeline-runs`.

## Versions

```bash
go run ./cmd/nivora pipeline definition versions <pipeline-id> --server http://localhost:8080
```

The endpoint returns the current saved version and hash. Full historical version browsing is still foundation-level; the response includes `historyComplete: false` until complete history storage is implemented.

## Safety

- Disabled Pipeline definitions cannot be run.
- Project-scoped tokens can only list, read, update, disable, and run definitions in their own project scope.
- Running a saved definition does not change executor safety defaults. The shell executor remains a local foundation and is not a sandbox.
- PostgreSQL persistence is used only when `database.runtime_store: postgres` is configured.
