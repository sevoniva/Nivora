package artifact

import "testing"

func TestParseOCIReferenceWithTag(t *testing.T) {
	ref, err := ParseReference("registry.example.com/team/app:1.0.0", ArtifactTypeImage)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if ref.Registry != "registry.example.com" || ref.Repository != "team/app" || ref.Tag != "1.0.0" || ref.Immutable {
		t.Fatalf("reference = %#v", ref)
	}
}

func TestParseOCIReferenceWithDigest(t *testing.T) {
	ref, err := ParseReference("registry.example.com/team/app@sha256:abcdef", ArtifactTypeImage)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if ref.Digest != "sha256:abcdef" || !ref.Immutable {
		t.Fatalf("reference = %#v", ref)
	}
}

func TestImmutabilityWarnings(t *testing.T) {
	latest, err := InspectReference("nginx:latest", ArtifactTypeImage)
	if err != nil {
		t.Fatalf("inspect latest: %v", err)
	}
	if len(latest.Warnings) == 0 {
		t.Fatal("expected latest warning")
	}
	digest, err := InspectReference("nginx@sha256:abcdef", ArtifactTypeImage)
	if err != nil {
		t.Fatalf("inspect digest: %v", err)
	}
	if len(digest.Warnings) != 0 {
		t.Fatalf("digest warnings = %#v", digest.Warnings)
	}
	missing, err := InspectReference("nginx", ArtifactTypeImage)
	if err != nil {
		t.Fatalf("inspect missing: %v", err)
	}
	if len(missing.Warnings) == 0 {
		t.Fatal("expected missing tag/digest warning")
	}
}
