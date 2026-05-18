# Module Boundaries

## Golden Rule

Domain owns meaning.
Use cases own orchestration.
Ports own contracts.
Adapters own external systems.
Infrastructure owns technical plumbing.
API owns transport.

## Forbidden Imports

`internal/domain` must not import:

- `internal/api`
- `internal/app`
- `internal/adapters`
- `internal/infra`
- database drivers
- HTTP routers
- cloud SDKs
- Kubernetes SDKs

`internal/usecase` must not import:

- `internal/api`
- concrete packages under `internal/adapters`
- cloud SDKs directly
- Kubernetes SDKs directly

`internal/api` must not contain business orchestration logic.

`internal/adapters` must not define domain concepts.

## Package Naming

Use singular domain package names when possible:

- `domain/project`
- `domain/pipeline`
- `domain/deployment`
- `domain/runner`

Avoid vague packages:

- `common`
- `utils`
- `helper`
- `manager`
- `service`

A package named `utils` is not allowed unless explicitly justified.

## Dependency Direction

Preferred direction:

```text
cmd -> app -> api -> usecase -> domain
                 usecase -> ports
                 adapters -> ports
                 infra -> ports where appropriate
```

Do not invert this direction to make implementation easier.
