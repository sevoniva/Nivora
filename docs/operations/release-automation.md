# Release Automation

This document captures the release-candidate automation baseline. It keeps the release process explicit while Nivora is still hardening.

## Release Inputs

- `VERSION`
- `CHANGELOG.md`
- `.github/release-template.md`
- `docs/releases/<version>-checklist.md`
- `docs/releases/<version>-release-notes-draft.md` when present
- `api/openapi/openapi.yaml`
- `api/asyncapi/asyncapi.yaml`

## RC Cut Checklist

1. Confirm the working tree is clean.
2. Run `make verify`.
3. Run optional packaging checks that depend on local tooling or registry access:

   ```sh
   make docker-build
   make helm-template
   make helm-lint
   ```

4. Update `VERSION` to the RC version.
5. Move the matching changelog section from draft language to release language.
6. Confirm the release checklist has no open blockers.
7. Commit the release metadata.
8. Tag the release according to maintainer policy.
9. Create the GitHub release using `.github/release-template.md`.

## Guardrails

- Do not include secret values in release notes, changelog entries, examples, or command transcripts.
- Do not describe guarded foundations as production-ready integrations.
- Do not publish release artifacts that were built from a dirty tree.
- Do not skip OpenAPI, AsyncAPI, architecture, secret, example, and smoke checks.

## Current Automation Limits

- Docker image publishing is not automated here.
- SBOM/signature publication is not automated here.
- Helm chart publication is not automated here.
- GitHub release creation remains a maintainer action.
