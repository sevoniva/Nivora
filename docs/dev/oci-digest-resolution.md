# OCI Digest Resolution

Phase 2.5 adds generic OCI image digest resolution. It uses the standard registry manifest API to read `Docker-Content-Digest` for an image tag when registry access is configured.

```bash
go run ./cmd/nivora artifact inspect registry.example.com/team/app:1.0.0
go run ./cmd/nivora artifact resolve registry.example.com/team/app@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
```

Network resolution is optional. CI tests use fake local registry servers and do not require external registry access.

For local HTTP registries, `--insecure` or `insecure: true` must be explicit:

```bash
go run ./cmd/nivora artifact resolve localhost:30500/team/app:1.0.0 --insecure
```

Do not pass credentials on command lines. Future credential-aware registry access should use config, environment variables, or SecretRef values without logging secret contents.

## Release Binding

Release definitions may set:

- `resolveDigest: true`: try to resolve tag references to digests.
- `requireDigest: true`: fail release creation if an artifact cannot be digest-pinned.

Digest-qualified ReleaseArtifacts are preferred for auditable deployment planning.
