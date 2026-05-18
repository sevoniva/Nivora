# Contributing

Thanks for helping build Nivora.

## Development Setup

Install Go 1.22 or newer and Docker. The default local tooling uses:

```sh
GOPROXY=https://proxy.golang.org,direct
```

Contributors in China can override it per command:

```sh
GOPROXY=https://goproxy.cn,direct make verify
```

Or export it for the shell:

```sh
export GOPROXY=https://goproxy.cn,direct
```

Then run:

```sh
make verify
```

Local dependencies can be started with:

```sh
make dev-up
```

## Code Conventions

- Keep Go code simple and explicit.
- Use `slog` for logging.
- Keep domain packages free of HTTP, database, queue, cloud, Kubernetes, and vendor SDK dependencies.
- Put interfaces near the usecase boundary under `internal/ports`.
- Put infrastructure details under `internal/adapters` or `internal/infra`.
- Avoid broad refactors in focused changes.

## Testing Expectations

- Add focused tests for changed behavior.
- Do not make network calls in tests.
- Prefer Go's standard `testing` package.
- Keep shell execution tests controlled and time-bounded.

## Architecture Rules

- Control plane and runner remain separate.
- Phase 0 does not include frontend work.
- GitOps is one deployment mode, not the only deployment mode.
- Adapters depend on ports and domain concepts; domain does not depend on adapters.
- New production integrations should start as ports plus small adapter packages.

## Adding a New Adapter

1. Find or create the relevant interface under `internal/ports`.
2. Add the adapter under `internal/adapters/<kind>/<provider>`.
3. Keep provider SDK details inside the adapter.
4. Add configuration only when the adapter is wired into an app.
5. Add tests that do not require external services.
