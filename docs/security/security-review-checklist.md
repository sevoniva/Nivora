# Security Review Checklist

This checklist supports Phase 9.2 security review before GA. It is not complete until maintainers close every unchecked release-blocking item.

## Threat Model

- [x] Control plane compromise scenario documented.
- [x] Runner compromise scenario documented.
- [x] Credential leakage scenario documented.
- [x] Malicious pipeline scenario documented.
- [x] Malicious deployment manifest scenario documented.
- [x] Supply-chain artifact issue scenario documented.
- [x] Audit tampering scenario documented.
- [x] Tenant isolation failure scenario documented.
- [ ] Maintainers have reviewed whether any threat requires a blocking code change before GA.

## Secure Defaults

- [x] `configs/production.example.yaml` has `auth.enabled: true`.
- [x] Production-shaped config uses token mode with `NIVORA_AUTH_TOKEN` as an environment variable reference, not a token value.
- [x] Insecure OCI registries require explicit `insecure: true` or CLI `--insecure`.
- [x] Kubernetes apply is guarded and not default in examples.
- [x] Argo CD sync is guarded by `sync`, `allowSync`, and confirmation.
- [x] Host remote deployment is guarded by explicit apply/confirm/credential/allow flags.
- [x] Secret values are represented by references or environment variables in examples.
- [x] Runner token hashes are stored; raw runner tokens are one-time outputs.
- [ ] Maintainers have reviewed production deployment values before RC/GA use.

## Credential Handling

- [x] Normal secret APIs return metadata or `SecretRef`, not secret values.
- [x] Credential examples contain placeholders only.
- [x] CLI docs prefer environment-variable based secret input.
- [x] Secret scan is part of `make verify`.
- [x] Example validation blocks common secret-like literals.
- [ ] External secret provider configuration has been reviewed for the target environment.

## API Auth And RBAC

- [x] Local dev auth is documented as non-production.
- [x] Token auth and OIDC foundation are documented.
- [x] Credential management APIs require credential management permissions.
- [x] Runner token rotation/revocation requires runner management permission.
- [x] Runner mutation endpoints require runner token validation.
- [ ] Maintainers have reviewed all mutation routes for permission coverage.
- [ ] Cross-tenant negative tests have been reviewed for critical resources.

## Runner Trust Boundary

- [x] Runner and server are separate concepts.
- [x] Runner protocol uses compact job claims rather than full domain object exposure.
- [x] Runner heartbeat, lease, concurrency, label, and capability foundations exist.
- [x] Runner sandbox limitations are documented in the threat model and runner docs.
- [ ] Maintainers have approved runner placement rules for any non-disposable environment.
- [ ] Untrusted workloads are isolated from privileged runner hosts.

## Redaction And Logging

- [x] Redaction tests cover passwords, authorization, bearer text, kubeconfig, access keys, client secrets, refresh tokens, ID tokens, session cookies, and private keys.
- [x] Security docs state that logs, audit, events, diagnostics, release notes, and examples must not include secret values.
- [x] Diagnostics are documented as metadata-only.
- [ ] Maintainers have reviewed representative logs from smoke tests for sensitive values.

## Audit Integrity

- [x] Audit records exist for important runtime and security-sensitive actions.
- [x] Evidence bundle docs avoid raw secret values.
- [x] Backup/restore docs include audit/event/log data.
- [ ] Maintainers have reviewed database permissions for audit tables in the target environment.
- [ ] Append-only or tamper-evident audit export has been evaluated if required by compliance goals.

## Supply Chain

- [x] Direct Go dependency set is small and documented in the threat model.
- [x] OCI digest binding and mutable tag warnings exist.
- [x] Release automation docs require clean-tree verification before release.
- [x] Docker builds are documented as dependent on external base-image registries.
- [ ] Maintainers have reviewed dependency advisory output for the RC/GA candidate.
- [ ] Maintainers have reviewed build provenance, SBOM, and signing requirements for GA.

## Required Commands

Run before closing this checklist:

```sh
go mod tidy
go test ./...
go vet ./...
go build ./cmd/nivora-server
go build ./cmd/nivora-worker
go build ./cmd/nivora-runner
go build ./cmd/nivora
./scripts/verify-architecture.sh
./scripts/verify-no-secrets.sh
./scripts/verify-api-specs.sh
./scripts/validate-examples.sh
make verify
git diff --check
```

## Release Blockers

- [ ] Any committed secret or realistic credential.
- [ ] Any normal API response returning secret values, token hashes, kubeconfigs, private keys, or credential payloads.
- [ ] Auth accidentally disabled in production-shaped config.
- [ ] Apply, sync, remote host deployment, Git push, prune, or destructive rollback enabled by default.
- [ ] Runner mutation endpoint accepting unauthenticated writes.
- [ ] Critical mutation route missing permission review.
- [ ] Secret-like values in logs, diagnostics, audit, events, release notes, or examples.
- [ ] Security review commands failing.
