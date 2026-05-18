# Decision Making

## Issues First When Useful

Use issues to discuss problems, scope, and expected behavior before implementation. Small typo fixes and obvious documentation corrections can go directly to pull request.

## Small Pull Requests

Prefer small, focused pull requests. Avoid unrelated refactors. Keep changes aligned with the current phase.

## ADRs

Use ADRs to record accepted architecture decisions such as language, monolith strategy, event model, storage direction, or runner separation.

## RFCs

Use RFCs for large or uncertain proposals. An RFC should explain problem, goals, non-goals, architecture impact, API impact, data model impact, security impact, observability impact, rollout, and open questions.

## Maintainer Review

Maintainers review architecture boundaries, tests, docs, API consistency, security impact, and phase scope.

## When to Reject Changes

Changes may be rejected when they:

- violate architecture boundaries
- add real integrations before design review
- introduce secrets or realistic fake credentials
- claim production readiness
- add frontend work outside the approved phase
- refactor unrelated code
- weaken guardrails

