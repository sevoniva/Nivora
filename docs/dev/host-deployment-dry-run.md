# Host Deployment Dry-Run

Phase 3.5 supports safe host deployment planning and local/noop execution. It does not run remote SSH by default.

## Plan Locally

```sh
nivora deployment host plan --file examples/deployments/host-dry-run.yaml --local
```

The plan includes:

- artifact reference
- target host list
- versioned release directory
- `current`, `previous`, and `next` symlink paths
- health check metadata
- non-destructive rollback baseline

## Run Through Noop Runtime

```sh
nivora deployment host run --file examples/deployments/host-dry-run.yaml --local
```

The noop executor records logs, events, audit records, and per-host details without mutating local or remote machines.

## Guarded Remote Shape

Remote execution remains disabled by default. A future real SSH adapter must require:

- `options.apply: true`
- `--confirm`
- `--allow-remote-host-deploy`
- a `credentialRef`
- explicit adapter configuration

Do not put SSH keys, passwords, or realistic fake credentials in examples, tests, logs, audit records, or documentation.
