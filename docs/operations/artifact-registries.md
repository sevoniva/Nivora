# Artifact Registries

Phase 6.2 hardens Nivora's OCI artifact registry foundation for beta-grade local and server-side validation. Harbor is treated as an OCI-compatible registry through the standard OCI Distribution API.

Nivora still does not implement Harbor administration, Nexus/JFrog administration, vulnerability scanning, signing, or GA production registry management.

## Supported in Phase 6.2

- OCI image reference parsing and normalization.
- Digest-pinned image recognition.
- Digest resolution through OCI manifest `HEAD` / `GET` requests.
- Artifact metadata capture:
  - digest
  - digest-qualified reference
  - media type
  - size when returned by the registry
  - manifest schema summary when available
- Explicit insecure registry configuration for local HTTP registries.
- Registry credentials through config, environment variables, or `SecretProvider`-backed credential refs. In server runtime wiring, credential, artifact, and artifact-registry services share the same configured provider so saved registry listing can resolve the referenced secret internally.
- Release artifact binding with `resolveDigest`, `requireDigest`, and `blockMutable`.
- Deployment image verification and guarded digest substitution when a deployment artifact target requests substitution.

## Safety Rules

- Registry access is optional and not required for CI.
- `insecure: true` must be explicit for HTTP registries.
- Registry endpoint URLs must not contain inline username/password material. Use `CredentialRef` metadata and a secret provider instead.
- Credentials must not be committed.
- CLI examples should use `--username-env`, `--password-env`, or `--token-env` rather than literal values.
- Secret values are never returned by normal APIs and should not appear in logs or audit records.
- `latest` and tag-only references produce mutable reference warnings.
- `requireDigest: true` fails release creation if no digest can be resolved.
- `blockMutable: true` fails release creation for mutable references unless digest resolution succeeds.

## CLI Examples

Inspect without network:

```bash
go run ./cmd/nivora artifact inspect registry.example.com/team/app:1.0.0
```

Resolve against an explicitly configured registry:

```bash
go run ./cmd/nivora artifact resolve registry.example.com/team/app:1.0.0 --registry registry.example.com
```

Resolve against a local insecure OCI registry:

```bash
go run ./cmd/nivora artifact resolve localhost:30500/team/app:dev --registry localhost:30500 --insecure
```

Resolve with credentials from environment variables:

```bash
go run ./cmd/nivora artifact resolve registry.example.com/team/app:1.0.0 \
  --registry registry.example.com \
  --username-env NIVORA_REGISTRY_USERNAME \
  --password-env NIVORA_REGISTRY_PASSWORD
```

Validate registry metadata without contacting the registry:

```bash
go run ./cmd/nivora artifact registry validate \
  --name local-oci \
  --endpoint http://localhost:30500 \
  --insecure \
  --credential-ref cred-local-registry
```

Validation rejects endpoints such as `https://user:password@registry.example.com`; registry credentials belong behind `CredentialRef`/`SecretRef` boundaries.

Saved registry validation is metadata-only. It checks endpoint shape, enabled state, and unsafe settings, but it does not contact the registry or read the secret. Registry listing and digest resolution are the paths that may resolve a `CredentialRef` through the configured `SecretProvider`.

## Release Binding

```yaml
apiVersion: nivora.io/v1alpha1
kind: Release
metadata:
  name: demo-release
spec:
  version: 1.0.0
  resolveDigest: true
  requireDigest: true
  blockMutable: true
  artifacts:
    - name: demo
      type: image
      reference: registry.example.com/team/demo:1.0.0
      required: true
```

## Current Limitations

- Harbor is supported only through OCI-compatible image manifest APIs.
- Harbor project/user/robot account APIs are not implemented.
- Nexus and JFrog management APIs remain future work.
- Token refresh and advanced registry auth flows remain future work.
- The builtin in-memory secret provider is development-only; production-like installs should use an external secret provider foundation or operator-managed secret source when available.
- CI tests use fake/local HTTP registries and do not require external registry access.
