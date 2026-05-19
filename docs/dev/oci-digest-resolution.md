# OCI Digest Resolution

Phase 6.2 hardens generic OCI image digest resolution. It uses the standard registry manifest API to read `Docker-Content-Digest` for an image tag when registry access is configured, and captures media type, size, and schema summary when the registry returns them.

```bash
go run ./cmd/nivora artifact inspect registry.example.com/team/app:1.0.0
go run ./cmd/nivora artifact resolve registry.example.com/team/app@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
```

Network resolution is optional. CI tests use fake local registry servers and do not require external registry access.

For local HTTP registries, `--insecure` or `insecure: true` must be explicit:

```bash
go run ./cmd/nivora artifact resolve localhost:30500/team/app:1.0.0 --registry localhost:30500 --insecure
```

Do not pass credentials as literal command-line values. Use environment variable indirection or SecretRef values without logging secret contents:

```bash
go run ./cmd/nivora artifact resolve registry.example.com/team/app:1.0.0 \
  --registry registry.example.com \
  --username-env NIVORA_REGISTRY_USERNAME \
  --password-env NIVORA_REGISTRY_PASSWORD
```

## Release Binding

Release definitions may set:

- `resolveDigest: true`: try to resolve tag references to digests.
- `requireDigest: true`: fail release creation if an artifact cannot be digest-pinned.
- `blockMutable: true`: fail release creation for mutable tag references unless digest resolution succeeds.

Digest-qualified ReleaseArtifacts are preferred for auditable deployment planning.

Harbor-compatible registries are treated as generic OCI registries in this phase. Harbor administration APIs, robot account management, scanning, signing, Nexus/JFrog management APIs, and cloud registry APIs remain future work.
