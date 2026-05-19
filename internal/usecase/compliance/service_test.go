package compliance

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	domainartifact "github.com/sevoniva/nivora/internal/domain/artifact"
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
