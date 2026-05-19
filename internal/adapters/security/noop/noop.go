package noop

import (
	"context"
	"time"

	domainsecurity "github.com/sevoniva/nivora/internal/domain/security"
	portsecurity "github.com/sevoniva/nivora/internal/ports/security"
)

type Scanner struct {
	Name     string
	Findings []domainsecurity.SecurityFinding
	Err      error
}

func New() Scanner {
	return Scanner{Name: "noop-security-scanner"}
}

func Fake(findings []domainsecurity.SecurityFinding) Scanner {
	return Scanner{Name: "fake-security-scanner", Findings: findings}
}

func (s Scanner) ScanArtifact(ctx context.Context, request portsecurity.ScanRequest) (portsecurity.ScanResult, error) {
	return s.result(ctx)
}

func (s Scanner) ScanManifest(ctx context.Context, request portsecurity.ScanRequest) (portsecurity.ScanResult, error) {
	return s.result(ctx)
}

func (s Scanner) ScanDeploymentPlan(ctx context.Context, request portsecurity.ScanRequest) (portsecurity.ScanResult, error) {
	return s.result(ctx)
}

func (s Scanner) GetCapabilities(ctx context.Context) ([]portsecurity.Capability, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	return []portsecurity.Capability{
		portsecurity.CapabilityVulnerability,
		portsecurity.CapabilitySecret,
		portsecurity.CapabilityMisconfiguration,
		portsecurity.CapabilityLicense,
		portsecurity.CapabilitySBOM,
		portsecurity.CapabilitySignature,
	}, nil
}

func (s Scanner) result(ctx context.Context) (portsecurity.ScanResult, error) {
	select {
	case <-ctx.Done():
		return portsecurity.ScanResult{}, ctx.Err()
	default:
	}
	if s.Err != nil {
		return portsecurity.ScanResult{}, s.Err
	}
	name := s.Name
	if name == "" {
		name = "noop-security-scanner"
	}
	findings := append([]domainsecurity.SecurityFinding(nil), s.Findings...)
	for i := range findings {
		if findings[i].ID == "" {
			findings[i].ID = "finding-" + string(findings[i].Severity) + "-" + string(findings[i].Category)
		}
		if findings[i].Metadata == nil {
			findings[i].Metadata = map[string]string{"generatedBy": name, "phase": "3.0"}
		}
	}
	return portsecurity.ScanResult{Scanner: name, Findings: findings}, nil
}

type SignatureVerifier struct{}

func (SignatureVerifier) VerifyArtifactSignature(ctx context.Context, subject string) (domainsecurity.SignatureCheck, error) {
	select {
	case <-ctx.Done():
		return domainsecurity.SignatureCheck{}, ctx.Err()
	default:
	}
	return domainsecurity.SignatureCheck{
		Subject:  subject,
		Verifier: "noop-signature-verifier",
		Status:   domainsecurity.ScanSucceeded,
		Result:   domainsecurity.GateAllow,
		Warnings: []string{"signature verification is a Phase 3.0 noop foundation"},
	}, nil
}

func SBOM(subject string) domainsecurity.SBOMRef {
	return domainsecurity.SBOMRef{
		ArtifactID:  subject,
		Format:      "unknown",
		StorageRef:  "memory://" + subject + "/sbom",
		GeneratedBy: "noop-sbom-generator",
		CreatedAt:   time.Now(),
	}
}
