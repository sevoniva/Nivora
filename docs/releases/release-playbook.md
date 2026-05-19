# Release Playbook

This playbook is for maintainers preparing Nivora releases. It complements the version-specific checklist under `docs/releases/` and the automation notes in `docs/operations/release-automation.md`.

## Release Principles

- Do not hide limitations.
- Do not claim production readiness unless the matching checklist explicitly passes.
- Do not publish from a dirty working tree.
- Do not include secrets, real credentials, kubeconfigs, private keys, tokens, or realistic fake credentials.
- Keep guarded operations explicit: apply, sync, push, prune, remote host deploy, and destructive rollback.

## Before The Release Cut

1. Confirm scope and target version.
2. Review `README.md`, `CHANGELOG.md`, `VERSION`, OpenAPI, AsyncAPI, examples, docs, Dockerfile, Helm chart, Makefile, and scripts.
3. Review the matching checklist:
   - `docs/releases/v0.5.0-beta-checklist.md`
   - `docs/releases/v1.0.0-rc.1-checklist.md`
   - `docs/releases/v1.0.0-ga-checklist.md`
4. Review [Security Review Checklist](../security/security-review-checklist.md).
5. Run the baseline verification commands:

```sh
go mod tidy
go test ./...
go vet ./...
go build ./cmd/nivora-server
go build ./cmd/nivora-worker
go build ./cmd/nivora-runner
go build ./cmd/nivora
./scripts/verify-architecture.sh
./scripts/verify-no-secrets.sh
./scripts/verify-api-specs.sh
./scripts/validate-examples.sh
make verify
git diff --check
```

6. Run optional packaging checks when tooling and network access are available:

```sh
make docker-build
make helm-template
make helm-lint
```

## Version And Changelog

For a release cut only:

1. Update `VERSION`.
2. Move the relevant `CHANGELOG.md` section out of draft language.
3. Confirm release notes match the actual capability matrix and known limitations.
4. Confirm `.github/release-template.md` has no stale version-specific text.

Do not update `VERSION` during ordinary hardening work unless the user explicitly asks to cut the release.

## API And Event Review

1. Confirm implemented public routes are reflected in `api/openapi/openapi.yaml`.
2. Confirm emitted public events are reflected in `api/asyncapi/asyncapi.yaml`.
3. Confirm placeholder APIs return structured `not_implemented` responses.
4. Confirm no API returns secret values, token hashes, kubeconfigs, private keys, or credential payloads.

## Examples Review

1. Run `./scripts/validate-examples.sh`.
2. Confirm apply/sync/host deploy examples are guarded.
3. Confirm placeholder URLs and placeholder images are clearly example-only.
4. Remove examples only when they are invalid and cannot be marked accurately as future or guarded.

## Security Review

1. Run `./scripts/verify-no-secrets.sh`.
2. Review `docs/security/threat-model.md`.
3. Review `docs/security/security-review-checklist.md`.
4. Confirm production-shaped config keeps auth enabled.
5. Confirm logs, audit, diagnostics, examples, and release notes do not include secret values.

## Tag And Publish

After all checks pass and the maintainer approves:

```sh
git status --short
git tag <version>
git push origin <branch>
git push origin <version>
```

Then create the GitHub release using `.github/release-template.md`.

## After Publish

1. Verify the release page.
2. Verify checks passed on the pushed tag/branch.
3. Update the next development roadmap or checklist.
4. Record any follow-up blockers as issues or roadmap entries.
