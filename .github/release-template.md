# v0.1.0-alpha.1

This is a public alpha release of Nivora. It is intended for architecture review, local demos, contributor onboarding, and early feedback.

Nivora is not production-ready.

## Highlights

- Backend control-plane foundation in Go.
- PipelineRun, DeploymentRun, Release, Artifact, GitOps, policy, credential, auth, approval, cloud inventory, host deployment, visualization, observability, plugin, and packaging foundations.
- Local verification through `make verify`.
- Docker Compose, Helm, and Kubernetes manifest examples.

## Known Limitations

- No production-grade distributed runtime or external queue.
- No production Kubernetes apply semantics or destructive rollback.
- No production Argo CD automation.
- No full cloud, Git provider, registry, scanner, signer, notification, ITSM, Vault, or SSO integrations.
- Web UI is a minimal foundation.

## Verification

- `make verify`
- `make helm-template`
- `make helm-lint`
- `make docker-build` when Docker Hub and base image registries are reachable

## Upgrade Notes

This is the first alpha release. APIs and configuration may change before beta.
