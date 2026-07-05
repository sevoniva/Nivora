# Kubernetes Deployment Operations

Phase 6.0 hardens the Kubernetes YAML runtime from a planning foundation into a beta-grade controlled deployment path.

Nivora is still not GA production-ready. Kubernetes apply and rollback remain guarded operations.

## Safety Defaults

- Dry-run remains the default for examples.
- Apply is never default.
- Apply requires explicit confirmation through API or CLI.
- Rollback requires explicit confirmation.
- Prune/delete is not performed by default.
- Manifest safety checks run before server-side dry-run or apply.
- The default safety policy rejects missing target namespace, denied Kubernetes system namespaces, cluster-scoped resources, manifest namespace mismatches, privileged containers, hostPath volumes, host namespace modes, `:latest` images, manifests larger than 1 MiB, and more than 100 rendered resources.
- Kubeconfig files and credentials must not be committed.
- CI must not require a Kubernetes cluster.

## Supported Runtime Path

The Kubernetes YAML path is:

1. Render static YAML manifests.
2. Validate manifest shape.
3. Build desired resource inventory.
4. Create manifest snapshot.
5. Run policy and security gates.
6. Run Kubernetes manifest safety checks.
7. Run server-side dry-run through the configured `KubernetesManifestClient`.
8. Apply only when explicitly confirmed.
9. Watch rollout for common workload kinds.
10. Evaluate lightweight health.
11. Record logs, events, audit, timeline, diff, resource inventory, and rollback baseline.

Plan-only requests surface safety warnings without mutating a cluster. DeploymentRun execution fails before dry-run/apply when the safety policy denies the rendered manifest set.

Supported rollout checks:

- `Deployment`
- `StatefulSet`
- `DaemonSet`
- `Job`

Other resource kinds are kept in inventory and may produce rollout warnings rather than failing by default.

## CLI

Dry-run:

```sh
nivora deployment dry-run --local examples/deployments/yaml-dry-run.yaml
```

Guarded apply:

```sh
nivora deployment apply --local examples/deployments/yaml-apply-local.yaml --confirm
```

Guarded server-backed rollback:

```sh
nivora deployment rollback <deployment-run-id> --confirm --token-env NIVORA_AUTH_TOKEN
```

Server-backed deployment reads, apply, rollback, and sync commands use `--token-env` for the Nivora API bearer token. Local `--local` planning and dry-run commands do not need a server token.

## API

Create a dry-run DeploymentRun:

```http
POST /api/v1/deployments
```

Create a confirmed apply DeploymentRun:

```http
POST /api/v1/deployments/apply
Content-Type: application/json

{
  "definition": {
    "apiVersion": "nivora.io/v1alpha1",
    "kind": "Deployment",
    "metadata": {"name": "demo"},
    "spec": {
      "application": "demo",
      "environment": "dev",
      "target": {
        "type": "kubernetes-yaml",
        "name": "dev",
        "context": "my-explicit-context",
        "namespace": "default"
      },
      "manifests": ["examples/yaml/deployment.yaml"],
      "options": {"dryRun": false, "apply": true}
    }
  },
  "confirm": true
}
```

Run a guarded rollback:

```http
POST /api/v1/deployments/{id}/rollback
Content-Type: application/json

{"confirm": true}
```

## Kubectl Adapter

The kubectl adapter is isolated under `internal/adapters/executor/yaml_apply`.

It shells out to `kubectl` with manifest content passed through standard input. It requires:

- explicit Kubernetes context
- explicit namespace
- manifest documents

It supports:

- `kubectl apply --server-side --dry-run=server -f -`
- `kubectl apply -f -`
- rollout checks for supported workload kinds
- guarded manifest-restore rollback using apply, without prune/delete by default

The default in-process runtime still uses the no-op manifest client so normal tests do not require a cluster.

## Rollback

Phase 6.0 rollback is a manifest restore foundation:

- it is guarded by `confirm=true` / `--confirm`
- it does not delete resources
- it does not prune resources
- it records rollback events and audit entries
- it uses the configured manifest client

Future phases can add previous live-state capture, richer revision history, namespace locks, and environment-specific rollout policies.

## Limitations

- Helm and Kustomize rendering are not implemented here.
- Argo CD is not part of this phase.
- CRD-specific health is not implemented.
- Production namespace locking and advanced pruning policy remain future work.
