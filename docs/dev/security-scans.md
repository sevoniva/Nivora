# Security Scans

Phase 3.0 adds local security scan foundations.

```sh
go run ./cmd/nivora security scan artifact registry.example.com/demo/app:latest --local
go run ./cmd/nivora security scan manifest examples/security/manifest-privileged-warning.yaml --local
```

Stored scan results can be queried through the API:

```sh
curl "$NIVORA_URL/api/v1/security/scans?subjectType=manifest&limit=50"
curl "$NIVORA_URL/api/v1/security/findings?severity=High"
curl "$NIVORA_URL/api/v1/security/scans/<scan-id>/findings"
```

The matching CLI read commands are:

```sh
nivora security scans list --subject-type manifest
nivora security findings list --severity High
nivora security findings list --scan-id <scan-id>
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

The catalog query API is a foundation-level read path over stored scan records. It does not start new scans and does not replace Trivy, Cosign, SBOM, OPA, Kyverno, or Gatekeeper integrations.

Do not put credentials, tokens, kubeconfigs, or secret values into scan inputs, findings, logs, or audit records.
