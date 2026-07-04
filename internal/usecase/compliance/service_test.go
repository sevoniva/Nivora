package compliance

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	domainapproval "github.com/sevoniva/nivora/internal/domain/approval"
	domainartifact "github.com/sevoniva/nivora/internal/domain/artifact"
	"github.com/sevoniva/nivora/internal/domain/audit"
	domaincompliance "github.com/sevoniva/nivora/internal/domain/compliance"
	"github.com/sevoniva/nivora/internal/domain/event"
	domainnotification "github.com/sevoniva/nivora/internal/domain/notification"
	domainsecurity "github.com/sevoniva/nivora/internal/domain/security"
	portartifact "github.com/sevoniva/nivora/internal/ports/artifact"
	portexecutor "github.com/sevoniva/nivora/internal/ports/executor"
	portnotification "github.com/sevoniva/nivora/internal/ports/notification"
	"github.com/sevoniva/nivora/internal/ports/policy"
	portsecurity "github.com/sevoniva/nivora/internal/ports/security"
	approvalusecase "github.com/sevoniva/nivora/internal/usecase/approval"
	artifactusecase "github.com/sevoniva/nivora/internal/usecase/artifact"
	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
	releaseusecase "github.com/sevoniva/nivora/internal/usecase/releaseorchestration"
	securityusecase "github.com/sevoniva/nivora/internal/usecase/security"
)

func TestEvidenceBundleGeneration(t *testing.T) {
	pipelines := pipelineusecase.NewService(pipelineusecase.NewMemoryStore(), fakeRunner{}, fakeEventBus{})
	result, err := pipelines.CreateAndRun(context.Background(), pipelineusecase.CreateRunInput{Definition: pipelineusecase.Definition{
		APIVersion: "nivora.io/v1alpha1",
		Kind:       "Pipeline",
		Metadata:   pipelineusecase.Metadata{Name: "audit-demo"},
		Spec: pipelineusecase.Spec{Stages: []pipelineusecase.Stage{{
			Name: "build",
			Jobs: []pipelineusecase.Job{{
				Name:     "job",
				Executor: "shell",
				Steps:    []pipelineusecase.Step{{Name: "step", Run: `printf "ok"`}},
			}},
		}}},
	}})
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	service := newTestComplianceService(pipelines)
	bundle, err := service.EvidenceBundle(context.Background(), EvidenceInput{SubjectType: "pipelineRun", SubjectID: result.Record.Run.ID})
	if err != nil {
		t.Fatalf("evidence bundle: %v", err)
	}
	if len(bundle.Audits) == 0 || len(bundle.Events) == 0 || len(bundle.LogReferences) == 0 {
		t.Fatalf("bundle missing evidence: %#v", bundle)
	}
	stored, err := service.store.GetEvidenceBundle(context.Background(), bundle.ID)
	if err != nil {
		t.Fatalf("stored evidence bundle: %v", err)
	}
	if stored.SubjectID != bundle.SubjectID || len(stored.Audits) != len(bundle.Audits) {
		t.Fatalf("stored evidence mismatch: %#v", stored)
	}
	if markdown := service.ExportMarkdown(bundle); !strings.Contains(markdown, "Evidence Bundle") {
		t.Fatalf("markdown export missing title: %s", markdown)
	}
}

func TestRetentionPolicy(t *testing.T) {
	service := newTestComplianceService(nil)
	policy, err := service.SetRetentionPolicy(context.Background(), RetentionInput{ScopeType: "project", ScopeID: "project-a", LogDays: 14, AuditDays: 730})
	if err != nil {
		t.Fatalf("set retention: %v", err)
	}
	if policy.LogDays != 14 || policy.AuditDays != 730 || !policy.ImmutableAudit {
		t.Fatalf("unexpected policy: %#v", policy)
	}
	got, err := service.RetentionPolicy(context.Background(), "project", "project-a")
	if err != nil {
		t.Fatalf("get retention: %v", err)
	}
	if got.LogDays != 14 || got.AuditDays != 730 {
		t.Fatalf("retention not persisted: %#v", got)
	}
	stored, err := service.store.GetRetentionPolicy(context.Background(), "project", "project-a")
	if err != nil {
		t.Fatalf("stored retention policy: %v", err)
	}
	if stored.LogDays != 14 || stored.AuditDays != 730 {
		t.Fatalf("stored retention policy mismatch: %#v", stored)
	}
}

func TestRetentionRunPreviewAndEnforceEvidenceOnly(t *testing.T) {
	service := newTestComplianceService(nil)
	now := time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	if _, err := service.SetRetentionPolicy(context.Background(), RetentionInput{ScopeType: "project", ScopeID: "project-a", EvidenceDays: 30, AuditDays: 30}); err != nil {
		t.Fatalf("set retention: %v", err)
	}
	oldBundle := domaincompliance.EvidenceBundle{ID: "evb-old", SubjectType: "release", SubjectID: "rel-old", ScopeType: "project", ScopeID: "project-a", Summary: "old", GeneratedAt: now.AddDate(0, 0, -60)}
	newBundle := domaincompliance.EvidenceBundle{ID: "evb-new", SubjectType: "release", SubjectID: "rel-new", ScopeType: "project", ScopeID: "project-a", Summary: "new", GeneratedAt: now.AddDate(0, 0, -5)}
	otherScope := domaincompliance.EvidenceBundle{ID: "evb-other", SubjectType: "release", SubjectID: "rel-other", ScopeType: "project", ScopeID: "project-b", Summary: "other", GeneratedAt: now.AddDate(0, 0, -60)}
	for _, bundle := range []domaincompliance.EvidenceBundle{oldBundle, newBundle, otherScope} {
		if err := service.store.SaveEvidenceBundle(context.Background(), bundle); err != nil {
			t.Fatalf("save evidence %s: %v", bundle.ID, err)
		}
	}
	if err := service.store.AppendAuditLog(context.Background(), audit.AuditLog{ID: "audit-old", ActorID: "ops", Action: "old", Subject: "release/rel-old", ScopeType: "project", ScopeID: "project-a", CreatedAt: now.AddDate(0, 0, -60)}); err != nil {
		t.Fatalf("append audit: %v", err)
	}

	preview, err := service.RunRetention(context.Background(), RetentionRunInput{ScopeType: "project", ScopeID: "project-a", DryRun: true, ActorID: "ops"})
	if err != nil {
		t.Fatalf("preview retention: %v", err)
	}
	evidencePreview := retentionTarget(t, preview.Targets, domaincompliance.RetentionTargetEvidence)
	if !preview.DryRun || evidencePreview.Candidates != 1 || evidencePreview.Deleted != 0 {
		t.Fatalf("unexpected evidence preview: result=%#v target=%#v", preview, evidencePreview)
	}
	auditPreview := retentionTarget(t, preview.Targets, domaincompliance.RetentionTargetAudit)
	if !auditPreview.Immutable || auditPreview.Candidates != 1 || auditPreview.Deleted != 0 {
		t.Fatalf("unexpected audit preview: %#v", auditPreview)
	}
	if _, err := service.store.GetEvidenceBundle(context.Background(), oldBundle.ID); err != nil {
		t.Fatalf("dry-run should not delete old evidence: %v", err)
	}

	enforced, err := service.RunRetention(context.Background(), RetentionRunInput{ScopeType: "project", ScopeID: "project-a", Confirm: true, ActorID: "ops"})
	if err != nil {
		t.Fatalf("enforce retention: %v", err)
	}
	evidenceEnforced := retentionTarget(t, enforced.Targets, domaincompliance.RetentionTargetEvidence)
	if enforced.DryRun || evidenceEnforced.Candidates != 1 || evidenceEnforced.Deleted != 1 {
		t.Fatalf("unexpected evidence enforcement: result=%#v target=%#v", enforced, evidenceEnforced)
	}
	if _, err := service.store.GetEvidenceBundle(context.Background(), oldBundle.ID); err != ErrEvidenceBundleNotFound {
		t.Fatalf("old evidence lookup err = %v, want not found", err)
	}
	for _, id := range []string{newBundle.ID, otherScope.ID} {
		if _, err := service.store.GetEvidenceBundle(context.Background(), id); err != nil {
			t.Fatalf("evidence %s should remain: %v", id, err)
		}
	}
}

func TestEvidenceBundleRedactsSecrets(t *testing.T) {
	artifacts := artifactusecase.NewService(artifactusecase.NewMemoryStore(), fakeArtifactProvider{}, fakeEventBus{})
	record, err := artifacts.CreateRelease(context.Background(), artifactusecase.CreateReleaseInput{Definition: artifactusecase.ReleaseDefinition{
		APIVersion: "nivora.io/v1alpha1",
		Kind:       "Release",
		Metadata:   artifactusecase.ReleaseMetadata{Name: "redaction"},
		Spec: artifactusecase.ReleaseSpec{
			Version: "1.0.0",
			Artifacts: []artifactusecase.ReleaseArtifactSpec{{
				Name:      "demo",
				Type:      "image",
				Reference: "registry.example.com/demo/app:1.0.0",
				Metadata:  map[string]string{"token": "placeholder-sensitive-token"},
			}},
		},
	}})
	if err != nil {
		t.Fatalf("create release: %v", err)
	}
	service := NewService(nil, newTestDeploymentService(), artifacts, newTestReleaseService(artifacts, newTestDeploymentService()), newTestSecurityService(), approvalusecase.NewService(approvalusecase.NewMemoryStore(), fakeNotificationProvider{}, fakeEventBus{}))
	bundle, err := service.EvidenceBundle(context.Background(), EvidenceInput{SubjectType: "release", SubjectID: record.Release.ID})
	if err != nil {
		t.Fatalf("release evidence: %v", err)
	}
	body, err := json.Marshal(bundle)
	if err != nil {
		t.Fatalf("marshal bundle: %v", err)
	}
	if strings.Contains(string(body), "placeholder-sensitive-token") {
		t.Fatalf("evidence leaked secret-like value: %s", string(body))
	}
}

func retentionTarget(t *testing.T, targets []domaincompliance.RetentionTargetResult, name string) domaincompliance.RetentionTargetResult {
	t.Helper()
	for _, target := range targets {
		if target.Target == name {
			return target
		}
	}
	t.Fatalf("retention target %s not found in %#v", name, targets)
	return domaincompliance.RetentionTargetResult{}
}

func TestReleaseEvidenceBundleIncludesExecutionDeploymentAndDigest(t *testing.T) {
	service := newTestComplianceService(nil)
	fixed := time.Date(2026, 7, 4, 9, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return fixed }

	execution, err := service.releases.Deploy(context.Background(), releaseusecase.DeployInput{
		Definition: releaseEvidenceDefinition(false),
		ActorID:    "release-operator",
	})
	if err != nil {
		t.Fatalf("deploy release: %v", err)
	}
	if execution.Execution.Status != releaseusecase.ExecutionSucceeded {
		t.Fatalf("execution status = %s", execution.Execution.Status)
	}

	bundle, err := service.EvidenceBundle(context.Background(), EvidenceInput{SubjectType: "release", SubjectID: execution.Release.ID})
	if err != nil {
		t.Fatalf("release evidence: %v", err)
	}
	if bundle.GeneratedBy != "nivora" || !strings.HasPrefix(bundle.Digest, "sha256:") {
		t.Fatalf("bundle metadata missing generatedBy/digest: %#v", bundle)
	}
	if len(bundle.ReleaseExecutions) != 1 || len(bundle.ReleasePlans) != 1 {
		t.Fatalf("release execution evidence missing: executions=%d plans=%d", len(bundle.ReleaseExecutions), len(bundle.ReleasePlans))
	}
	if len(bundle.DeploymentRuns) != 1 || len(bundle.DeploymentPlans) == 0 || len(bundle.LogReferences) == 0 {
		t.Fatalf("deployment evidence missing: deployments=%d plans=%d logs=%d", len(bundle.DeploymentRuns), len(bundle.DeploymentPlans), len(bundle.LogReferences))
	}
	if len(bundle.PolicyResults) == 0 || len(bundle.Events) == 0 || len(bundle.Audits) == 0 {
		t.Fatalf("policy/event/audit evidence missing: policies=%d events=%d audits=%d", len(bundle.PolicyResults), len(bundle.Events), len(bundle.Audits))
	}
	if got := bundle.SubjectSummary["executionCount"]; got != 1 {
		t.Fatalf("subject summary executionCount = %#v, want 1", got)
	}

	again, err := service.EvidenceBundle(context.Background(), EvidenceInput{SubjectType: "release", SubjectID: execution.Release.ID})
	if err != nil {
		t.Fatalf("second release evidence: %v", err)
	}
	if again.Digest != bundle.Digest {
		t.Fatalf("evidence digest should be stable: %s != %s", again.Digest, bundle.Digest)
	}
	markdown := service.ExportMarkdown(bundle)
	if !strings.Contains(markdown, "Digest") || !strings.Contains(markdown, "Deployment runs") {
		t.Fatalf("markdown summary missing v2 fields: %s", markdown)
	}
}

func TestReleaseEvidenceBundleIncludesApprovalGate(t *testing.T) {
	service := newTestComplianceService(nil)
	service.releases.WithGovernance(testReleaseGovernance{})

	execution, err := service.releases.Deploy(context.Background(), releaseusecase.DeployInput{
		Definition: releaseEvidenceDefinition(true),
		ActorID:    "release-operator",
	})
	if err != nil {
		t.Fatalf("deploy release waiting approval: %v", err)
	}
	if execution.Execution.Status != releaseusecase.ExecutionWaitingApproval {
		t.Fatalf("execution status = %s", execution.Execution.Status)
	}

	bundle, err := service.EvidenceBundle(context.Background(), EvidenceInput{SubjectType: "release", SubjectID: execution.Release.ID})
	if err != nil {
		t.Fatalf("release evidence: %v", err)
	}
	if len(bundle.ReleaseExecutions) != 1 || len(bundle.Approvals) != 1 {
		t.Fatalf("approval evidence missing: executions=%d approvals=%d", len(bundle.ReleaseExecutions), len(bundle.Approvals))
	}
}

func releaseEvidenceDefinition(approvalRequired bool) releaseusecase.Definition {
	return releaseusecase.Definition{
		APIVersion: "nivora.io/v1alpha1",
		Kind:       "ReleaseOrchestration",
		Metadata:   releaseusecase.Metadata{Name: "evidence-release"},
		Spec: releaseusecase.Spec{
			Environment:      "dev",
			Strategy:         releaseusecase.StrategySequential,
			ApprovalRequired: approvalRequired,
			Release: artifactusecase.ReleaseDefinition{
				APIVersion: "nivora.io/v1alpha1",
				Kind:       "Release",
				Metadata:   artifactusecase.ReleaseMetadata{Name: "evidence-demo"},
				Spec: artifactusecase.ReleaseSpec{
					Version:     "1.0.0",
					Application: "demo",
					Environment: "dev",
					Artifacts: []artifactusecase.ReleaseArtifactSpec{{
						Name:      "demo",
						Type:      "image",
						Required:  true,
						Reference: "registry.example.com/demo/app@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					}},
				},
			},
			Targets: []releaseusecase.TargetSpec{{
				Name:  "dev-yaml",
				Type:  "kubernetes-yaml",
				Order: 1,
				Deployment: deploymentusecase.Definition{
					APIVersion: "nivora.io/v1alpha1",
					Kind:       "Deployment",
					Metadata:   deploymentusecase.Metadata{Name: "evidence-deployment"},
					Spec: deploymentusecase.Spec{
						Application: "demo",
						Environment: "dev",
						Target: deploymentusecase.Target{
							Type:      "kubernetes-yaml",
							Name:      "dev-yaml",
							Namespace: "default",
						},
						Artifacts: []deploymentusecase.Artifact{{
							Name:      "demo",
							Type:      "image",
							Reference: "registry.example.com/demo/app@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
						}},
						Manifests: []string{"../../../examples/yaml/deployment.yaml"},
						Options:   deploymentusecase.Options{DryRun: true, Apply: false},
					},
				},
			}},
		},
	}
}

func newTestComplianceService(pipelines *pipelineusecase.Service) *Service {
	if pipelines == nil {
		pipelines = pipelineusecase.NewService(pipelineusecase.NewMemoryStore(), fakeRunner{}, fakeEventBus{})
	}
	artifacts := artifactusecase.NewService(artifactusecase.NewMemoryStore(), fakeArtifactProvider{}, fakeEventBus{})
	deployments := newTestDeploymentService()
	security := newTestSecurityService()
	approvals := approvalusecase.NewService(approvalusecase.NewMemoryStore(), fakeNotificationProvider{}, fakeEventBus{})
	releases := newTestReleaseService(artifacts, deployments)
	return NewService(pipelines, deployments, artifacts, releases, security, approvals)
}

type testReleaseGovernance struct{}

func (testReleaseGovernance) RequestApproval(ctx context.Context, subjectType string, subjectID string, environmentID string, requestedBy string, reason string) (domainapproval.ApprovalRequest, error) {
	if err := ctx.Err(); err != nil {
		return domainapproval.ApprovalRequest{}, err
	}
	return domainapproval.ApprovalRequest{
		ID:               "appr-evidence",
		SubjectType:      subjectType,
		SubjectID:        subjectID,
		EnvironmentID:    environmentID,
		RequiredByPolicy: true,
		Status:           domainapproval.StatusPending,
		RequestedBy:      requestedBy,
		Reason:           reason,
	}, nil
}

func newTestDeploymentService() *deploymentusecase.Service {
	return deploymentusecase.NewService(deploymentusecase.NewMemoryStore(), deploymentusecase.NewStaticManifestRenderer(), fakeManifestClient{}, allowAllPolicy{}, fakeEventBus{})
}

func newTestReleaseService(artifacts *artifactusecase.Service, deployments *deploymentusecase.Service) *releaseusecase.Service {
	return releaseusecase.NewService(releaseusecase.NewMemoryStore(), artifacts, deployments, allowAllPolicy{}, fakeEventBus{})
}

func newTestSecurityService() *securityusecase.Service {
	return securityusecase.NewService(securityusecase.NewMemoryStore(), fakeSecurityScanner{}, fakeSignatureVerifier{}, fakeEventBus{})
}

type allowAllPolicy struct{}

func (allowAllPolicy) Evaluate(ctx context.Context, request policy.Request) (policy.Result, error) {
	return policy.Result{Allowed: true}, ctx.Err()
}

type fakeRunner struct{}

func (fakeRunner) ID() string { return "test-runner" }

func (fakeRunner) RunShellStep(ctx context.Context, jobRunID string, command string, timeout time.Duration) (portexecutor.Result, error) {
	if err := ctx.Err(); err != nil {
		return portexecutor.Result{}, err
	}
	return portexecutor.Result{Stdout: "ok"}, nil
}

type fakeEventBus struct{}

func (fakeEventBus) Publish(ctx context.Context, evt event.Event) error { return ctx.Err() }

func (fakeEventBus) Subscribe(ctx context.Context, eventType string) (<-chan event.Event, error) {
	ch := make(chan event.Event)
	close(ch)
	return ch, nil
}

type fakeArtifactProvider struct{}

func (fakeArtifactProvider) ValidateCredential(ctx context.Context, credential portartifact.CredentialRef) error {
	return ctx.Err()
}

func (fakeArtifactProvider) GetArtifact(ctx context.Context, name string, reference string) (domainartifact.Artifact, error) {
	if err := ctx.Err(); err != nil {
		return domainartifact.Artifact{}, err
	}
	return domainartifact.Artifact{Name: name, Reference: reference}, nil
}

func (fakeArtifactProvider) ListArtifacts(ctx context.Context, repository string) ([]domainartifact.Artifact, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, nil
}

func (fakeArtifactProvider) ResolveDigest(ctx context.Context, name string, reference string) (domainartifact.Resolution, error) {
	inspection, err := domainartifact.InspectReference(reference, domainartifact.ArtifactTypeImage)
	if err != nil {
		return domainartifact.Resolution{}, err
	}
	if err := ctx.Err(); err != nil {
		return domainartifact.Resolution{}, err
	}
	return domainartifact.Resolution{Reference: inspection.Reference, Warnings: inspection.Warnings}, nil
}

func (fakeArtifactProvider) InspectReference(ctx context.Context, reference string, artifactType domainartifact.ArtifactType) (domainartifact.Inspection, error) {
	if err := ctx.Err(); err != nil {
		return domainartifact.Inspection{}, err
	}
	return domainartifact.InspectReference(reference, artifactType)
}

func (fakeArtifactProvider) Capabilities() portartifact.Capabilities {
	return portartifact.Capabilities{}
}

type fakeManifestClient struct{}

func (fakeManifestClient) ServerDryRun(ctx context.Context, request deploymentusecase.ManifestRequest) (deploymentusecase.KubernetesDryRunResult, error) {
	if err := ctx.Err(); err != nil {
		return deploymentusecase.KubernetesDryRunResult{}, err
	}
	return deploymentusecase.KubernetesDryRunResult{Mode: "test", Message: "dry-run ok", Resources: request.Plan.Resources}, nil
}

func (fakeManifestClient) Apply(ctx context.Context, request deploymentusecase.ManifestRequest) (deploymentusecase.KubernetesApplyResult, error) {
	if err := ctx.Err(); err != nil {
		return deploymentusecase.KubernetesApplyResult{}, err
	}
	return deploymentusecase.KubernetesApplyResult{Mode: "test", Message: "apply ok", Resources: request.Plan.Resources}, nil
}

func (fakeManifestClient) WatchRollout(ctx context.Context, request deploymentusecase.ManifestRequest) (deploymentusecase.RolloutResult, error) {
	if err := ctx.Err(); err != nil {
		return deploymentusecase.RolloutResult{}, err
	}
	return deploymentusecase.RolloutResult{Mode: "test", Message: "rollout ok", Resources: request.Plan.Resources}, nil
}

func (fakeManifestClient) Rollback(ctx context.Context, request deploymentusecase.ManifestRequest) (deploymentusecase.KubernetesRollbackResult, error) {
	if err := ctx.Err(); err != nil {
		return deploymentusecase.KubernetesRollbackResult{}, err
	}
	return deploymentusecase.KubernetesRollbackResult{Mode: "test", Message: "rollback ok", Resources: request.Plan.Resources}, nil
}

type fakeSecurityScanner struct{}

func (fakeSecurityScanner) ScanArtifact(ctx context.Context, request portsecurity.ScanRequest) (portsecurity.ScanResult, error) {
	return fakeScan(ctx)
}

func (fakeSecurityScanner) ScanManifest(ctx context.Context, request portsecurity.ScanRequest) (portsecurity.ScanResult, error) {
	return fakeScan(ctx)
}

func (fakeSecurityScanner) ScanDeploymentPlan(ctx context.Context, request portsecurity.ScanRequest) (portsecurity.ScanResult, error) {
	return fakeScan(ctx)
}

func (fakeSecurityScanner) GetCapabilities(ctx context.Context) ([]portsecurity.Capability, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return []portsecurity.Capability{portsecurity.CapabilityVulnerability}, nil
}

func fakeScan(ctx context.Context) (portsecurity.ScanResult, error) {
	if err := ctx.Err(); err != nil {
		return portsecurity.ScanResult{}, err
	}
	return portsecurity.ScanResult{Scanner: "fake"}, nil
}

type fakeSignatureVerifier struct{}

func (fakeSignatureVerifier) VerifyArtifactSignature(ctx context.Context, subject string) (domainsecurity.SignatureCheck, error) {
	if err := ctx.Err(); err != nil {
		return domainsecurity.SignatureCheck{}, err
	}
	return domainsecurity.SignatureCheck{Subject: subject, Verifier: "fake", Result: "skipped"}, nil
}

type fakeNotificationProvider struct{}

var _ portnotification.Provider = fakeNotificationProvider{}

func (fakeNotificationProvider) Send(ctx context.Context, notification domainnotification.Notification) error {
	return ctx.Err()
}
