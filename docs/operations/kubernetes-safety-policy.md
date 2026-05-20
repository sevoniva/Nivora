# Kubernetes Deployment Safety Policy

Nivora provides a Kubernetes safety policy that validates manifests before dry-run or apply. The policy catches common misconfigurations that could cause production incidents.

## Quick Start

```bash
# Use the default safety policy
nivora deployment plan --local examples/deployments/yaml-dry-run.yaml

# Use a custom policy
nivora deployment plan --local --policy examples/policies/kubernetes-safety-policy.yaml examples/deployments/yaml-dry-run.yaml
```

## Policy Rules

| Rule | Default | Behavior |
|---|---|---|
| `requireNamespace` | true | Rejects deployments without an explicit namespace |
| `denyPrivileged` | true | Rejects pods with `securityContext.privileged: true` |
| `denyHostPath` | true | Rejects volumes using `hostPath` |
| `denyHostNetwork` | true | Rejects pods with `hostNetwork: true` |
| `denyHostPID` | true | Rejects pods with `hostPID: true` |
| `denyHostIPC` | true | Rejects pods with `hostIPC: true` |
| `denyLatestTag` | true | Rejects container images with `:latest` tag or no tag |
| `denyClusterScoped` | true | Rejects ClusterRole, ClusterRoleBinding, Namespace, PersistentVolume, etc. |
| `deniedNamespaces` | `[kube-system, kube-public, kube-node-lease]` | Rejects deployments to these namespaces |
| `allowedNamespaces` | `[]` (any except denied) | If set, only these namespaces are allowed |
| `allowedKinds` | `[]` (any except denied) | If set, only these kinds are allowed |
| `deniedKinds` | `[]` | Always reject these kinds |
| `requireDigest` | false | Warns if container images lack a digest (`@sha256:...`) |
| `maxManifestBytes` | 1MB | Rejects manifests exceeding this size |

## Example: Safe Pod

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: safe-app
  namespace: staging
spec:
  containers:
  - name: app
    image: nginx@sha256:abc123...
```

This pod passes all default rules: non-privileged, no hostPath, no hostNetwork, tagged with digest, in an allowed namespace.

## Example: Denied Pod

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: unsafe-app
spec:
  hostNetwork: true
  containers:
  - name: app
    image: nginx:latest
    securityContext:
      privileged: true
```

This pod fails three rules: hostNetwork, latest tag, and privileged container.

## Integration with Deployment Flow

The safety policy runs as part of the DeploymentPreCheck phase, before dry-run or apply:

```
DeploymentRun Created → Planning → PreCheck (safety policy runs here) → Deploying → Verifying → Succeeded
```

If the policy denies the deployment, the DeploymentRun transitions to `Failed` with policy details in the plan output.

## Production Config Gating

Kubernetes apply is additionally gated by `runtime.allow_kubernetes_apply`. In production config:

```yaml
runtime:
  allow_kubernetes_apply: false  # Must be true for any apply
  runner_isolation_profile: container-isolated
```

Apply requires both `allow_kubernetes_apply=true` AND passing the safety policy.

## Current Limitations

- Policy validates static YAML manifests, not live cluster state.
- No admission webhook or dynamic policy engine (OPA/Kyverno).
- No runtime enforcement (policy runs pre-deploy, not during).
- No CRD-specific validation.
- No Helm/Kustomize native rendering (uses static YAML renderer).
- Live cluster apply requires explicit confirmation (`--confirm`).
- Cluster-scoped resource detection is based on known kinds; CRDs may not be caught.
