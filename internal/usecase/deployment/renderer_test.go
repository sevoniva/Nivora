package deployment

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestStaticManifestRendererRendersAndSummarizesManifests(t *testing.T) {
	dir := t.TempDir()
	manifest := filepath.Join(dir, "resources.yaml")
	if err := os.WriteFile(manifest, []byte(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: demo
---

---
apiVersion: v1
kind: Service
metadata:
  name: demo
  namespace: custom
`), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	docs, err := (StaticManifestRenderer{}).Render(context.Background(), []string{manifest}, "default")
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("doc count = %d", len(docs))
	}
	if docs[0].Resource.Kind != "Deployment" || docs[0].Resource.Namespace != "default" {
		t.Fatalf("first resource = %#v", docs[0].Resource)
	}
	if docs[1].Resource.Kind != "Service" || docs[1].Resource.Namespace != "custom" {
		t.Fatalf("second resource = %#v", docs[1].Resource)
	}
}

func TestStaticManifestRendererRejectsInvalidManifest(t *testing.T) {
	dir := t.TempDir()
	manifest := filepath.Join(dir, "invalid.yaml")
	if err := os.WriteFile(manifest, []byte(`
kind: Deployment
metadata:
  name: demo
`), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	_, err := (StaticManifestRenderer{}).Render(context.Background(), []string{manifest}, "default")
	if err == nil {
		t.Fatal("expected render error")
	}
}
