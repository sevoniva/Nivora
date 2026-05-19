# Extensibility Model

Nivora should be extensible without weakening core architecture.

## Adapters

Adapters are the first extension mechanism. They implement Ports for Git providers, Artifact Registries, cloud providers, Executors, secret stores, notification systems, event buses, object stores, and policy engines.

## Built-In vs External Adapters

Small local Adapters can live in the repository. Large vendor-specific or security-sensitive Adapters may eventually live separately if that keeps the core maintainable.

## Plugin Direction

Phase 4.3 adds a plugin metadata and capability registry foundation. It lists built-in adapter capabilities and defines an external protocol skeleton, but it does not dynamically load code or install external plugins.

Future plugin execution, marketplace behavior, or external adapter runtime changes still require an RFC.

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
