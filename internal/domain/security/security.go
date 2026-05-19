package security

import "time"

type SubjectType string

const (
	SubjectArtifact       SubjectType = "artifact"
	SubjectManifest       SubjectType = "manifest"
	SubjectDeploymentPlan SubjectType = "deployment_plan"
	SubjectRelease        SubjectType = "release"
)

type ScanStatus string

const (
	ScanPending   ScanStatus = "Pending"
	ScanRunning   ScanStatus = "Running"
	ScanSucceeded ScanStatus = "Succeeded"
	ScanFailed    ScanStatus = "Failed"
	ScanCanceled  ScanStatus = "Canceled"
)

type Severity string

const (
	SeverityUnknown  Severity = "Unknown"
	SeverityLow      Severity = "Low"
	SeverityMedium   Severity = "Medium"
	SeverityHigh     Severity = "High"
	SeverityCritical Severity = "Critical"
)

type FindingCategory string

const (
	CategoryVulnerability    FindingCategory = "vulnerability"
	CategorySecret           FindingCategory = "secret"
	CategoryMisconfiguration FindingCategory = "misconfiguration"
	CategoryLicense          FindingCategory = "license"
	CategorySignature        FindingCategory = "signature"
	CategorySBOM             FindingCategory = "sbom"
	CategoryPolicy           FindingCategory = "policy"
)

type GateDecision string

const (
	GateAllow           GateDecision = "allow"
	GateDeny            GateDecision = "deny"
	GateWarn            GateDecision = "warn"
	GateRequireApproval GateDecision = "require_approval"
)

type SecurityScan struct {
	ID          string              `json:"id"`
	SubjectType SubjectType         `json:"subjectType"`
	SubjectID   string              `json:"subjectId"`
	Scanner     string              `json:"scanner"`
	Status      ScanStatus          `json:"status"`
	StartedAt   *time.Time          `json:"startedAt,omitempty"`
	FinishedAt  *time.Time          `json:"finishedAt,omitempty"`
	Summary     SecurityScanSummary `json:"summary"`
	Findings    []SecurityFinding   `json:"findings,omitempty"`
	CreatedAt   time.Time           `json:"createdAt"`
}

type SecurityScanSummary struct {
	Total    int `json:"total"`
	Low      int `json:"low"`
	Medium   int `json:"medium"`
	High     int `json:"high"`
	Critical int `json:"critical"`
}

type SecurityFinding struct {
	ID               string            `json:"id"`
	Severity         Severity          `json:"severity"`
	Category         FindingCategory   `json:"category"`
	Target           string            `json:"target"`
	Title            string            `json:"title"`
	Description      string            `json:"description,omitempty"`
	PackageName      string            `json:"packageName,omitempty"`
	InstalledVersion string            `json:"installedVersion,omitempty"`
	FixedVersion     string            `json:"fixedVersion,omitempty"`
	Reference        string            `json:"reference,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

type PolicyResult struct {
	ID          string            `json:"id"`
	PolicyID    string            `json:"policyId,omitempty"`
	SubjectType SubjectType       `json:"subjectType"`
	SubjectID   string            `json:"subjectId"`
	Decision    GateDecision      `json:"decision"`
	Reason      string            `json:"reason,omitempty"`
	Findings    []SecurityFinding `json:"findings,omitempty"`
	EvaluatedAt time.Time         `json:"evaluatedAt"`
}

type SignatureCheck struct {
	Subject             string       `json:"subject"`
	Verifier            string       `json:"verifier"`
	Status              ScanStatus   `json:"status"`
	KeyRef              string       `json:"keyRef,omitempty"`
	CertificateIdentity string       `json:"certificateIdentity,omitempty"`
	Issuer              string       `json:"issuer,omitempty"`
	Result              GateDecision `json:"result"`
	Warnings            []string     `json:"warnings,omitempty"`
}

type SBOMRef struct {
	ArtifactID  string    `json:"artifactId"`
	Format      string    `json:"format"`
	StorageRef  string    `json:"storageRef"`
	Digest      string    `json:"digest,omitempty"`
	GeneratedBy string    `json:"generatedBy,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

func Summarize(findings []SecurityFinding) SecurityScanSummary {
	summary := SecurityScanSummary{Total: len(findings)}
	for _, finding := range findings {
		switch finding.Severity {
		case SeverityLow:
			summary.Low++
		case SeverityMedium:
			summary.Medium++
		case SeverityHigh:
			summary.High++
		case SeverityCritical:
			summary.Critical++
		}
	}
	return summary
}
