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

func TestEvaluateRecordsPolicyIDAndAppliesPolicyMode(t *testing.T) {
	service := NewService(NewMemoryStore(), fakeScanner{}, nil, nil)
	result := service.Evaluate(EvaluateInput{
		SubjectType: domainsecurity.SubjectArtifact,
		SubjectID:   "demo",
		Reference:   "demo:latest",
		PolicyID:    "policy-latest",
		PolicyMode:  "deny",
	})
	if result.PolicyID != "policy-latest" {
		t.Fatalf("policyId = %q, want policy-latest", result.PolicyID)
	}
	if result.Decision != domainsecurity.GateDeny {
		t.Fatalf("decision = %s, want deny", result.Decision)
	}

	result = service.Evaluate(EvaluateInput{
		SubjectType: domainsecurity.SubjectArtifact,
		SubjectID:   "demo",
		Reference:   "demo:latest",
		PolicyMode:  "require_approval",
	})
	if result.Decision != domainsecurity.GateRequireApproval {
		t.Fatalf("decision = %s, want require_approval", result.Decision)
	}
}

func TestEvaluateAndStorePolicyResultCatalog(t *testing.T) {
	ctx := context.Background()
	service := NewService(NewMemoryStore(), fakeScanner{}, nil, nil)
	result, err := service.EvaluateAndStore(ctx, EvaluateInput{
		SubjectType:   domainsecurity.SubjectArtifact,
		SubjectID:     "registry.example.invalid/team/app:latest",
		ProjectID:     "project-a",
		EnvironmentID: "env-prod",
		Reference:     "registry.example.invalid/team/app:latest",
		PolicyID:      "policy-latest",
	})
	if err != nil {
		t.Fatalf("EvaluateAndStore() error = %v", err)
	}
	if result.ID == "" || result.ProjectID != "project-a" || result.EnvironmentID != "env-prod" {
		t.Fatalf("stored policy result missing scope metadata: %#v", result)
	}

	loaded, err := service.GetPolicyResult(ctx, GetPolicyResultInput{ResultID: result.ID, ProjectID: "project-a"})
	if err != nil {
		t.Fatalf("GetPolicyResult() error = %v", err)
	}
	if loaded.ID != result.ID || loaded.PolicyID != "policy-latest" || loaded.Decision != domainsecurity.GateWarn {
		t.Fatalf("loaded policy result = %#v, want %#v", loaded, result)
	}
	if _, err := service.GetPolicyResult(ctx, GetPolicyResultInput{ResultID: result.ID, ProjectID: "project-b"}); err != ErrPolicyResultNotFound {
		t.Fatalf("cross-project GetPolicyResult() error = %v, want ErrPolicyResultNotFound", err)
	}

	results, err := service.ListPolicyResults(ctx, ListPolicyResultsInput{PolicyID: "policy-latest", ProjectID: "project-a", Decision: domainsecurity.GateWarn})
	if err != nil {
		t.Fatalf("ListPolicyResults() error = %v", err)
	}
	if len(results) != 1 || results[0].ID != result.ID {
		t.Fatalf("policy results = %#v, want result %s", results, result.ID)
	}
	results, err = service.ListPolicyResults(ctx, ListPolicyResultsInput{ProjectID: "project-b"})
	if err != nil {
		t.Fatalf("ListPolicyResults(project-b) error = %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("cross-project results leaked: %#v", results)
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

	finding, err := service.GetFinding(ctx, GetFindingInput{FindingID: findings[0].ID})
	if err != nil {
		t.Fatalf("get finding: %v", err)
	}
	if finding.ID != findings[0].ID || finding.Metadata["scanId"] != manifest.Scan.ID {
		t.Fatalf("finding detail = %#v, want id %s scan %s", finding, findings[0].ID, manifest.Scan.ID)
	}
	if _, err := service.GetFinding(ctx, GetFindingInput{FindingID: findings[0].ID, ProjectID: "project-a"}); err != ErrFindingNotFound {
		t.Fatalf("scoped finding lookup error = %v, want ErrFindingNotFound", err)
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
