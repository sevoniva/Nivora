# Policy Gates

Run a local policy evaluation:

```sh
go run ./cmd/nivora policy evaluate --subject registry.example.com/demo/app:latest
go run ./cmd/nivora policy evaluate --subject registry.example.com/demo/app:1.0.0 --require-digest
```

Phase 3.0 supports minimal built-in policy behavior:

- Critical findings deny by default.
- High findings warn by default.
- `latest` and tag-only image references warn.
- `--require-digest` denies references without `@sha256:`.

DeploymentRun and ReleaseExecution pre-checks can attach security decisions through the local security service. Deny stops execution; warn continues with warnings. Approval-required gates are modeled but durable approval workflow remains future work.

Policy definitions can also be managed through the server-backed foundation catalog:

```sh
nivora policy create --name "Require digest" --project-id demo --environment-id prod --require-digest --token-env NIVORA_AUTH_TOKEN
nivora policy attach policy-example --scope-type environment --scope-id prod --token-env NIVORA_AUTH_TOKEN
nivora policy attachments policy-example --scope-type environment --token-env NIVORA_AUTH_TOKEN
nivora policy evaluate policy-example --subject registry.example.com/demo/app:latest --token-env NIVORA_AUTH_TOKEN
nivora policy list --project-id demo --token-env NIVORA_AUTH_TOKEN
nivora policy disable policy-example --token-env NIVORA_AUTH_TOKEN
```

The catalog records built-in gate configuration and foundation scope attachments only. Saved policy evaluation uses the stored policy definition and rejects disabled policies. OPA, Kyverno, Gatekeeper, and external policy distribution are future work.

Security scan creation can now use saved policies in two ways:

- `POST /api/v1/security/scans` may include `policyId` to apply one enabled saved policy.
- If `policyId` and an ad hoc `policy` are omitted, the scan resolves the first enabled security policy attachment for `environmentId`, then `projectId`, then `global`.

Disabled policies are rejected when explicitly requested and skipped during attachment resolution. This is still the built-in policy gate foundation; it is not OPA/Kyverno/Gatekeeper distribution or an enterprise policy engine.
