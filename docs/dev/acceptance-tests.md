# Acceptance Tests

This document lists acceptance checks Nivora maintainers should use before beta-facing changes.

## Baseline Acceptance

Run:

```sh
make verify
```

This verifies formatting, module tidiness, vet, Go tests, builds, architecture boundaries, secret scanning, examples, local smoke paths, API specs, packaging templates, alpha docs, and the web build when present.

## Runtime Acceptance

Run:

```sh
make verify-runtime
make verify-api
make verify-cli
```

Expected results:

- A local shell PipelineRun succeeds.
- API health, readiness, and version endpoints respond.
- API-created PipelineRun logs and timeline are retrievable.
- API-created DeploymentRun dry-run plan, resources, logs, and timeline are retrievable.
- CLI version, config validation, pipeline run, deployment plan, and artifact inspect commands work.

## Example Acceptance

Run:

```sh
make verify-examples
```

Expected results:

- Example YAML parses cleanly.
- Supported pipeline, deployment, release, and release orchestration examples validate through runtime parsers.
- Intentionally invalid examples remain invalid.
- Deployment examples reference existing manifest files.
- Migration files have reversible up/down pairs.
- High-risk secret-like literals are not present in examples.

## Optional Acceptance

Run locally when relevant:

```sh
make test-race
make coverage
make smoke-deployment-dry-run
make smoke-oci-resolve-local
```

These checks are useful before larger runtime changes, but they are not required for every edit because they may be slower or depend on optional local configuration.

## Non-Goals

Acceptance tests must not require cloud provider credentials, local Kubernetes clusters, Harbor, Nexus, Gitea, GitLab, Argo CD, production registry access, external notification providers, or real scanner binaries.
