package security

import (
	"context"

	domainsecurity "github.com/sevoniva/nivora/internal/domain/security"
)

type Capability string

const (
	CapabilityVulnerability    Capability = "vulnerability"
	CapabilitySecret           Capability = "secret"
	CapabilityMisconfiguration Capability = "misconfiguration"
	CapabilityLicense          Capability = "license"
	CapabilitySBOM             Capability = "sbom"
	CapabilitySignature        Capability = "signature"
)

type ScanRequest struct {
	SubjectType domainsecurity.SubjectType `json:"subjectType"`
	SubjectID   string                     `json:"subjectId"`
	Reference   string                     `json:"reference,omitempty"`
	Content     string                     `json:"content,omitempty"`
	Metadata    map[string]string          `json:"metadata,omitempty"`
}

type ScanResult struct {
	Scanner  string                           `json:"scanner"`
	Findings []domainsecurity.SecurityFinding `json:"findings,omitempty"`
	Warnings []string                         `json:"warnings,omitempty"`
}

type SecurityScanner interface {
	ScanArtifact(ctx context.Context, request ScanRequest) (ScanResult, error)
	ScanManifest(ctx context.Context, request ScanRequest) (ScanResult, error)
	ScanDeploymentPlan(ctx context.Context, request ScanRequest) (ScanResult, error)
	GetCapabilities(ctx context.Context) ([]Capability, error)
}

type SignatureVerifier interface {
	VerifyArtifactSignature(ctx context.Context, subject string) (domainsecurity.SignatureCheck, error)
}
