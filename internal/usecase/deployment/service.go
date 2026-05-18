package deployment

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	domainartifact "github.com/sevoniva/nivora/internal/domain/artifact"
	"github.com/sevoniva/nivora/internal/domain/audit"
	domaindeployment "github.com/sevoniva/nivora/internal/domain/deployment"
	"github.com/sevoniva/nivora/internal/domain/environment"
	"github.com/sevoniva/nivora/internal/domain/event"
	"github.com/sevoniva/nivora/internal/domain/release"
	"github.com/sevoniva/nivora/internal/ports/eventbus"
	"github.com/sevoniva/nivora/internal/ports/policy"
)

const (
	EventDeploymentCreated           = "devops.deployment.created"
	EventDeploymentPlanning          = "devops.deployment.planning"
	EventDeploymentPrecheckStarted   = "devops.deployment.precheck.started"
	EventDeploymentPrecheckCompleted = "devops.deployment.precheck.completed"
	EventDeploymentDryRunStarted     = "devops.deployment.dryrun.started"
	EventDeploymentDryRunCompleted   = "devops.deployment.dryrun.completed"
	EventDeploymentDryRunFailed      = "devops.deployment.dryrun.failed"
	EventDeploymentApplyStarted      = "devops.deployment.apply.started"
	EventDeploymentApplyCompleted    = "devops.deployment.apply.completed"
	EventDeploymentApplyFailed       = "devops.deployment.apply.failed"
	EventDeploymentVerifyStarted     = "devops.deployment.verify.started"
	EventDeploymentVerifyCompleted   = "devops.deployment.verify.completed"
	EventDeploymentVerifyFailed      = "devops.deployment.verify.failed"
	EventDeploymentSucceeded         = "devops.deployment.succeeded"
	EventDeploymentFailed            = "devops.deployment.failed"
	EventDeploymentCanceled          = "devops.deployment.canceled"
)

type KubernetesManifestClient interface {
	ServerDryRun(ctx context.Context, request ManifestRequest) (KubernetesDryRunResult, error)
	Apply(ctx context.Context, request ManifestRequest) (KubernetesApplyResult, error)
	WatchRollout(ctx context.Context, request ManifestRequest) (RolloutResult, error)
}

type ManifestClient = KubernetesManifestClient

type Service struct {
	store    Store
	renderer ManifestRenderer
	client   ManifestClient
	policy   policy.Engine
	eventBus eventbus.EventBus
	now      func() time.Time
}

func NewService(store Store, renderer ManifestRenderer, client ManifestClient, policyEngine policy.Engine, bus eventbus.EventBus) *Service {
	return &Service{
		store:    store,
		renderer: renderer,
		client:   client,
		policy:   policyEngine,
		eventBus: bus,
		now:      time.Now,
	}
}

func (s *Service) CreateAndRun(ctx context.Context, input CreateRunInput) (CreateRunResult, error) {
	if err := input.Definition.Validate(); err != nil {
		return CreateRunResult{}, err
	}
	if input.Definition.Spec.Options.Apply && !input.AllowApply {
		return CreateRunResult{}, fmt.Errorf("deployment apply requires explicit confirmation")
	}
	record := s.newRecord(input.Definition)
	if err := s.store.Save(ctx, record); err != nil {
		return CreateRunResult{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventDeploymentCreated, "DeploymentRun created", input.ActorID, string(record.Run.Status), "DeploymentRun created"); err != nil {
		return CreateRunResult{}, err
	}
	record, err := s.process(ctx, record, input.ActorID)
	if err != nil {
		return CreateRunResult{}, err
	}
	return CreateRunResult{Record: record}, nil
}

func (s *Service) Plan(ctx context.Context, input CreateRunInput) (CreateRunResult, error) {
	if err := input.Definition.Validate(); err != nil {
		return CreateRunResult{}, err
	}
	record := s.newRecord(input.Definition)
	if err := transitionDeploymentRun(&record.Run, domaindeployment.DeploymentRunPlanning, s.now(), ""); err != nil {
		return CreateRunResult{}, err
	}
	documents, err := s.renderer.Render(ctx, input.Definition.Spec.Manifests, input.Definition.Spec.Target.Namespace)
	if err != nil {
		return CreateRunResult{}, err
	}
	record.Plan = s.buildPlan(record.Run.ID, input.Definition, documents)
	record.Rollback = s.rollbackBaseline(record.Run.ID, input.Definition, record.Plan.Resources)
	record.Logs = append(record.Logs, s.logChunk(record.Run.ID, "system", fmt.Sprintf("rendered %d manifest document(s)", len(documents)), int64(len(record.Logs)+1)))
	return CreateRunResult{Record: record}, nil
}

func (s *Service) process(ctx context.Context, record RunRecord, actorID string) (RunRecord, error) {
	if err := transitionDeploymentRun(&record.Run, domaindeployment.DeploymentRunPlanning, s.now(), ""); err != nil {
		return RunRecord{}, err
	}
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventDeploymentPlanning, "DeploymentRun planned", actorID, string(record.Run.Status), "DeploymentRun planning started"); err != nil {
		return RunRecord{}, err
	}
	record, _ = s.store.Get(ctx, record.Run.ID)
	documents, err := s.renderer.Render(ctx, record.Definition.Spec.Manifests, record.Definition.Spec.Target.Namespace)
	if err != nil {
		return s.fail(ctx, record, actorID, err.Error())
	}
	record.Plan = s.buildPlan(record.Run.ID, record.Definition, documents)
	record.Rollback = s.rollbackBaseline(record.Run.ID, record.Definition, record.Plan.Resources)
	record.Logs = append(record.Logs, s.logChunk(record.Run.ID, "system", fmt.Sprintf("rendered %d manifest document(s)", len(documents)), int64(len(record.Logs)+1)))

	if err := transitionDeploymentRun(&record.Run, domaindeployment.DeploymentRunPreChecking, s.now(), ""); err != nil {
		return RunRecord{}, err
	}
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	if record, err = s.recordRuntimeEvent(ctx, record, EventDeploymentPrecheckStarted, string(record.Run.Status), "Deployment policy pre-check started"); err != nil {
		return RunRecord{}, err
	}
	policyResult, err := s.policy.Evaluate(ctx, policy.Request{
		Subject: record.Run.ID,
		Action:  "deployment.dryrun",
		Context: map[string]any{"targetType": record.Run.TargetType, "apply": record.Plan.Apply},
	})
	if err != nil {
		return s.fail(ctx, record, actorID, err.Error())
	}
	record.Policy = policyResult
	if !policyResult.Allowed {
		reason := "deployment policy denied"
		if len(policyResult.Reasons) > 0 {
			reason = policyResult.Reasons[0]
		}
		record, _ = s.recordRuntimeEvent(ctx, record, EventDeploymentPrecheckCompleted, string(domaindeployment.DeploymentRunFailed), reason)
		return s.fail(ctx, record, actorID, reason)
	}
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventDeploymentPrecheckCompleted, "Deployment policy pre-check completed", actorID, string(record.Run.Status), "Deployment policy pre-check allowed"); err != nil {
		return RunRecord{}, err
	}
	record, _ = s.store.Get(ctx, record.Run.ID)
	if err := transitionDeploymentRun(&record.Run, domaindeployment.DeploymentRunVerifying, s.now(), ""); err != nil {
		return RunRecord{}, err
	}
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	if record, err = s.recordRuntimeEvent(ctx, record, EventDeploymentDryRunStarted, string(record.Run.Status), "Deployment dry-run started"); err != nil {
		return RunRecord{}, err
	}
	request := ManifestRequest{Plan: record.Plan, Documents: documents, TimeoutSeconds: record.Plan.TimeoutSeconds}
	dryRunResult, err := s.client.ServerDryRun(ctx, request)
	if err != nil {
		record, _ = s.recordRuntimeEvent(ctx, record, EventDeploymentDryRunFailed, string(domaindeployment.DeploymentRunFailed), err.Error())
		return s.fail(ctx, record, actorID, err.Error())
	}
	record.DryRun = dryRunResult
	record.Logs = append(record.Logs, s.logChunk(record.Run.ID, "system", dryRunResult.Message, int64(len(record.Logs)+1)))
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventDeploymentDryRunCompleted, "Deployment dry-run completed", actorID, string(record.Run.Status), "Deployment dry-run completed"); err != nil {
		return RunRecord{}, err
	}
	record, _ = s.store.Get(ctx, record.Run.ID)
	if !record.Plan.Apply {
		record.Logs = append(record.Logs, s.logChunk(record.Run.ID, "system", "dry-run validation completed", int64(len(record.Logs)+1)))
		if err := transitionDeploymentRun(&record.Run, domaindeployment.DeploymentRunSucceeded, s.now(), "dry-run deployment run succeeded"); err != nil {
			return RunRecord{}, err
		}
		if err := s.store.Save(ctx, record); err != nil {
			return RunRecord{}, err
		}
		if err := s.recordEventAndAudit(ctx, record.Run.ID, EventDeploymentSucceeded, "DeploymentRun succeeded", actorID, string(record.Run.Status), "DeploymentRun dry-run succeeded"); err != nil {
			return RunRecord{}, err
		}
		return s.store.Get(ctx, record.Run.ID)
	}
	if err := transitionDeploymentRun(&record.Run, domaindeployment.DeploymentRunDeploying, s.now(), ""); err != nil {
		return RunRecord{}, err
	}
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventDeploymentApplyStarted, "Deployment apply started", actorID, string(record.Run.Status), "Deployment apply started"); err != nil {
		return RunRecord{}, err
	}
	record, _ = s.store.Get(ctx, record.Run.ID)
	applyResult, err := s.client.Apply(ctx, request)
	if err != nil {
		record, _ = s.recordRuntimeEvent(ctx, record, EventDeploymentApplyFailed, string(domaindeployment.DeploymentRunFailed), err.Error())
		return s.fail(ctx, record, actorID, err.Error())
	}
	record.Apply = applyResult
	record.Logs = append(record.Logs, s.logChunk(record.Run.ID, "system", applyResult.Message, int64(len(record.Logs)+1)))
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventDeploymentApplyCompleted, "Deployment apply completed", actorID, string(record.Run.Status), "Deployment apply completed"); err != nil {
		return RunRecord{}, err
	}
	record, _ = s.store.Get(ctx, record.Run.ID)
	if record.Plan.Wait {
		if err := transitionDeploymentRun(&record.Run, domaindeployment.DeploymentRunVerifying, s.now(), ""); err != nil {
			return RunRecord{}, err
		}
		if err := s.store.Save(ctx, record); err != nil {
			return RunRecord{}, err
		}
		if err := s.recordEventAndAudit(ctx, record.Run.ID, EventDeploymentVerifyStarted, "Deployment verification started", actorID, string(record.Run.Status), "Deployment rollout verification started"); err != nil {
			return RunRecord{}, err
		}
		record, _ = s.store.Get(ctx, record.Run.ID)
		rolloutResult, err := s.client.WatchRollout(ctx, request)
		if err != nil {
			record, _ = s.recordRuntimeEvent(ctx, record, EventDeploymentVerifyFailed, string(domaindeployment.DeploymentRunFailed), err.Error())
			return s.fail(ctx, record, actorID, err.Error())
		}
		record.Rollout = rolloutResult
		record.Logs = append(record.Logs, s.logChunk(record.Run.ID, "system", rolloutResult.Message, int64(len(record.Logs)+1)))
		if err := s.store.Save(ctx, record); err != nil {
			return RunRecord{}, err
		}
		if err := s.recordEventAndAudit(ctx, record.Run.ID, EventDeploymentVerifyCompleted, "Deployment verification completed", actorID, string(record.Run.Status), "Deployment rollout verification completed"); err != nil {
			return RunRecord{}, err
		}
		record, _ = s.store.Get(ctx, record.Run.ID)
	}
	if err := transitionDeploymentRun(&record.Run, domaindeployment.DeploymentRunSucceeded, s.now(), "kubernetes YAML apply completed"); err != nil {
		return RunRecord{}, err
	}
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventDeploymentSucceeded, "DeploymentRun succeeded", actorID, string(record.Run.Status), "DeploymentRun apply succeeded"); err != nil {
		return RunRecord{}, err
	}
	return s.store.Get(ctx, record.Run.ID)
}

func (s *Service) fail(ctx context.Context, record RunRecord, actorID string, reason string) (RunRecord, error) {
	_ = transitionDeploymentRun(&record.Run, domaindeployment.DeploymentRunFailed, s.now(), reason)
	record.Logs = append(record.Logs, s.logChunk(record.Run.ID, "system", reason, int64(len(record.Logs)+1)))
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventDeploymentFailed, "DeploymentRun failed", actorID, string(record.Run.Status), reason); err != nil {
		return RunRecord{}, err
	}
	return s.store.Get(ctx, record.Run.ID)
}

func (s *Service) Get(ctx context.Context, id string) (RunRecord, error) { return s.store.Get(ctx, id) }
func (s *Service) List(ctx context.Context) ([]RunRecord, error)         { return s.store.List(ctx) }
func (s *Service) Logs(ctx context.Context, id string) ([]event.LogChunk, error) {
	return s.store.Logs(ctx, id)
}
func (s *Service) Events(ctx context.Context, id string) ([]event.Event, error) {
	return s.store.Events(ctx, id)
}

func (s *Service) Resources(ctx context.Context, id string) ([]ManifestResourceSummary, error) {
	record, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return append([]ManifestResourceSummary(nil), record.Plan.Resources...), nil
}

func (s *Service) Timeline(ctx context.Context, id string) ([]TimelineEntry, error) {
	events, err := s.Events(ctx, id)
	if err != nil {
		return nil, err
	}
	timeline := make([]TimelineEntry, 0, len(events))
	for _, evt := range events {
		entry := TimelineEntry{Type: evt.Type, Time: evt.Time, Subject: evt.Subject}
		if status, ok := evt.Data["status"].(string); ok {
			entry.Status = status
		}
		if message, ok := evt.Data["message"].(string); ok {
			entry.Message = message
		}
		timeline = append(timeline, entry)
	}
	return timeline, nil
}

func (s *Service) Cancel(ctx context.Context, id string, actorID string) (RunRecord, error) {
	record, err := s.store.Get(ctx, id)
	if err != nil {
		return RunRecord{}, err
	}
	if isTerminalDeploymentStatus(record.Run.Status) {
		return record, ErrRunTerminal
	}
	if err := transitionDeploymentRun(&record.Run, domaindeployment.DeploymentRunCanceled, s.now(), "canceled by request"); err != nil {
		return RunRecord{}, err
	}
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, id, EventDeploymentCanceled, "DeploymentRun canceled", actorID, string(record.Run.Status), "DeploymentRun canceled"); err != nil {
		return RunRecord{}, err
	}
	return s.store.Get(ctx, id)
}

func (s *Service) newRecord(def Definition) RunRecord {
	now := s.now()
	releaseID := newID("rel")
	runID := newID("drun")
	envID := newID("env")
	targetID := newID("target")
	artifactRefs := make([]string, 0, len(def.Spec.Artifacts))
	artifacts := make([]release.ReleaseArtifact, 0, len(def.Spec.Artifacts))
	for _, artifact := range def.Spec.Artifacts {
		artifactRefs = append(artifactRefs, artifact.Reference)
		artifacts = append(artifacts, release.ReleaseArtifact{
			ID:        newID("artifact"),
			ReleaseID: releaseID,
			Name:      artifact.Name,
			Type:      artifact.Type,
			Role:      "deployment",
			Required:  true,
			Reference: artifact.Reference,
			Digest:    artifact.Digest,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}
	return RunRecord{
		Definition: def,
		Release: release.Release{
			ID:            releaseID,
			Name:          def.Metadata.Name,
			ApplicationID: def.Spec.Application,
			Version:       def.Metadata.Name,
			Status:        "Created",
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		Artifacts: artifacts,
		Environment: environment.Environment{
			ID:        envID,
			Name:      def.Spec.Environment,
			CreatedAt: now,
			UpdatedAt: now,
		},
		Target: environment.ReleaseTarget{
			ID:            targetID,
			EnvironmentID: envID,
			Name:          def.Spec.Target.Name,
			TargetType:    def.Spec.Target.Type,
			Context:       def.Spec.Target.Context,
			Namespace:     def.Spec.Target.Namespace,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		Run: domaindeployment.DeploymentRun{
			ID:                  runID,
			ReleaseID:           releaseID,
			ApplicationID:       def.Spec.Application,
			EnvironmentID:       envID,
			ReleaseTargetID:     targetID,
			TargetType:          def.Spec.Target.Type,
			Status:              domaindeployment.DeploymentRunCreated,
			ArtifactReferences:  artifactRefs,
			ManifestSnapshotRef: "memory://" + runID + "/manifests",
			CreatedAt:           now,
			UpdatedAt:           now,
		},
	}
}

func (s *Service) buildPlan(runID string, def Definition, docs []ManifestDocument) DeploymentPlan {
	artifacts := make([]string, 0, len(def.Spec.Artifacts))
	artifactDetails := make([]domainartifact.Inspection, 0, len(def.Spec.Artifacts))
	for _, artifact := range def.Spec.Artifacts {
		reference := artifact.Reference
		if artifact.Digest != "" && !containsDigest(reference) {
			reference = reference + "@" + artifact.Digest
		}
		inspection, err := domainartifact.InspectReference(reference, domainartifact.ArtifactType(artifact.Type))
		if err != nil {
			warnings := []string{"live cluster diff is not implemented in Phase 2.2", fmt.Sprintf("artifact %q could not be inspected: %v", artifact.Name, err)}
			return DeploymentPlan{
				DeploymentRunID: runID,
				TargetType:      def.Spec.Target.Type,
				TargetContext:   def.Spec.Target.Context,
				Namespace:       def.Spec.Target.Namespace,
				Artifacts:       []string{artifact.Reference},
				DryRun:          def.Spec.Options.dryRunOnly(),
				Apply:           def.Spec.Options.Apply,
				Wait:            def.Spec.Options.Wait,
				TimeoutSeconds:  def.Spec.Options.TimeoutSeconds,
				Actions:         []string{"render manifests", "validate manifests", "policy pre-check", "server-side dry-run"},
				Warnings:        warnings,
				DiffSummary:     "artifact inspection failed",
			}
		}
		artifacts = append(artifacts, inspection.Reference.Normalized)
		artifactDetails = append(artifactDetails, inspection)
	}
	resources := make([]ManifestResourceSummary, 0, len(docs))
	for _, doc := range docs {
		resources = append(resources, doc.Resource)
	}
	manifestImages := ExtractManifestImages(docs)
	warnings := []string{"live cluster diff is not implemented in Phase 2.2"}
	for _, detail := range artifactDetails {
		for _, warning := range detail.Warnings {
			warnings = append(warnings, fmt.Sprintf("artifact %s: %s", detail.Reference.Normalized, warning.Message))
		}
	}
	warnings = append(warnings, verifyManifestImages(def.Spec.Artifacts, artifactDetails, manifestImages)...)
	if def.Spec.Options.Apply {
		warnings = append(warnings, "apply requested; Phase 2.2 apply requires explicit confirmation and uses the configured manifest client")
	}
	actions := []string{"render manifests", "validate manifests", "policy pre-check", "server-side dry-run"}
	if def.Spec.Options.Apply {
		actions = append(actions, "apply manifests")
	}
	if def.Spec.Options.Apply && def.Spec.Options.Wait {
		actions = append(actions, "rollout verification")
	}
	return DeploymentPlan{
		DeploymentRunID: runID,
		TargetType:      def.Spec.Target.Type,
		TargetContext:   def.Spec.Target.Context,
		Namespace:       def.Spec.Target.Namespace,
		ManifestCount:   len(docs),
		Resources:       resources,
		Artifacts:       artifacts,
		ArtifactDetails: artifactDetails,
		ManifestImages:  manifestImages,
		DryRun:          def.Spec.Options.dryRunOnly(),
		Apply:           def.Spec.Options.Apply,
		Wait:            def.Spec.Options.Wait,
		TimeoutSeconds:  def.Spec.Options.TimeoutSeconds,
		Actions:         actions,
		Warnings:        warnings,
		DiffSummary:     fmt.Sprintf("desired state contains %d manifest resource(s); live diff is not available in Phase 2.2", len(docs)),
	}
}

func verifyManifestImages(specArtifacts []Artifact, artifactDetails []domainartifact.Inspection, images []ManifestImage) []string {
	if len(images) == 0 {
		return nil
	}
	expectedByContainer := make(map[string]domainartifact.Inspection)
	expectedByName := make(map[string]domainartifact.Inspection)
	for i, artifact := range specArtifacts {
		if i >= len(artifactDetails) {
			continue
		}
		detail := artifactDetails[i]
		if artifact.Target.ImageName != "" {
			expectedByContainer[artifact.Target.ImageName] = detail
		}
		expectedByName[detail.Reference.Name] = detail
	}
	var warnings []string
	for _, image := range images {
		inspection, err := domainartifact.InspectReference(image.Image, domainartifact.ArtifactTypeImage)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("manifest image %q could not be inspected: %v", image.Image, err))
			continue
		}
		for _, warning := range inspection.Warnings {
			warnings = append(warnings, fmt.Sprintf("manifest image %s: %s", image.Image, warning.Message))
		}
		expected, ok := expectedByContainer[image.Container]
		if !ok {
			expected, ok = expectedByName[inspection.Reference.Name]
		}
		if !ok {
			warnings = append(warnings, fmt.Sprintf("manifest image %s is not bound to a release artifact", image.Image))
			continue
		}
		if expected.Reference.Normalized != inspection.Reference.Normalized {
			warnings = append(warnings, fmt.Sprintf("manifest image %s differs from bound artifact %s", image.Image, expected.Reference.Normalized))
		}
	}
	return warnings
}

func containsDigest(reference string) bool {
	return strings.Contains(reference, "@sha256:")
}

func (s *Service) rollbackBaseline(runID string, def Definition, resources []ManifestResourceSummary) *domaindeployment.RollbackRecord {
	refs := make([]string, 0, len(resources))
	for _, resource := range resources {
		refs = append(refs, fmt.Sprintf("%s/%s/%s", resource.Kind, resource.Namespace, resource.Name))
	}
	now := s.now()
	return &domaindeployment.RollbackRecord{
		ID:                  newID("rollback"),
		DeploymentRunID:     runID,
		Strategy:            "manifest-snapshot",
		Status:              "placeholder",
		TargetType:          def.Spec.Target.Type,
		TargetName:          def.Spec.Target.Name,
		ManifestSnapshotRef: "memory://" + runID + "/previous-manifests",
		ResourceRefs:        refs,
		Reason:              "rollback execution is not implemented in Phase 2.2",
		CreatedAt:           now,
		UpdatedAt:           now,
	}
}

func (s *Service) logChunk(runID string, stream string, content string, sequence int64) event.LogChunk {
	return event.LogChunk{
		ID:              newID("log"),
		DeploymentRunID: runID,
		Stream:          stream,
		Sequence:        sequence,
		Content:         content,
		CreatedAt:       s.now(),
	}
}

func validateManifestRequest(request ManifestRequest) error {
	if request.Plan.DeploymentRunID == "" {
		return fmt.Errorf("deploymentRunId is required")
	}
	if len(request.Documents) == 0 {
		return fmt.Errorf("at least one manifest document is required")
	}
	return nil
}

func (s *Service) recordEventAndAudit(ctx context.Context, runID string, eventType string, action string, actorID string, status string, message string) error {
	if err := s.recordEvent(ctx, runID, eventType, status, message); err != nil {
		return err
	}
	return s.store.AppendAudit(ctx, runID, audit.AuditLog{
		ID:        newID("audit"),
		ActorID:   actorID,
		Action:    action,
		Subject:   runID,
		CreatedAt: s.now(),
	})
}

func (s *Service) recordEvent(ctx context.Context, runID string, eventType string, status string, message string) error {
	evt := event.Event{
		SpecVersion:     "1.0",
		ID:              newID("evt"),
		Type:            eventType,
		Source:          "nivora/deployment",
		Subject:         runID,
		Time:            s.now(),
		DataContentType: "application/json",
		Data: map[string]any{
			"status":  status,
			"message": message,
		},
	}
	if err := s.eventBus.Publish(ctx, evt); err != nil {
		return err
	}
	return s.store.AppendEvent(ctx, runID, evt)
}

func (s *Service) recordRuntimeEvent(ctx context.Context, record RunRecord, eventType string, status string, message string) (RunRecord, error) {
	evt := event.Event{
		SpecVersion:     "1.0",
		ID:              newID("evt"),
		Type:            eventType,
		Source:          "nivora/deployment",
		Subject:         record.Run.ID,
		Time:            s.now(),
		DataContentType: "application/json",
		Data: map[string]any{
			"status":  status,
			"message": message,
		},
	}
	if err := s.eventBus.Publish(ctx, evt); err != nil {
		return RunRecord{}, err
	}
	record.Events = append(record.Events, evt)
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	return record, nil
}

func newID(prefix string) string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(b[:])
}
