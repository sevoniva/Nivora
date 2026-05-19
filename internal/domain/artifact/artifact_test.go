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

func TestParseOCIReferenceWithRegistryPort(t *testing.T) {
	ref, err := ParseReference("localhost:30500/team/app:dev", ArtifactTypeImage)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if ref.Registry != "localhost:30500" || ref.Repository != "team/app" || ref.Name != "app" || ref.Tag != "dev" {
		t.Fatalf("reference = %#v", ref)
	}
}

func TestParseOCIReferenceWithTagAndDigest(t *testing.T) {
	ref, err := ParseReference("registry.example.com/team/app:1.0.0@sha256:abcdef", ArtifactTypeImage)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if ref.Tag != "1.0.0" || ref.Digest != "sha256:abcdef" || !ref.IsDigestPinned || !ref.Immutable {
		t.Fatalf("reference = %#v", ref)
	}
	if got := DigestQualifiedReference(ref, ref.Digest); got != "registry.example.com/team/app:1.0.0@sha256:abcdef" {
		t.Fatalf("digest qualified = %q", got)
	}
}

func TestParseRepositoryImageWithoutRegistry(t *testing.T) {
	ref, err := ParseReference("team/app:1.0.0", ArtifactTypeImage)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if ref.Registry != "" || ref.Repository != "team/app" || ref.Tag != "1.0.0" {
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
	digest, err := InspectReference("nginx@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ArtifactTypeImage)
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

func TestShortDigestProducesNonCanonicalWarning(t *testing.T) {
	inspection, err := InspectReference("nginx@sha256:abcdef", ArtifactTypeImage)
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if len(inspection.Warnings) != 1 || inspection.Warnings[0].Code != "non_canonical_digest" {
		t.Fatalf("warnings = %#v", inspection.Warnings)
	}
}
