# Release Target

A Release Target is the concrete destination for a DeploymentRun.

## Why It Exists

Environments describe delivery context. Release Targets describe where work would happen inside that context, such as a host group, Kubernetes namespace, Argo CD application, cloud target, or webhook target.

## Current Implementation

Phase 2.1 supports the `kubernetes-yaml` target type for static manifest dry-run and explicit no-op local apply. The target may include a Kubernetes context name, but no context is hardcoded and no kubeconfig is stored.

## Common Confusion

A Release Target is not always a Kubernetes cluster. GitOps, host, cloud, and webhook targets remain future modes.
