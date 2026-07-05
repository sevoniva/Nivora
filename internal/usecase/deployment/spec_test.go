package deployment

import "testing"

func TestParseDefinitionValidatesDeploymentSpec(t *testing.T) {
	def, err := ParseDefinition([]byte(`
apiVersion: nivora.io/v1alpha1
kind: Deployment
metadata:
  name: demo
spec:
  application: demo-app
  environment: dev
  target:
    type: kubernetes-yaml
    name: dev-kind
    namespace: default
  artifacts:
    - name: demo-app
      type: image
      reference: example.local/demo:dev
  manifests:
    - deployment.yaml
  options:
    dryRun: true
    apply: false
`))
	if err != nil {
		t.Fatalf("parse definition: %v", err)
	}
	if def.Metadata.Name != "demo" {
		t.Fatalf("metadata.name = %q", def.Metadata.Name)
	}
	if err := def.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
}

func TestParseDefinitionKubernetesSafetyOptions(t *testing.T) {
	def, err := ParseDefinition([]byte(`
apiVersion: nivora.io/v1alpha1
kind: Deployment
metadata:
  name: demo
spec:
  application: demo-app
  environment: dev
  target:
    type: kubernetes-yaml
    name: dev-kind
    namespace: staging
  manifests:
    - deployment.yaml
  kubernetesSafety:
    allowedNamespaces: ["staging"]
    deniedNamespaces: ["prod"]
    allowedKinds: ["Deployment", "Service"]
    deniedKinds: ["Secret"]
    maxManifestBytes: 65536
    maxResourceCount: 10
    requireDigest: true
  options:
    dryRun: true
    apply: false
`))
	if err != nil {
		t.Fatalf("parse definition: %v", err)
	}
	if got := def.Spec.KubernetesSafety.AllowedNamespaces; len(got) != 1 || got[0] != "staging" {
		t.Fatalf("allowed namespaces = %#v", got)
	}
	if !def.Spec.KubernetesSafety.RequireDigest {
		t.Fatal("expected requireDigest")
	}
	if def.Spec.KubernetesSafety.MaxResourceCount != 10 {
		t.Fatalf("max resource count = %d", def.Spec.KubernetesSafety.MaxResourceCount)
	}
}

func TestDefinitionValidationRejectsInvalidSpec(t *testing.T) {
	tests := []struct {
		name string
		def  Definition
	}{
		{name: "missing name", def: Definition{Kind: "Deployment", Spec: Spec{Application: "app", Environment: "dev", Target: Target{Type: "kubernetes-yaml", Name: "target"}, Manifests: []string{"a.yaml"}}}},
		{name: "missing target type", def: Definition{Kind: "Deployment", Metadata: Metadata{Name: "demo"}, Spec: Spec{Application: "app", Environment: "dev", Target: Target{Name: "target"}, Manifests: []string{"a.yaml"}}}},
		{name: "unsupported target type", def: Definition{Kind: "Deployment", Metadata: Metadata{Name: "demo"}, Spec: Spec{Application: "app", Environment: "dev", Target: Target{Type: "argocd", Name: "target"}, Manifests: []string{"a.yaml"}}}},
		{name: "missing namespace", def: Definition{Kind: "Deployment", Metadata: Metadata{Name: "demo"}, Spec: Spec{Application: "app", Environment: "dev", Target: Target{Type: "kubernetes-yaml", Name: "target"}, Manifests: []string{"a.yaml"}}}},
		{name: "missing manifests", def: Definition{Kind: "Deployment", Metadata: Metadata{Name: "demo"}, Spec: Spec{Application: "app", Environment: "dev", Target: Target{Type: "kubernetes-yaml", Name: "target", Namespace: "default"}}}},
		{name: "apply with dry run", def: Definition{Kind: "Deployment", Metadata: Metadata{Name: "demo"}, Spec: Spec{Application: "app", Environment: "dev", Target: Target{Type: "kubernetes-yaml", Name: "target", Namespace: "default"}, Manifests: []string{"a.yaml"}, Options: Options{Apply: true, DryRun: true}}}},
		{name: "negative timeout", def: Definition{Kind: "Deployment", Metadata: Metadata{Name: "demo"}, Spec: Spec{Application: "app", Environment: "dev", Target: Target{Type: "kubernetes-yaml", Name: "target", Namespace: "default"}, Manifests: []string{"a.yaml"}, Options: Options{TimeoutSeconds: -1}}}},
		{name: "negative kubernetes safety bytes", def: Definition{Kind: "Deployment", Metadata: Metadata{Name: "demo"}, Spec: Spec{Application: "app", Environment: "dev", Target: Target{Type: "kubernetes-yaml", Name: "target", Namespace: "default"}, Manifests: []string{"a.yaml"}, KubernetesSafety: KubernetesSafetyOptions{MaxManifestBytes: -1}}}},
		{name: "negative kubernetes safety resources", def: Definition{Kind: "Deployment", Metadata: Metadata{Name: "demo"}, Spec: Spec{Application: "app", Environment: "dev", Target: Target{Type: "kubernetes-yaml", Name: "target", Namespace: "default"}, Manifests: []string{"a.yaml"}, KubernetesSafety: KubernetesSafetyOptions{MaxResourceCount: -1}}}},
		{name: "empty kubernetes safety namespace", def: Definition{Kind: "Deployment", Metadata: Metadata{Name: "demo"}, Spec: Spec{Application: "app", Environment: "dev", Target: Target{Type: "kubernetes-yaml", Name: "target", Namespace: "default"}, Manifests: []string{"a.yaml"}, KubernetesSafety: KubernetesSafetyOptions{AllowedNamespaces: []string{""}}}}},
		{name: "empty kubernetes safety kind", def: Definition{Kind: "Deployment", Metadata: Metadata{Name: "demo"}, Spec: Spec{Application: "app", Environment: "dev", Target: Target{Type: "kubernetes-yaml", Name: "target", Namespace: "default"}, Manifests: []string{"a.yaml"}, KubernetesSafety: KubernetesSafetyOptions{DeniedKinds: []string{" "}}}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.def.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestParseDefinitionValidatesGitOpsSpec(t *testing.T) {
	def, err := ParseDefinition([]byte(`
apiVersion: nivora.io/v1alpha1
kind: Deployment
metadata:
  name: demo-gitops
spec:
  application: demo-app
  environment: dev
  target:
    type: argocd
    name: demo-argocd
    applicationName: demo-app
    repoURL: https://example.com/gitops/demo.git
    path: apps/demo/dev
    revision: main
  artifacts:
    - name: demo-app
      type: image
      reference: registry.example.com/demo/app@sha256:example
  gitops:
    mode: plan
    writeToWorkingTree: false
    sync: false
`))
	if err != nil {
		t.Fatalf("parse gitops definition: %v", err)
	}
	if def.Spec.Target.Type != "argocd" {
		t.Fatalf("target type = %q", def.Spec.Target.Type)
	}
	if def.Spec.GitOps.Sync {
		t.Fatal("sync should default false")
	}
}

func TestParseDefinitionAllowsGitOpsRepositoryID(t *testing.T) {
	def, err := ParseDefinition([]byte(`
apiVersion: nivora.io/v1alpha1
kind: Deployment
metadata:
  name: demo-gitops
spec:
  application: demo-app
  environment: dev
  target:
    type: argocd
    name: demo-argocd
    applicationName: demo-app
    repositoryId: repo-1
    path: apps/demo/dev
  artifacts:
    - name: demo-app
      type: image
      reference: registry.example.com/demo/app@sha256:example
  gitops:
    mode: plan
`))
	if err != nil {
		t.Fatalf("parse gitops definition: %v", err)
	}
	if err := def.Validate(); err != nil {
		t.Fatalf("validate repositoryId gitops definition: %v", err)
	}
}
