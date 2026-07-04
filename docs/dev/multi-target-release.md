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

## Cancellation

Server-backed ReleaseExecution records can be canceled explicitly:

```sh
go run ./cmd/nivora release execution cancel <execution-id>
go run ./cmd/nivora release cancel <release-id>
```

Canceling a ReleaseExecution marks the execution `Canceled`, marks non-terminal target executions `Canceled`, and asks linked non-terminal DeploymentRuns to cancel through the DeploymentRun service. Canceling a Release also cascades to non-terminal ReleaseExecutions for that Release. These paths do not run rollback, delete resources, or mutate already-terminal execution records.

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
