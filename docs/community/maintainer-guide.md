# Maintainer Guide

Maintainers should review for correctness, scope, and architecture fit.

## Review Checklist

- Architecture boundaries are respected.
- Domain does not import infrastructure, adapters, HTTP, database, cloud, or Kubernetes packages.
- Use cases do not import concrete Adapters.
- Tests cover changed behavior.
- Docs are updated when concepts, APIs, phases, or architecture change.
- OpenAPI and AsyncAPI stay consistent with public behavior.
- No secrets or realistic fake credentials are introduced.
- Logs and errors do not leak sensitive values.
- No fake production readiness claims.
- No phase creep.
- No unrelated refactor.
- No frontend code unless explicitly in scope.

## Merge Discipline

Prefer focused PRs. Ask contributors to split broad changes. Require RFCs for large architecture or integration changes.

