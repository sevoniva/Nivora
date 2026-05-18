# Local Examples

These examples are optional and local-only. They must not be required by CI and must not contain credentials.

Phase 1 minimal PipelineRun execution can run without Kubernetes, Harbor, Nexus, GitLab, Gitea, Argo CD, or cloud services:

```sh
go run ./cmd/nivora pipeline run --local examples/pipelines/simple-shell.yaml
go run ./cmd/nivora pipeline run --local examples/pipelines/failing-shell.yaml
```

For optional local service discovery, see `docs/dev/local-dev-environment.md`.

