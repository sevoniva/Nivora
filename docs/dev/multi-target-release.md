# Multi-Target Release Development

Phase 2.7 introduces local release orchestration for contributors.

## Local Planning

```sh
make release-plan-local
```

or:

```sh
go run ./cmd/nivora release plan --file examples/releases/multi-target-release.yaml --local
```

This creates a ReleasePlan from an inline Release and target definitions. It does not require Kubernetes, Argo CD, Git providers, registries, or cloud services.

## Server-backed Release ID Planning

After a Release has been created on a Nivora server, contributors can plan a safe noop/webhook orchestration without writing a separate orchestration file:

```sh
go run ./cmd/nivora release plan <release-id> --environment dev --target audit-only --local=false
```

Release ID mode is server-backed because the CLI local process does not retain previously saved Release records. Without `--file`, only `noop` and `webhook` targets are accepted; Kubernetes, Argo CD, and host targets still require an orchestration file with a full Deployment spec and the existing guarded execution flags.

## Local Execution

```sh
make release-deploy-local
```

or:

```sh
go run ./cmd/nivora release deploy --file examples/releases/sequential-release.yaml --local
```

The local runtime executes safe targets only:

- `kubernetes-yaml` uses the existing noop manifest client for dry-run behavior.
- `argocd` uses the existing noop Argo CD provider.
- `noop` records target execution without external side effects.

ReleaseTarget metadata can also be managed through the catalog API and CLI:

```sh
go run ./cmd/nivora target create --environment-id env-dev --name dev-noop --type noop
go run ./cmd/nivora target list --project-id project-dev
go run ./cmd/nivora target validate target-id
```

These commands manage metadata only. They do not run apply, sync, remote host deployment, rollback, or Git push.

For a saved server-side Release and safe noop/webhook targets:

```sh
go run ./cmd/nivora release deploy <release-id> --environment dev --target audit-only --local=false
```

This creates a ReleaseExecution through the server API. It does not enable Kubernetes apply, Argo CD sync, remote host deployment, Git push, rollback execution, or any external provider integration.

## Cancellation

Server-backed ReleaseExecution records can be canceled explicitly:

```sh
go run ./cmd/nivora release execution cancel <execution-id>
go run ./cmd/nivora release cancel <release-id>
```

Canceling a ReleaseExecution marks the execution `Canceled`, marks non-terminal target executions `Canceled`, and asks linked non-terminal DeploymentRuns to cancel through the DeploymentRun service. Canceling a Release also cascades to non-terminal ReleaseExecutions for that Release. These paths do not run rollback, delete resources, or mutate already-terminal execution records.

## Release Status Lifecycle

Release orchestration now updates the bound Release record as the plan or execution advances:

- created releases start as `Ready` after artifact binding succeeds
- planning records `Planning`
- approval gates record `WaitingApproval`
- active execution records `Deploying`
- successful execution records `Succeeded`
- failed or partially successful target execution records `Failed`
- explicit cancellation records `Canceled`

Each status change emits `devops.release.status.updated` and records release audit evidence. This is still a synchronous foundation state model, not a production workflow engine; it does not perform rollback, delete resources, or recover and mutate stale executions automatically.

## Current Limitations

- Sequential execution is the only real execution strategy.
- Parallel execution is future work.
- Approval is a foundation gate.
- Rollback readiness is target-level only through DeploymentRun rollback plans.
- Release target metadata is persisted when the server is configured with `database.runtime_store: postgres`. Memory mode remains available for local development, and target execution still depends on the DeploymentRun and ReleaseExecution foundations.

## Safety Rules

- Do not add credentials to examples.
- Do not hardcode local endpoints.
- Do not make apply or Argo CD sync the default.
- Keep cloud, host SSH, Helm, Kustomize, and production GitOps automation out of this phase.
