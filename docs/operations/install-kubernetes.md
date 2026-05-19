# Install on Kubernetes

Nivora provides a minimal Helm chart and raw Kubernetes manifests for local validation. These assets are packaging foundations, not a production platform or operator.

## Helm

Render the chart:

```sh
make helm-template
```

Lint the chart when Helm is available:

```sh
make helm-lint
```

Install into the current Kubernetes context only when you explicitly allow it:

```sh
NIVORA_ALLOW_KIND_INSTALL=true make kind-install
```

The chart includes:

- server Deployment and Service
- worker Deployment
- optional runner Deployment
- ConfigMap with server, worker, and runner configs
- Secret template placeholder
- optional Ingress, disabled by default
- optional migration Job, disabled by default

## Raw Manifests

Minimal manifests live under `deployments/k8s/`:

- `configmap.yaml`
- `server.yaml`
- `worker.yaml`
- `runner.yaml`
- `service.yaml`

Example render/apply flow for local experimentation:

```sh
kubectl apply -f deployments/k8s/configmap.yaml
kubectl apply -f deployments/k8s/server.yaml
kubectl apply -f deployments/k8s/worker.yaml
kubectl apply -f deployments/k8s/runner.yaml
kubectl apply -f deployments/k8s/service.yaml
```

Do not use these raw manifests as a production hardening baseline.

## Secrets

No real secret values are committed. If auth tokens, database credentials, registry credentials, kubeconfigs, or cloud credentials are needed later, store them as Kubernetes Secrets or an external secret manager and reference them explicitly.

## Migrations

The Helm migration Job is disabled by default. Phase 4.4 does not introduce a production migration runner image. Operators should run migrations intentionally using a reviewed image and command.

## Current Limitations

- No Kubernetes operator.
- No cloud-provider dependency.
- HA, backup, and restore are documented as operating procedures, not automated by the chart.
- No production ingress/TLS defaults.
- Runtime persistence remains early-stage.

## HA and Backup Direction

For production-direction validation:

- run server replicas only with shared PostgreSQL/object store configuration
- keep workers independently restartable
- keep runners independently restartable and token-authenticated
- back up PostgreSQL outside the application pods
- back up object store buckets or persistent volumes
- keep Helm values free of raw secrets

See [HA and Disaster Recovery](ha-disaster-recovery.md) and [Backup and Restore](backup-restore.md).
