package artifact

import "testing"

func TestDigestPinnedReferenceAccepted(t *testing.T) {
	p := DefaultImmutabilityPolicy()
	ref, err := ParseReference("registry.example.com/app@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ArtifactTypeImage)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	result := p.Evaluate([]Reference{ref}, "")
	if !result.Allowed {
		t.Fatalf("expected digest-pinned to be allowed, got denials: %v", result.Denials)
	}
	if !result.IsDigestPinned {
		t.Fatal("expected IsDigestPinned=true")
	}
	if !result.Immutable {
		t.Fatal("expected Immutable=true")
	}
}

func TestLatestTagDeniedInProduction(t *testing.T) {
	p := DefaultImmutabilityPolicy()
	ref, err := ParseReference("registry.example.com/app:latest", ArtifactTypeImage)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	result := p.Evaluate([]Reference{ref}, "")
	if result.Allowed {
		t.Fatal("expected latest tag to be denied in production")
	}
	if len(result.Denials) == 0 {
		t.Fatal("expected denials for latest tag")
	}
}

func TestLatestTagWarnedInDev(t *testing.T) {
	p := DevImmutabilityPolicy()
	ref, err := ParseReference("registry.example.com/app:latest", ArtifactTypeImage)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	result := p.Evaluate([]Reference{ref}, "")
	if !result.Allowed {
		t.Fatal("expected latest tag to be allowed in dev mode")
	}
	if len(result.Warnings) == 0 {
		t.Fatal("expected warnings for latest tag in dev mode")
	}
}

func TestMissingDigestDeniedWhenRequired(t *testing.T) {
	p := DefaultImmutabilityPolicy()
	ref, err := ParseReference("registry.example.com/app:1.0.0", ArtifactTypeImage)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	result := p.Evaluate([]Reference{ref}, "")
	if result.Allowed {
		t.Fatal("expected tag-only reference to be denied when RequireDigest=true")
	}
}

func TestOverrideRequiresReason(t *testing.T) {
	p := DefaultImmutabilityPolicy()
	ref, err := ParseReference("registry.example.com/app:latest", ArtifactTypeImage)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	// With override reason, latest tag should be allowed.
	result := p.Evaluate([]Reference{ref}, "emergency hotfix approved by security")
	if !result.Allowed {
		t.Fatalf("expected override to allow latest tag, got denials: %v", result.Denials)
	}
	if result.OverrideReason == "" {
		t.Fatal("expected override reason to be recorded")
	}
}

func TestImmutabilityResultIsDigestPinned(t *testing.T) {
	p := DefaultImmutabilityPolicy()
	// Canonical digest.
	ref, _ := ParseReference("registry.example.com/app@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ArtifactTypeImage)
	result := p.Evaluate([]Reference{ref}, "")
	if !result.IsDigestPinned {
		t.Fatal("expected canonical digest to be pinned")
	}
	if !result.Immutable {
		t.Fatal("expected canonical digest to be immutable")
	}
}

func TestEmptyRefsProducesWarning(t *testing.T) {
	p := DefaultImmutabilityPolicy()
	result := p.Evaluate(nil, "")
	if len(result.Warnings) == 0 {
		t.Fatal("expected warning for empty refs")
	}
}

func TestDevPolicyAllowsTaggedRefs(t *testing.T) {
	p := DevImmutabilityPolicy()
	ref, _ := ParseReference("registry.example.com/app:1.0.0", ArtifactTypeImage)
	result := p.Evaluate([]Reference{ref}, "")
	if !result.Allowed {
		t.Fatalf("expected tagged ref to be allowed in dev policy, got: %v", result.Denials)
	}
}

func TestProductionPolicyDeniesTagWithoutDigest(t *testing.T) {
	p := DefaultImmutabilityPolicy()
	ref, _ := ParseReference("registry.example.com/team/app:1.0.0", ArtifactTypeImage)
	result := p.Evaluate([]Reference{ref}, "")
	if result.Allowed {
		t.Fatal("expected tag-only to be denied in production")
	}
}
