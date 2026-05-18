# GitOps

GitOps is a deployment mode where desired state is stored in Git and an external controller, such as Argo CD, reconciles that state to a target environment.

## Why It Exists

Many teams already use Argo CD or similar controllers for Kubernetes delivery. Nivora should coordinate release intent, artifacts, policy, audit, and timelines around those systems rather than replacing them.

## Current Implementation

Phase 2.6 supports GitOps planning and guarded Argo CD status/sync modeling:

- parse `argocd` deployment targets
- build a GitOpsChangePlan
- optionally update a local working tree
- model Argo CD application status with a noop provider
- read modeled Argo CD application resources
- require `gitops.allowSync=true`, explicit confirmation, and allow flags for sync requests
- reject force sync

No production Argo CD automation is implemented. No remote Git push or Git provider integration is implemented.

## Common Confusion

GitOps is not the same as deployment as a whole. Nivora also models shell pipelines, YAML deployment planning, host deployment as future work, cloud targets as future work, and artifact-based releases.
