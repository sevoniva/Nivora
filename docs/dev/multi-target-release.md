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

## Current Limitations

- Sequential execution is the only real execution strategy.
- Parallel execution is future work.
- Approval is a placeholder gate.
- Rollback readiness is target-level only through DeploymentRun rollback plans.
- The runtime is in-memory and not production-ready.

## Safety Rules

- Do not add credentials to examples.
- Do not hardcode local endpoints.
- Do not make apply or Argo CD sync the default.
- Keep cloud, host SSH, Helm, Kustomize, and production GitOps automation out of this phase.
