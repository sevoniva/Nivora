# Testing Strategy

Nivora uses layered tests so contributors can verify changes without requiring cloud accounts, local kind clusters, registries, Argo CD, or external scanners.

Nivora is still an alpha-stage project. These gates keep the foundation reliable; they do not claim production readiness.

## Test Layers

| Layer | Purpose | Command | CI |
| --- | --- | --- | --- |
| Unit tests | Validate domain, usecase, adapter, API, and CLI behavior with deterministic fakes. | `go test ./...` | Yes |
| Architecture checks | Keep domain/usecase/adapters/API boundaries intact. | `./scripts/verify-architecture.sh` | Yes |
| Secret scan | Catch high-confidence committed secret patterns. | `./scripts/verify-no-secrets.sh` | Yes |
| Example validation | Parse examples, validate supported runtime examples, check manifest references, and verify migration hygiene. | `make verify-examples` | Yes |
| API smoke | Start a local server, check health/readiness/version, create a PipelineRun, and create a DeploymentRun dry-run. | `make verify-api` | Yes |
| CLI smoke | Run version/config/pipeline/deployment/artifact CLI paths. | `make verify-cli` | Yes |
| Runtime smoke | Execute a local shell PipelineRun. | `make verify-runtime` | Yes |
| Packaging checks | Template and lint Helm chart when Helm is available. | `make verify-packaging` | Yes |
| Web build | Typecheck and build the minimal web foundation when `web/` exists. | `make verify-web` | Yes |
| Race tests | Exercise selected concurrent runtime/API packages with Go's race detector. | `make test-race` | Manual |
| Coverage | Generate a local Go coverage report. | `make coverage` | Manual |

## Baseline CI

CI should remain self-contained and deterministic. It runs formatting, module tidiness, vet, unit tests, binary builds, web build, architecture checks, secret scans, example validation, local runtime/API/CLI smoke checks, API spec parsing, packaging template checks, and alpha release documentation checks.

## Manual Checks

Maintainers can run these locally when useful:

```sh
make test-race
make coverage
make smoke-deployment-dry-run
make smoke-oci-resolve-local
make docker-build
make helm-template
make helm-lint
```

Optional local environment checks for kind, Harbor, Nexus, Gitea, or Argo CD are not CI requirements. They must remain explicitly opt-in and must not require committed credentials.

## Flake Prevention Rules

- Baseline tests must not require external network services.
- Tests should use fake/noop adapters for registries, cloud providers, scanners, notifications, runners, Argo CD, and Kubernetes.
- Avoid sleep-based assertions. Prefer explicit leases, deterministic timestamps, or short bounded polling around local processes.
- Do not require local kind clusters, Docker daemons, Harbor, Nexus, Gitea, GitLab, Argo CD, or cloud accounts in normal tests.
- Example files must stay credential-free and safe.
- Smoke scripts must print clear failure messages and clean up local processes.

## Quality Dashboard

| Signal | Healthy When |
| --- | --- |
| Formatting | `make fmt-check` passes |
| Dependencies | `make tidy-check` has no diff |
| Unit tests | `go test ./...` passes |
| Static analysis | `make vet` passes |
| Boundaries | `make verify-architecture` passes |
| Secrets | `make verify-no-secrets` passes |
| Examples | `make verify-examples` passes |
| API/CLI smoke | `make verify-api verify-cli` passes |
| Packaging | `make verify-packaging` passes or cleanly skips unavailable Helm |
| Full gate | `make verify` passes |
