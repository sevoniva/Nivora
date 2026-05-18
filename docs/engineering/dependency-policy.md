# Dependency Policy

Nivora should remain lightweight and maintainable.

## Default Position

Do not add a dependency unless it clearly improves correctness, maintainability, or interoperability.

## Allowed Early Dependencies

- `go-chi/chi` for HTTP routing
- `spf13/cobra` for CLI
- `spf13/viper` for config if needed
- `pgx` for PostgreSQL
- `goose` or `golang-migrate` for migrations
- `testcontainers-go` only for integration tests when needed

## Discouraged Early Dependencies

- heavy web frameworks
- large dependency injection frameworks
- reflection-heavy validation frameworks
- cloud provider SDKs before actual adapter implementation
- Kubernetes client packages before Kubernetes executor implementation
- ORMs before persistence strategy is settled
- broad utility libraries for trivial helpers

## New Dependency Checklist

Before adding a dependency, document:

- why the standard library is insufficient
- why existing dependencies are insufficient
- expected scope of usage
- whether it affects runtime binary size
- whether it is actively maintained
