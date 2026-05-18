# Non-Goals

Nivora is intentionally not trying to become every DevOps tool.

## Not a Jenkins Clone

Nivora should not recreate Jenkins as a plugin-heavy CI server. It should model delivery lifecycle state and integrate with execution systems through Runners and Executors.

## Not an Argo CD Replacement

Nivora should integrate with Argo CD as one GitOps deployment mode. It should not replace Argo CD's reconciliation model.

## Not Kubernetes-Only

Nivora should support Kubernetes targets, but Environment and Release Target are broader concepts. A Release Target may be a host group, Kubernetes cluster, Argo CD application, cloud target, or webhook target.

## Not Cloud-Provider-Specific

Nivora should not center the architecture on AWS, Aliyun, Tencent Cloud, or any other provider. Cloud behavior belongs behind CloudProvider Adapters.

## Not Frontend-First

Nivora is backend-first. Visualization APIs and a frontend may arrive later, but current phases focus on backend architecture and delivery state.

## Not Production-Ready in Early Phases

Phase 0 / Phase 0.5 / Phase 0.6 are skeleton, guardrails, and planning phases. They do not provide a production delivery platform.

## Not Opaque Magic

Nivora should not hide all underlying tools. Operators should be able to see which Adapter, Executor, Artifact, Environment, Release Target, Policy, and Runner were involved.

