# Host Deployment Dry-Run

Phase 8.1 supports safe host deployment planning and local/noop execution. It does not run remote SSH by default.

## Plan Locally

```sh
nivora deployment host plan --file examples/deployments/host-dry-run.yaml --local
```

The plan includes:

- artifact reference
- target host list
- versioned release directory
- `current`, `previous`, and `next` symlink paths
- HTTP, TCP, or command health check metadata
- batch rollout settings
- guarded symlink rollback baseline

## Run Through Noop Runtime

```sh
nivora deployment host run --file examples/deployments/host-dry-run.yaml --local
```

The noop executor records logs, events, audit records, and per-host details without mutating local or remote machines.

The same dry-run/noop host definition can be submitted to a running Nivora server:

```sh
nivora deployment host run --file examples/deployments/host-dry-run.yaml --local=false --server http://localhost:8080
```

Server-backed host run uses `POST /api/v1/deployments` and only accepts safe dry-run/noop input from the CLI. If the file requests `options.apply: true`, `host.allowRemoteHostDeploy: true`, `--confirm`, or `--allow-remote-host-deploy`, the CLI refuses the server-backed run instead of implying remote SSH execution.

## Guarded Remote Shape

Remote execution remains disabled by default. The SSH adapter surface requires:

- `options.apply: true`
- `--confirm`
- `--allow-remote-host-deploy`
- a `credentialRef`
- explicit adapter transport configuration

## Guarded Rollback Shape

Host rollback uses the same guarded deployment rollback endpoint and CLI shape:

```sh
nivora deployment rollback <deployment-run-id> --confirm
```

The default noop runtime records rollback logs, events, audit records, and timeline entries without mutating local or remote machines. A real SSH transport must still satisfy the same confirmation and CredentialRef requirements.

Do not put SSH keys, passwords, or realistic fake credentials in examples, tests, logs, audit records, or documentation.
