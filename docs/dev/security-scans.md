# Security Scans

Phase 3.0 adds local security scan foundations.

```sh
go run ./cmd/nivora security scan artifact registry.example.com/demo/app:latest --local
go run ./cmd/nivora security scan manifest examples/security/manifest-privileged-warning.yaml --local
```

The default scanner is a noop scanner with lightweight built-in manifest checks. It is safe for CI and does not require Trivy, Cosign, registries, Kubernetes, or network access.

Security scan records include:

- subject type and ID
- scanner name
- status
- summary counts
- findings
- policy decision
- events and audit records

Do not put credentials, tokens, kubeconfigs, or secret values into scan inputs, findings, logs, or audit records.
