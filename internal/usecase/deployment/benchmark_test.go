package deployment

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkDeploymentResourceInventoryPlan(b *testing.B) {
	ctx := context.Background()
	service, def := newBenchmarkDeploymentService(b)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := service.Plan(ctx, CreateRunInput{Definition: def}); err != nil {
			b.Fatal(err)
		}
	}
}

func newBenchmarkDeploymentService(b *testing.B) (*Service, Definition) {
	b.Helper()
	dir := b.TempDir()
	manifest := filepath.Join(dir, "resources.yaml")
	if err := os.WriteFile(manifest, []byte(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: bench
spec:
  selector:
    matchLabels:
      app: bench
  template:
    metadata:
      labels:
        app: bench
    spec:
      containers:
        - name: bench
          image: example.invalid/bench/app:dev
---
apiVersion: v1
kind: Service
metadata:
  name: bench
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: bench-config
`), 0o600); err != nil {
		b.Fatalf("write manifest: %v", err)
	}
	def := Definition{
		APIVersion: "nivora.io/v1alpha1",
		Kind:       "Deployment",
		Metadata:   Metadata{Name: "bench-yaml"},
		Spec: Spec{
			Application: "bench",
			Environment: "dev",
			Target:      Target{Type: "kubernetes-yaml", Name: "bench-target", Namespace: "default"},
			Artifacts: []Artifact{{
				Name:      "bench",
				Type:      "image",
				Reference: "example.invalid/bench/app:dev",
			}},
			Manifests: []string{manifest},
			Options:   Options{DryRun: true, Apply: false},
		},
	}
	return NewService(NewMemoryStore(), StaticManifestRenderer{}, testManifestClient{}, testPolicy{allowed: true}, testEventBus{}), def
}
