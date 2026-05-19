# Developer Guide

This guide is for contributors working on Nivora internals. Follow `AGENTS.md`, architecture guardrails, and phase boundaries before changing code.

## Architecture

Read these first:

- [Architecture contract](../architecture/architecture-contract.md)
- [Module boundaries](../architecture/module-boundaries.md)
- [Control plane](../architecture/control-plane.md)
- [Workflow model](../architecture/workflow-model.md)
- [Data model](../architecture/data-model.md)
- [Security model](../architecture/security-model.md)

Rules of thumb:

- Domain packages must not import API, infra, adapters, HTTP routers, DB drivers, Kubernetes SDKs, cloud SDKs, Argo CD clients, Git providers, or registry clients.
- Use cases orchestrate behavior through ports and repositories.
- Adapters implement ports.
- API handlers call use cases.
- Runner and Executor remain separate concepts.

## Adapters

Adapter docs:

- [Integration model](../architecture/integration-model.md)
- [Executor design](../architecture/executor-design.md)
- [Writing adapters](../dev/writing-adapters.md)
- [Artifact model](../architecture/artifact-model.md)
- [GitOps model](../architecture/gitops-model.md)
- [Cloud provider model](../architecture/cloud-provider-model.md)
- [Secret model](../architecture/secret-model.md)

Adapters should keep vendor/client dependencies out of domain and usecase packages.

## Plugins

Plugin docs:

- [Plugin system](../architecture/plugin-system.md)
- [Plugin authoring](../dev/plugin-authoring.md)
- [Plugin RFC](../rfcs/plugin-api.md)
- [Plugin template](../rfcs/plugin-template.md)

Do not use unsafe Go plugin loading. The current model is metadata, capability registry, templates, and external protocol foundation.

## Tests

Testing docs:

- [Testing strategy](../dev/testing-strategy.md)
- [Acceptance tests](../dev/acceptance-tests.md)
- [Quality dashboard](../dev/quality-dashboard.md)

Baseline commands:

```sh
go test ./...
go vet ./...
./scripts/verify-architecture.sh
./scripts/verify-no-secrets.sh
./scripts/validate-examples.sh
make verify
```

Optional checks:

```sh
make test-race
make benchmark
```

Baseline tests must not require Kubernetes, cloud services, registries, Argo CD, Vault, external scanners, or external notification systems.

## Release Work

Release docs:

- [Release playbook](../releases/release-playbook.md)
- [Release automation](../operations/release-automation.md)
- [v1.0.0-rc.1 checklist](../releases/v1.0.0-rc.1-checklist.md)
- [Security review checklist](../security/security-review-checklist.md)

Do not update `VERSION`, tag, or publish artifacts unless the release cut is explicitly in scope.
