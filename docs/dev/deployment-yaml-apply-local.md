# Local YAML Apply

Phase 2.1 includes an explicit local apply command backed by the safe no-op manifest client.

```bash
go run ./cmd/nivora deployment apply --local examples/deployments/yaml-apply-local.yaml --confirm
```

The command requires `--confirm`. The Makefile target also requires:

```bash
NIVORA_ALLOW_LOCAL_APPLY=true make deployment-apply-local
```

In the default repository runtime this does not mutate a Kubernetes cluster. It exercises the apply path, events, audit records, logs, resource inventory, rollout model, and rollback baseline.

Future kubectl or Kubernetes SDK adapters must be optional, configurable, and isolated outside domain and usecase packages.
