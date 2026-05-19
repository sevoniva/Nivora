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

Render the production-like profile without installing it:

```sh
helm template nivora deployments/helm -f deployments/helm/values-production.yaml
./scripts/smoke-helm-production-profile.sh
```

`deployments/helm/values-production.yaml` is a safer profile for operator review. It sets `environment: production`, uses `database.runtime_store: postgres`, references an existing Kubernetes Secret for `NIVORA_AUTH_TOKEN`, disables local shell, privileged executor, Kubernetes apply, Argo sync, remote host deploy, and insecure registry defaults, and adds starter resource requests/limits. It is still not a production-ready install by itself.

Install into the current Kubernetes context only when you explicitly allow it:

```sh
NIVORA_ALLOW_KIND_INSTALL=true make kind-install
```

The chart includes:

- server Deployment and Service
- worker Deployment
- optional runner Deployment
- ConfigMap with server, worker, and runner configs
- Secret template placeholder for non-production installs, or an existing Secret reference for production-like values
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

The Helm migration Job is disabled by default, including in `values-production.yaml`. Operators should run migrations intentionally using a reviewed image and command before starting production-like server/worker pods. The chart documents the job shape but does not automate a production migration process yet.

## Production Profile Checklist

- Use `deployments/helm/values-production.yaml` as a review starting point.
- Keep `database.runtime_store: postgres`.
- Do not render placeholder Secret values; set `secret.existingName`.
- Keep unsafe runtime flags false unless a specific runbook explains the risk.
- Run `./scripts/smoke-helm-production-profile.sh` before install.
- Back up PostgreSQL and object-store data before migration or restore drills.

## Current Limitations

- No Kubernetes operator.
- No cloud-provider dependency.
- HA, backup, and restore are documented as operating procedures, not automated by the chart.
- No production ingress/TLS defaults.
- Runtime persistence is improved for PipelineRun, DeploymentRun, Release, ReleaseExecution, and compliance evidence foundations, but production recovery is still hardening work.

## HA and Backup Direction

For production-direction validation:

- run server replicas only with shared PostgreSQL/object store configuration
- keep workers independently restartable
- keep runners independently restartable and token-authenticated
- back up PostgreSQL outside the application pods
- back up object store buckets or persistent volumes
- keep Helm values free of raw secrets

See [HA and Disaster Recovery](ha-disaster-recovery.md) and [Backup and Restore](backup-restore.md).
