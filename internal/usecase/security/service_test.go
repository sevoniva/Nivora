package security

import (
	"context"
	"testing"

	domainsecurity "github.com/sevoniva/nivora/internal/domain/security"
	portsecurity "github.com/sevoniva/nivora/internal/ports/security"
)

func TestFakeScannerCriticalFindingDenied(t *testing.T) {
	service := NewService(NewMemoryStore(), fakeScanner{findings: []domainsecurity.SecurityFinding{{
		Severity: domainsecurity.SeverityCritical,
		Category: domainsecurity.CategoryVulnerability,
		Target:   "demo",
		Title:    "critical vulnerability",
	}}}, nil, nil)
	record, err := service.Scan(context.Background(), ScanInput{SubjectType: domainsecurity.SubjectArtifact, SubjectID: "demo", Reference: "demo:1.0.0"})
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if record.Policy.Decision != domainsecurity.GateDeny {
		t.Fatalf("decision = %s, want deny", record.Policy.Decision)
	}
	if record.Scan.Summary.Critical != 1 {
		t.Fatalf("critical = %d, want 1", record.Scan.Summary.Critical)
	}
}

func TestNoFindingsAllowed(t *testing.T) {
	service := NewService(NewMemoryStore(), fakeScanner{}, nil, nil)
	record, err := service.Scan(context.Background(), ScanInput{SubjectType: domainsecurity.SubjectArtifact, SubjectID: "demo", Reference: "demo@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"})
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if record.Policy.Decision != domainsecurity.GateAllow {
		t.Fatalf("decision = %s, want allow", record.Policy.Decision)
	}
}

func TestLatestTagWarns(t *testing.T) {
	service := NewService(NewMemoryStore(), fakeScanner{}, nil, nil)
	result := service.Evaluate(EvaluateInput{SubjectType: domainsecurity.SubjectArtifact, SubjectID: "demo", Reference: "demo:latest"})
	if result.Decision != domainsecurity.GateWarn {
		t.Fatalf("decision = %s, want warn", result.Decision)
	}
}

func TestRequireDigestDenies(t *testing.T) {
	service := NewService(NewMemoryStore(), fakeScanner{}, nil, nil)
	result := service.Evaluate(EvaluateInput{SubjectType: domainsecurity.SubjectArtifact, SubjectID: "demo", Reference: "demo:1.0.0", Policy: PolicyConfig{RequireDigest: true, CriticalDenyThreshold: 1, HighWarnThreshold: 1}})
	if result.Decision != domainsecurity.GateDeny {
		t.Fatalf("decision = %s, want deny", result.Decision)
	}
}

func TestRequireDigestWithoutThresholdsStillDenies(t *testing.T) {
	service := NewService(NewMemoryStore(), fakeScanner{}, nil, nil)
	result := service.Evaluate(EvaluateInput{SubjectType: domainsecurity.SubjectArtifact, SubjectID: "demo", Reference: "demo:1.0.0", Policy: PolicyConfig{RequireDigest: true}})
	if result.Decision != domainsecurity.GateDeny {
		t.Fatalf("decision = %s, want deny", result.Decision)
	}
}

func TestManifestPrivilegedWarning(t *testing.T) {
	service := NewService(NewMemoryStore(), fakeScanner{}, nil, nil)
	record, err := service.Scan(context.Background(), ScanInput{SubjectType: domainsecurity.SubjectManifest, SubjectID: "manifest", Content: "securityContext:\n  privileged: true\n"})
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if record.Policy.Decision != domainsecurity.GateWarn {
		t.Fatalf("decision = %s, want warn", record.Policy.Decision)
	}
}

func TestListScansAndFindingsFilters(t *testing.T) {
	ctx := context.Background()
	service := NewService(NewMemoryStore(), fakeScanner{findings: []domainsecurity.SecurityFinding{{
		Severity: domainsecurity.SeverityCritical,
		Category: domainsecurity.CategoryVulnerability,
		Target:   "demo",
		Title:    "critical vulnerability",
	}}}, nil, nil)
	artifact, err := service.Scan(ctx, ScanInput{SubjectType: domainsecurity.SubjectArtifact, SubjectID: "demo-artifact", Reference: "registry.example.com/demo/app:1.0.0"})
	if err != nil {
		t.Fatalf("artifact scan: %v", err)
	}
	manifest, err := service.Scan(ctx, ScanInput{SubjectType: domainsecurity.SubjectManifest, SubjectID: "demo-manifest", Content: "securityContext:\n  privileged: true\n"})
	if err != nil {
		t.Fatalf("manifest scan: %v", err)
	}

	scans, err := service.ListScans(ctx, ListScansInput{SubjectType: domainsecurity.SubjectManifest})
	if err != nil {
		t.Fatalf("list scans: %v", err)
	}
	if len(scans) != 1 || scans[0].Scan.ID != manifest.Scan.ID {
		t.Fatalf("manifest scans = %#v, artifact=%s manifest=%s", scans, artifact.Scan.ID, manifest.Scan.ID)
	}

	findings, err := service.ListFindings(ctx, ListFindingsInput{SubjectType: domainsecurity.SubjectArtifact, Severity: domainsecurity.SeverityCritical})
	if err != nil {
		t.Fatalf("list findings: %v", err)
	}
	if len(findings) != 1 || findings[0].Metadata["scanId"] != artifact.Scan.ID {
		t.Fatalf("critical artifact findings = %#v", findings)
	}

	findings, err = service.ListFindings(ctx, ListFindingsInput{ScanID: manifest.Scan.ID, Category: domainsecurity.CategoryMisconfiguration})
	if err != nil {
		t.Fatalf("list manifest findings: %v", err)
	}
	if len(findings) == 0 {
		t.Fatalf("expected manifest misconfiguration finding")
	}
}

type fakeScanner struct {
	findings []domainsecurity.SecurityFinding
}

func (f fakeScanner) ScanArtifact(ctx context.Context, request portsecurity.ScanRequest) (portsecurity.ScanResult, error) {
	return portsecurity.ScanResult{Scanner: "fake", Findings: f.findings}, nil
}

func (f fakeScanner) ScanManifest(ctx context.Context, request portsecurity.ScanRequest) (portsecurity.ScanResult, error) {
	return portsecurity.ScanResult{Scanner: "fake", Findings: f.findings}, nil
}

func (f fakeScanner) ScanDeploymentPlan(ctx context.Context, request portsecurity.ScanRequest) (portsecurity.ScanResult, error) {
	return portsecurity.ScanResult{Scanner: "fake", Findings: f.findings}, nil
}

func (f fakeScanner) GetCapabilities(ctx context.Context) ([]portsecurity.Capability, error) {
	return nil, nil
}
