# Quality Dashboard

The quality dashboard is a lightweight checklist for deciding whether the repository is healthy enough for alpha and beta preparation.

| Area | Gate | Command |
| --- | --- | --- |
| Formatting | Go files are gofmt-formatted. | `make fmt-check` |
| Modules | `go.mod` and `go.sum` are tidy. | `make tidy-check` |
| Static checks | Go vet passes. | `make vet` |
| Unit tests | All packages pass self-contained tests. | `make test` |
| Binaries | Server, worker, runner, and CLI build. | `make build` |
| Architecture | Boundary guardrails pass. | `make verify-architecture` |
| Secrets | No high-confidence secret patterns are committed. | `make verify-no-secrets` |
| Examples | Examples and migration files validate. | `make verify-examples` |
| API smoke | Local server health/readiness/version and runtime smoke pass. | `make verify-api` |
| CLI smoke | Developer CLI paths work. | `make verify-cli` |
| Specs | OpenAPI and AsyncAPI parse. | `make verify-api-specs` |
| Packaging | Helm templates and lint pass when Helm exists. | `make verify-packaging` |
| Full gate | All baseline checks pass. | `make verify` |

## Release Readiness Notes

- A passing dashboard means the repository is consistent enough for contributor review.
- It does not mean Nivora is production-ready.
- Optional integration checks must stay separate from baseline CI until they can run deterministically without secrets or external services.
