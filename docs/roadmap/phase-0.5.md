# Phase 0.5: Guardrails and Validation

## Objective

Harden the repository for future contributors and AI coding agents.

## Scope

- AI coding rules.
- Architecture verification.
- Secret checks.
- CI hardening.
- Makefile verification targets.
- GOPROXY configurability.
- Structured placeholder responses.

## Non-Goals

- Business logic.
- Real external integrations.
- Frontend work.

## Expected Deliverables

Guardrails that make unsafe architecture drift easier to catch early.

## Acceptance Criteria

- CI runs formatting, tidy, vet, tests, builds, architecture checks, and secret checks.
- `make verify` runs important local checks.
- `AGENTS.md` remains canonical.

## Contribution Opportunities

- Improve conservative scripts.
- Improve docs around contribution flow.
- Add focused tests for existing behavior.

