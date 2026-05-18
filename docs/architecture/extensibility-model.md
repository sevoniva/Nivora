# Extensibility Model

Nivora should be extensible without weakening core architecture.

## Adapters

Adapters are the first extension mechanism. They implement Ports for Git providers, Artifact Registries, cloud providers, Executors, secret stores, notification systems, event buses, object stores, and policy engines.

## Built-In vs External Adapters

Small local Adapters can live in the repository. Large vendor-specific or security-sensitive Adapters may eventually live separately if that keeps the core maintainable.

## Plugin Direction

A plugin system is possible in future phases, but it requires an RFC. The project should first stabilize core Ports, data model, runner protocol, and security model.

## What Requires an RFC

- runner protocol changes
- workflow runtime changes
- database model changes
- cloud provider adapters
- Kubernetes or Argo CD integration design
- plugin system changes
- security model changes
- public API breaking changes

## Adding an Adapter Safely

1. Confirm the Port contract.
2. Keep SDK code inside `internal/adapters`.
3. Avoid leaking provider types into domain or usecase packages.
4. Add tests that do not require real external services.
5. Update docs and OpenAPI or AsyncAPI if behavior becomes public.

