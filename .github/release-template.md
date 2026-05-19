# <version>

This release is intended for architecture review, local validation, contributor onboarding, and maintainer-controlled release-candidate hardening.

Nivora must not be described as production-ready unless the matching release checklist has passed and maintainers explicitly approve that claim.

## Highlights

- <highlight 1>
- <highlight 2>
- <highlight 3>

## Known Limitations

- <limitation 1>
- <limitation 2>
- <limitation 3>

## Verification

- `make verify`
- `make helm-template` when Helm is available
- `make helm-lint` when Helm is available
- `make docker-build` when base image registries are reachable

## Install And Upgrade Notes

- Review `docs/operations/production-install.md`.
- Review `docs/operations/upgrade.md`.
- Review `docs/operations/backup-restore.md`.
- Review the matching checklist in `docs/releases/`.

## Security Notes

- Do not publish secret values, tokens, kubeconfigs, private keys, registry credentials, cloud credentials, or realistic fake credentials.
- Guarded operations such as apply, sync, remote host deploy, push, prune, and destructive rollback must remain explicit.
