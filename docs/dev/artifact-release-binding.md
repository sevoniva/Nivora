# Artifact and Release Binding

Phase 2.2 introduces a local, testable foundation for release artifacts. It is designed for contributor development and does not require a registry.

## Inspect an Artifact Reference

```bash
go run ./cmd/nivora artifact inspect registry.example.com/team/app:1.0.0
go run ./cmd/nivora artifact inspect registry.example.com/team/app@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
```

Inspection parses the reference, normalizes it, and returns warnings for mutable or incomplete references.

## Create a Release Locally

```bash
go run ./cmd/nivora release create --local --file examples/releases/simple-release.yaml
```

The local command creates an in-memory Release record, marks it `Ready` after artifact binding succeeds, binds ReleaseArtifacts, emits events, and records audit entries. The local process does not persist records after the command exits.

## Generate Release Evidence

For server-backed release records:

```bash
nivora release evidence <release-id>
nivora release evidence <release-id> --format markdown
```

The matching API is `POST /api/v1/releases/{id}/evidence`. It creates a compliance evidence bundle for the release and includes available release, artifact binding, event, and audit references. It does not deploy, approve, roll back, or mutate ReleaseExecution state.

## Cancel Release Intent

Server-backed release records can be canceled without executing rollback or target actions:

```bash
nivora release cancel <release-id>
```

The matching API is `POST /api/v1/releases/{id}/cancel`. It marks the Release record `Canceled`, appends a release event, records release audit evidence, and safely cancels non-terminal ReleaseExecutions for the same Release. Each canceled ReleaseExecution also asks linked non-terminal DeploymentRuns to cancel. It does not execute rollback, delete resources, or mutate ReleaseExecutions or DeploymentRuns that are already terminal. The response includes `canceledExecutionIds` so operators can confirm which execution records changed.

## Query Tracked Artifacts

Server-backed artifacts created through release binding can be listed and traced back to releases:

```bash
nivora artifact list --registry registry.example.com
nivora artifact get <artifact-id>
nivora artifact releases <artifact-id>
```

The matching APIs are `GET /api/v1/artifacts`, `GET /api/v1/artifacts/{id}`, and `GET /api/v1/artifacts/{id}/releases`. This is a control-plane inventory of artifacts Nivora has seen through releases. It does not enumerate an external registry.

## Deployment Planning With Artifacts

Deployment specs can include artifact references. Phase 2.2 verifies simple Kubernetes workload image references against those artifacts and adds warnings to the DeploymentPlan when:

- a manifest image is not bound to an artifact
- a manifest image uses `latest`
- a manifest image lacks a digest
- an explicitly targeted artifact does not match the manifest image

Manifest mutation is not the default. Image substitution is reserved for explicit future work and must stay auditable.

## Registry Access

Harbor, Nexus, JFrog, cloud registries, authenticated digest resolution, artifact scanning, signing, and SBOM verification are future phases. Do not hardcode registry endpoints or credentials. Optional local registry values must come from environment variables only and must never be committed.
