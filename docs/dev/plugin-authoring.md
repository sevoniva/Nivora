# Plugin Authoring

Phase 7.4 stabilizes the plugin API foundation so contributors can describe external adapters safely.

Nivora still does not load Go plugins, install marketplace packages, or execute untrusted code by default. External plugins are separate processes that must expose an explicit HTTP or gRPC protocol and implement stable lifecycle operations.

## Plugin API Version

Current plugin API version:

```text
v1alpha1
```

Current manifest API version:

```text
nivora.io/plugin/v1alpha1
```

Every external plugin manifest should include:

- `apiVersion`
- `name`
- `type`
- `version`
- `protocol`
- `endpoint`
- `capabilities`
- `compatibility.pluginApiVersion`
- `compatibility.nivoraMinVersion`
- `lifecycle`

## Lifecycle

External plugins should support:

- `health`: reports whether the plugin process is available.
- `capabilities`: returns the plugin manifest and supported capabilities.
- `validateConfig`: validates adapter configuration without returning secret values.
- `execute`: performs the requested capability only after Nivora has authorized and audited the action.

## Safety Rules

- Do not put credentials in manifests.
- Use SecretRef or CredentialRef for sensitive values.
- Do not log secret values.
- Keep plugin endpoint URLs explicit in config or manifests; do not hardcode local services.
- Do not use Go `plugin` dynamic loading.
- Do not add marketplace behavior without an RFC.

## Validate a Manifest

Local validation:

```sh
go run ./cmd/nivora plugins validate --local --file examples/plugins/templates/scanner-plugin.yaml
```

Server validation:

```sh
go run ./cmd/nivora plugins validate --server http://localhost:8080 --file examples/plugins/templates/scanner-plugin.yaml
```

## Templates

Templates live under `examples/plugins/templates/`:

- `scm-plugin.yaml`
- `artifact-plugin.yaml`
- `cloud-plugin.yaml`
- `executor-plugin.yaml`
- `secret-plugin.yaml`
- `scanner-plugin.yaml`
- `notification-plugin.yaml`

They are examples only and use placeholder endpoints.
