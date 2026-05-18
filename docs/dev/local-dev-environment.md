# Local Development Environment

This page documents optional local services that maintainers may use for manual validation. These services are not required for CI, unit tests, Phase 1 shell execution, or Phase 1.5 runtime foundation work.

Do not commit credentials. Do not hardcode these endpoints in core code. Do not assume these services exist outside a maintainer's local machine. Use environment variables for credentials.

## Kubernetes Contexts

DevOps kind cluster:

```sh
kubectl config use-context kind-devops-kind
kubectl get pods -n fdo-pvt
```

Argo CD test kind cluster:

```sh
kubectl config use-context kind-argocd-test
kubectl get pods -A
```

## DevOps Demo Services

Harbor-lite registry:

- local: `localhost:30500`
- cluster: `harbor.fdo-pvt.svc.cluster.local:5000`

GitLab-lite:

```sh
kubectl -n fdo-pvt port-forward svc/gitlab 8088:80
```

Example local repository URL:

```text
http://localhost:8088/devops-demo-springboot.git
```

Nexus:

```sh
kubectl -n fdo-pvt port-forward svc/nexus 8081:8081
```

Use environment variables for credentials:

```sh
export NIVORA_LOCAL_NEXUS_USERNAME='<username>'
export NIVORA_LOCAL_NEXUS_PASSWORD='<password>'
```

Demo SpringBoot cluster service:

```text
demo-springboot.fdo-pvt.svc.cluster.local:8080
```

## Argo CD Test Services

Argo CD:

```sh
kubectl --context kind-argocd-test -n argocd port-forward svc/argocd-server 8080:80
```

Gitea:

```sh
kubectl --context kind-argocd-test -n gitea port-forward svc/gitea-http 3000:3000
```

Harbor:

```sh
kubectl --context kind-argocd-test -n harbor port-forward svc/harbor 8082:80
```

Optional external Harbor credentials must come from environment variables:

```sh
export NIVORA_EXTERNAL_HARBOR_URL='<url>'
export NIVORA_EXTERNAL_HARBOR_USERNAME='<username>'
export NIVORA_EXTERNAL_HARBOR_PASSWORD='<password>'
```

## Phase 1 / 1.5 Relevance

Phase 1 and Phase 1.5 minimal PipelineRun execution do not require any of these services. They only use the local shell Executor and in-memory runtime for tests and examples.

Do not treat successful local discovery against these services as proof that a real integration is complete. Kubernetes, Argo CD, Git provider, artifact registry, and cloud provider integrations remain future phases.
