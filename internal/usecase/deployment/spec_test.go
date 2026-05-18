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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.def.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
