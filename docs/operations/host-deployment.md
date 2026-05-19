# Host Deployment Operations

Nivora host deployment is a controlled beta foundation. It is intended for VM and bare-metal release flows, but remote execution is disabled by default and the project remains not production-ready.

## Safe Defaults

- Examples use dry-run/noop execution.
- Remote SSH requires `options.apply: true`, `--confirm`, `host.allowRemoteHostDeploy: true`, and a CredentialRef.
- Secret values, SSH keys, passwords, and tokens must never be placed in deployment specs, logs, or audit records.
- Tests use fake or noop executors and do not require real SSH.

## Plan

```sh
nivora deployment host plan --file examples/deployments/host-dry-run.yaml --local
```

The plan includes:

- versioned release directory under `deployPath/releases/<deployment-run-id>`
- `current`, `previous`, and `next` symlink paths
- batch index per host
- HTTP, TCP, or command health checks
- guarded rollback plan

## Dry-Run

```sh
nivora deployment host run --file examples/deployments/host-dry-run.yaml --local
```

The noop executor records per-host details, logs, events, audit records, and timeline entries without mutating a host.

## Guarded Apply Shape

Remote apply remains disabled unless an explicitly configured host executor transport is present.

```sh
nivora deployment host run --file host-apply.yaml --local --confirm --allow-remote-host-deploy
```

The deployment spec must also opt in:

```yaml
options:
  dryRun: false
  apply: true
host:
  allowRemoteHostDeploy: true
  credentialRef: cred-host-placeholder
```

## Rollback

Host rollback is guarded and restores the `current` symlink from `previous`. It does not delete release directories by default.

```sh
nivora deployment rollback <deployment-run-id> --confirm
```

Real service restart behavior should use `serviceName` or `restartCommand`, and credentials must be supplied by SecretRef/CredentialRef.
