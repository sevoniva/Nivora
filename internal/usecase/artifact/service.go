package artifact

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	domainartifact "github.com/sevoniva/nivora/internal/domain/artifact"
	"github.com/sevoniva/nivora/internal/domain/audit"
	"github.com/sevoniva/nivora/internal/domain/event"
	"github.com/sevoniva/nivora/internal/domain/release"
	"github.com/sevoniva/nivora/internal/ports/artifact"
	"github.com/sevoniva/nivora/internal/ports/eventbus"
)

const (
	EventReleaseCreated                  = "devops.release.created"
	EventReleaseArtifactBound            = "devops.release.artifact.bound"
	EventArtifactReferenceParsed         = "devops.artifact.reference.parsed"
	EventArtifactDigestResolutionStarted = "devops.artifact.digest.resolution.started"
	EventArtifactDigestResolved          = "devops.artifact.digest.resolved"
	EventArtifactDigestResolutionFailed  = "devops.artifact.digest.resolution.failed"
	EventArtifactResolved                = "devops.artifact.resolved"
	EventArtifactWarningDetected         = "devops.artifact.warning.detected"
	EventArtifactMutableWarning          = "devops.artifact.mutable.warning"
	EventReleaseCanceled                 = "devops.release.canceled"
	EventReleaseStatusUpdated            = "devops.release.status.updated"
)

type Service struct {
	store    Store
	provider artifact.ArtifactProvider
	eventBus eventbus.EventBus
	now      func() time.Time
}

func NewService(store Store, provider artifact.ArtifactProvider, bus eventbus.EventBus) *Service {
	return &Service{store: store, provider: provider, eventBus: bus, now: time.Now}
}

func (s *Service) Inspect(ctx context.Context, reference string, artifactType domainartifact.ArtifactType) (domainartifact.Inspection, error) {
	return s.provider.InspectReference(ctx, reference, artifactType)
}

func (s *Service) Resolve(ctx context.Context, reference string, artifactType domainartifact.ArtifactType) (domainartifact.Resolution, error) {
	inspection, err := s.Inspect(ctx, reference, artifactType)
	if err != nil {
		return domainartifact.Resolution{}, err
	}
	resolution := domainartifact.Resolution{
		Reference: inspection.Reference,
		Digest:    inspection.Reference.Digest,
		Resolved:  inspection.Reference.Digest != "",
		Warnings:  inspection.Warnings,
	}
	if resolution.Resolved {
		resolution.DigestQualifiedReference = domainartifact.DigestQualifiedReference(inspection.Reference, resolution.Digest)
		resolution.ResolvedAt = s.now()
		return resolution, nil
	}
	resolution, err = s.provider.ResolveDigest(ctx, inspection.Reference.Name, inspection.Reference.Normalized)
	if err != nil {
		return domainartifact.Resolution{}, err
	}
	return resolution, nil
}

func (s *Service) TrackArtifact(ctx context.Context, input TrackArtifactInput) (domainartifact.Artifact, error) {
	inspection, err := s.Inspect(ctx, input.Reference, domainartifact.ArtifactType(input.Type))
	if err != nil {
		return domainartifact.Artifact{}, err
	}
	now := s.now()
	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = inspection.Reference.Name
	}
	if name == "" {
		name = inspection.Reference.Repository
	}
	artifact := domainartifact.Artifact{
		ID:         defaultArtifactID(input.ID),
		Type:       inspection.Reference.Type,
		Name:       name,
		Version:    inspection.Reference.Version,
		Reference:  inspection.Reference.Normalized,
		Digest:     inspection.Reference.Digest,
		Registry:   inspection.Reference.Registry,
		Repository: inspection.Reference.Repository,
		Metadata:   cloneMap(input.Metadata),
		CreatedAt:  now,
	}
	if err := s.store.SaveArtifact(ctx, artifact); err != nil {
		return domainartifact.Artifact{}, err
	}
	return s.store.GetArtifact(ctx, artifact.ID)
}

func (s *Service) CreateRelease(ctx context.Context, input CreateReleaseInput) (ReleaseRecord, error) {
	if input.Definition.Metadata.Name == "" {
		return ReleaseRecord{}, fmt.Errorf("release metadata.name is required")
	}
	if input.Definition.Spec.Version == "" {
		return ReleaseRecord{}, fmt.Errorf("release spec.version is required")
	}
	if len(input.Definition.Spec.Artifacts) == 0 {
		return ReleaseRecord{}, fmt.Errorf("release must bind at least one artifact")
	}
	now := s.now()
	rel := release.Release{
		ID:                  newID("rel"),
		Name:                input.Definition.Metadata.Name,
		Version:             input.Definition.Spec.Version,
		ApplicationID:       input.Definition.Spec.Application,
		EnvironmentID:       input.Definition.Spec.Environment,
		SourcePipelineRunID: input.Definition.Spec.SourcePipelineRunID,
		Commit:              input.Definition.Spec.Commit,
		Status:              string(release.ReleaseStatusReady),
		Metadata:            map[string]string{"phase": "2.5"},
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if projectID := strings.TrimSpace(input.ProjectID); projectID != "" {
		rel.Metadata["projectId"] = projectID
	}
	record := ReleaseRecord{Release: rel}
	var pendingEvents []struct {
		eventType string
		message   string
	}
	for _, item := range input.Definition.Spec.Artifacts {
		inspection, err := s.Inspect(ctx, item.Reference, domainartifact.ArtifactType(item.Type))
		if err != nil {
			return ReleaseRecord{}, err
		}
		pendingEvents = append(pendingEvents, struct {
			eventType string
			message   string
		}{EventArtifactReferenceParsed, inspection.Reference.Normalized})
		resolveDigest := input.Definition.Spec.ResolveDigest
		if item.ResolveDigest != nil {
			resolveDigest = *item.ResolveDigest
		}
		requireDigest := input.Definition.Spec.RequireDigest
		if item.RequireDigest != nil {
			requireDigest = *item.RequireDigest
		}
		blockMutable := input.Definition.Spec.BlockMutable
		if item.BlockMutable != nil {
			blockMutable = *item.BlockMutable
		}
		resolution := domainartifact.Resolution{
			Reference: inspection.Reference,
			Digest:    inspection.Reference.Digest,
			Resolved:  inspection.Reference.Digest != "",
			Warnings:  append([]domainartifact.Warning(nil), inspection.Warnings...),
		}
		if resolution.Resolved {
			resolution.DigestQualifiedReference = domainartifact.DigestQualifiedReference(inspection.Reference, resolution.Digest)
			resolution.ResolvedAt = now
		}
		if resolveDigest && !resolution.Resolved {
			pendingEvents = append(pendingEvents, struct {
				eventType string
				message   string
			}{EventArtifactDigestResolutionStarted, inspection.Reference.Normalized})
			resolution, err = s.provider.ResolveDigest(ctx, inspection.Reference.Name, inspection.Reference.Normalized)
			if err != nil {
				pendingEvents = append(pendingEvents, struct {
					eventType string
					message   string
				}{EventArtifactDigestResolutionFailed, err.Error()})
				if requireDigest {
					return ReleaseRecord{}, fmt.Errorf("resolve artifact digest for %q: %w", item.Reference, err)
				}
				resolution = domainartifact.Resolution{
					Reference: inspection.Reference,
					Digest:    inspection.Reference.Digest,
					Resolved:  false,
					Warnings: append(append([]domainartifact.Warning(nil), inspection.Warnings...), domainartifact.Warning{
						Code:    "digest_resolution_failed",
						Message: "digest resolution failed; release artifact remains tag-based",
					}),
				}
			}
		}
		if requireDigest && !resolution.Resolved {
			return ReleaseRecord{}, fmt.Errorf("artifact %q requires a digest but no digest was resolved", item.Reference)
		}
		if blockMutable && isMutableArtifact(inspection) && !resolution.Resolved {
			return ReleaseRecord{}, fmt.Errorf("artifact %q is mutable and blockMutable=true", item.Reference)
		}
		if resolution.Resolved {
			pendingEvents = append(pendingEvents, struct {
				eventType string
				message   string
			}{EventArtifactDigestResolved, resolution.DigestQualifiedReference})
		}
		metadata := cloneMap(item.Metadata)
		if projectID := strings.TrimSpace(input.ProjectID); projectID != "" && metadata["projectId"] == "" {
			if metadata == nil {
				metadata = map[string]string{}
			}
			metadata["projectId"] = projectID
		}
		artifactID := newID("artifact")
		artifact := domainartifact.Artifact{
			ID:             artifactID,
			Type:           inspection.Reference.Type,
			Name:           item.Name,
			Version:        inspection.Reference.Version,
			Reference:      inspection.Reference.Normalized,
			Digest:         resolution.Digest,
			Registry:       inspection.Reference.Registry,
			Repository:     inspection.Reference.Repository,
			MediaType:      resolution.MediaType,
			SizeBytes:      resolution.SizeBytes,
			ManifestSchema: resolution.ManifestSchema,
			Metadata:       metadata,
			CreatedAt:      now,
		}
		bound := release.ReleaseArtifact{
			ID:              newID("relart"),
			ReleaseID:       rel.ID,
			ArtifactID:      artifactID,
			Name:            item.Name,
			Type:            string(inspection.Reference.Type),
			Role:            item.Role,
			Required:        item.Required,
			Reference:       inspection.Reference.Normalized,
			Digest:          resolution.Digest,
			DigestReference: resolution.DigestQualifiedReference,
			MediaType:       resolution.MediaType,
			SizeBytes:       resolution.SizeBytes,
			ManifestSchema:  resolution.ManifestSchema,
			Metadata:        cloneMap(metadata),
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		record.Artifacts = append(record.Artifacts, artifact)
		record.Bindings = append(record.Bindings, bound)
		record.Inspections = append(record.Inspections, inspection)
		record.Resolutions = append(record.Resolutions, resolution)
		for _, warning := range resolution.Warnings {
			record.Warnings = append(record.Warnings, warning)
		}
	}
	if err := s.store.SaveRelease(ctx, record); err != nil {
		return ReleaseRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, rel.ID, EventReleaseCreated, "Release created", input.ActorID, "Release created"); err != nil {
		return ReleaseRecord{}, err
	}
	for _, pending := range pendingEvents {
		_ = s.recordEvent(ctx, rel.ID, pending.eventType, pending.message)
	}
	for _, binding := range record.Bindings {
		_ = s.recordEventAndAudit(ctx, rel.ID, EventReleaseArtifactBound, "Artifact bound to release", input.ActorID, binding.Reference)
		if binding.Digest != "" {
			_ = s.recordEvent(ctx, rel.ID, EventArtifactResolved, binding.Reference)
		}
	}
	for _, warning := range record.Warnings {
		_ = s.recordEvent(ctx, rel.ID, EventArtifactWarningDetected, warning.Message)
		if warning.Code == "mutable_latest_tag" || warning.Code == "tag_without_digest" {
			_ = s.recordEvent(ctx, rel.ID, EventArtifactMutableWarning, warning.Message)
		}
		_ = s.store.AppendAudit(ctx, rel.ID, audit.AuditLog{
			ID:        newID("audit"),
			ActorID:   input.ActorID,
			Action:    "Artifact warning detected",
			Subject:   rel.ID,
			CreatedAt: s.now(),
		})
	}
	return s.store.GetRelease(ctx, rel.ID)
}

func isMutableArtifact(inspection domainartifact.Inspection) bool {
	for _, warning := range inspection.Warnings {
		if warning.Code == "mutable_latest_tag" || warning.Code == "tag_without_digest" || warning.Code == "missing_version_or_digest" {
			return true
		}
	}
	return false
}

func (s *Service) GetRelease(ctx context.Context, id string) (ReleaseRecord, error) {
	return s.store.GetRelease(ctx, id)
}

func (s *Service) ListReleases(ctx context.Context, inputs ...ListReleasesInput) ([]ReleaseRecord, error) {
	records, err := s.store.ListReleases(ctx)
	if err != nil {
		return nil, err
	}
	input := ListReleasesInput{}
	if len(inputs) > 0 {
		input = inputs[0]
	}
	filter := normalizeReleaseFilter(input)
	filtered := make([]ReleaseRecord, 0, len(records))
	for _, record := range records {
		if releaseMatches(record, filter) {
			filtered = append(filtered, cloneReleaseRecord(record))
		}
	}
	return filtered, nil
}

func (s *Service) UpdateReleaseStatus(ctx context.Context, id string, status release.ReleaseStatus, actorID string, reason string) (ReleaseRecord, error) {
	if !release.ValidStatus(status) {
		return ReleaseRecord{}, fmt.Errorf("invalid release status %q", status)
	}
	record, err := s.store.GetRelease(ctx, strings.TrimSpace(id))
	if err != nil {
		return ReleaseRecord{}, err
	}
	current := release.ReleaseStatus(strings.TrimSpace(record.Release.Status))
	if current == status {
		return record, nil
	}
	if release.TerminalStatus(current) {
		return ReleaseRecord{}, ErrReleaseAlreadyTerminal
	}
	now := s.now()
	record.Release.Status = string(status)
	record.Release.UpdatedAt = now
	if record.Release.Metadata == nil {
		record.Release.Metadata = map[string]string{}
	}
	record.Release.Metadata["statusUpdatedAt"] = now.UTC().Format(time.RFC3339)
	record.Release.Metadata["statusReason"] = strings.TrimSpace(reason)
	if actorID = strings.TrimSpace(actorID); actorID != "" {
		record.Release.Metadata["statusUpdatedBy"] = actorID
	}
	if err := s.store.SaveRelease(ctx, record); err != nil {
		return ReleaseRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Release.ID, EventReleaseStatusUpdated, "Release status updated", actorID, strings.TrimSpace(reason)); err != nil {
		return ReleaseRecord{}, err
	}
	return s.store.GetRelease(ctx, record.Release.ID)
}

func (s *Service) CancelRelease(ctx context.Context, id string, actorID string) (ReleaseRecord, error) {
	record, err := s.store.GetRelease(ctx, strings.TrimSpace(id))
	if err != nil {
		return ReleaseRecord{}, err
	}
	switch release.ReleaseStatus(strings.TrimSpace(record.Release.Status)) {
	case release.ReleaseStatusCanceled:
		return record, nil
	case release.ReleaseStatusSucceeded, release.ReleaseStatusFailed:
		return ReleaseRecord{}, ErrReleaseAlreadyTerminal
	}
	now := s.now()
	record.Release.Status = string(release.ReleaseStatusCanceled)
	record.Release.UpdatedAt = now
	if record.Release.Metadata == nil {
		record.Release.Metadata = map[string]string{}
	}
	record.Release.Metadata["canceledAt"] = now.UTC().Format(time.RFC3339)
	if actorID = strings.TrimSpace(actorID); actorID != "" {
		record.Release.Metadata["canceledBy"] = actorID
	}
	if err := s.store.SaveRelease(ctx, record); err != nil {
		return ReleaseRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Release.ID, EventReleaseCanceled, "Release canceled", actorID, "Release canceled"); err != nil {
		return ReleaseRecord{}, err
	}
	return s.store.GetRelease(ctx, record.Release.ID)
}

func (s *Service) ListArtifacts(ctx context.Context, input ListArtifactsInput) ([]domainartifact.Artifact, error) {
	standalone, err := s.store.ListArtifacts(ctx)
	if err != nil {
		return nil, err
	}
	records, err := s.store.ListReleases(ctx)
	if err != nil {
		return nil, err
	}
	filter := normalizeArtifactFilter(input)
	artifacts := make([]domainartifact.Artifact, 0, len(standalone))
	seen := map[string]struct{}{}
	for _, item := range standalone {
		if !artifactMatches(item, filter) {
			continue
		}
		artifacts = append(artifacts, cloneArtifact(item))
		seen[item.ID] = struct{}{}
	}
	for _, record := range records {
		for _, item := range record.Artifacts {
			if !artifactMatches(item, filter) {
				continue
			}
			if _, ok := seen[item.ID]; ok {
				continue
			}
			artifacts = append(artifacts, cloneArtifact(item))
			seen[item.ID] = struct{}{}
		}
	}
	sort.Slice(artifacts, func(i, j int) bool {
		if artifacts[i].CreatedAt.Equal(artifacts[j].CreatedAt) {
			return artifacts[i].ID < artifacts[j].ID
		}
		return artifacts[i].CreatedAt.Before(artifacts[j].CreatedAt)
	})
	return artifacts, nil
}

func (s *Service) GetArtifact(ctx context.Context, id string) (domainartifact.Artifact, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return domainartifact.Artifact{}, ErrArtifactNotFound
	}
	if artifact, err := s.store.GetArtifact(ctx, id); err == nil {
		return cloneArtifact(artifact), nil
	}
	artifacts, err := s.ListArtifacts(ctx, ListArtifactsInput{})
	if err != nil {
		return domainartifact.Artifact{}, err
	}
	for _, item := range artifacts {
		if item.ID == id {
			return cloneArtifact(item), nil
		}
	}
	return domainartifact.Artifact{}, fmt.Errorf("%w: %s", ErrArtifactNotFound, id)
}

func defaultArtifactID(id string) string {
	if strings.TrimSpace(id) != "" {
		return strings.TrimSpace(id)
	}
	return newID("artifact")
}

func cloneMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func (s *Service) ArtifactReleases(ctx context.Context, artifactID string) ([]ArtifactReleaseBinding, error) {
	artifact, err := s.GetArtifact(ctx, artifactID)
	if err != nil {
		return nil, err
	}
	records, err := s.store.ListReleases(ctx)
	if err != nil {
		return nil, err
	}
	bindings := make([]ArtifactReleaseBinding, 0)
	for _, record := range records {
		for _, binding := range record.Bindings {
			if binding.ArtifactID != artifact.ID {
				continue
			}
			bindings = append(bindings, ArtifactReleaseBinding{
				Release: record.Release,
				Binding: binding,
			})
		}
	}
	sort.Slice(bindings, func(i, j int) bool {
		if bindings[i].Release.CreatedAt.Equal(bindings[j].Release.CreatedAt) {
			return bindings[i].Release.ID < bindings[j].Release.ID
		}
		return bindings[i].Release.CreatedAt.Before(bindings[j].Release.CreatedAt)
	})
	return bindings, nil
}

func (s *Service) ReleaseArtifacts(ctx context.Context, id string) ([]release.ReleaseArtifact, error) {
	record, err := s.store.GetRelease(ctx, id)
	if err != nil {
		return nil, err
	}
	return append([]release.ReleaseArtifact(nil), record.Bindings...), nil
}

func normalizeArtifactFilter(input ListArtifactsInput) ListArtifactsInput {
	return ListArtifactsInput{
		Type:          strings.TrimSpace(strings.ToLower(input.Type)),
		Name:          strings.TrimSpace(strings.ToLower(input.Name)),
		Registry:      strings.TrimSpace(strings.ToLower(input.Registry)),
		Repository:    strings.TrimSpace(strings.ToLower(input.Repository)),
		Digest:        strings.TrimSpace(input.Digest),
		Reference:     strings.TrimSpace(strings.ToLower(input.Reference)),
		ProjectID:     strings.TrimSpace(input.ProjectID),
		EnvironmentID: strings.TrimSpace(input.EnvironmentID),
	}
}

func normalizeReleaseFilter(input ListReleasesInput) ListReleasesInput {
	return ListReleasesInput{
		ProjectID:     strings.TrimSpace(input.ProjectID),
		EnvironmentID: strings.TrimSpace(input.EnvironmentID),
		ApplicationID: strings.TrimSpace(input.ApplicationID),
		Status:        strings.TrimSpace(strings.ToLower(input.Status)),
	}
}

func releaseMatches(record ReleaseRecord, filter ListReleasesInput) bool {
	if filter.ProjectID != "" && releaseProjectID(record) != filter.ProjectID {
		return false
	}
	if filter.EnvironmentID != "" && releaseEnvironmentID(record) != filter.EnvironmentID {
		return false
	}
	if filter.ApplicationID != "" && record.Release.ApplicationID != filter.ApplicationID {
		return false
	}
	if filter.Status != "" && strings.ToLower(record.Release.Status) != filter.Status {
		return false
	}
	return true
}

func releaseProjectID(record ReleaseRecord) string {
	if record.Release.Metadata != nil && record.Release.Metadata["projectId"] != "" {
		return record.Release.Metadata["projectId"]
	}
	for _, binding := range record.Bindings {
		if binding.Metadata != nil && binding.Metadata["projectId"] != "" {
			return binding.Metadata["projectId"]
		}
	}
	for _, artifact := range record.Artifacts {
		if artifact.Metadata != nil && artifact.Metadata["projectId"] != "" {
			return artifact.Metadata["projectId"]
		}
	}
	return ""
}

func releaseEnvironmentID(record ReleaseRecord) string {
	if record.Release.EnvironmentID != "" {
		return record.Release.EnvironmentID
	}
	if record.Release.Metadata != nil && record.Release.Metadata["environmentId"] != "" {
		return record.Release.Metadata["environmentId"]
	}
	for _, binding := range record.Bindings {
		if binding.Metadata != nil && binding.Metadata["environmentId"] != "" {
			return binding.Metadata["environmentId"]
		}
	}
	for _, artifact := range record.Artifacts {
		if artifact.Metadata != nil && artifact.Metadata["environmentId"] != "" {
			return artifact.Metadata["environmentId"]
		}
	}
	return ""
}

func artifactMatches(item domainartifact.Artifact, filter ListArtifactsInput) bool {
	if filter.ProjectID != "" && item.Metadata["projectId"] != filter.ProjectID {
		return false
	}
	if filter.EnvironmentID != "" && item.Metadata["environmentId"] != filter.EnvironmentID {
		return false
	}
	if filter.Type != "" && strings.ToLower(string(item.Type)) != filter.Type {
		return false
	}
	if filter.Name != "" && !strings.Contains(strings.ToLower(item.Name), filter.Name) {
		return false
	}
	if filter.Registry != "" && strings.ToLower(item.Registry) != filter.Registry {
		return false
	}
	if filter.Repository != "" && !strings.Contains(strings.ToLower(item.Repository), filter.Repository) {
		return false
	}
	if filter.Digest != "" && item.Digest != filter.Digest {
		return false
	}
	if filter.Reference != "" && !strings.Contains(strings.ToLower(item.Reference), filter.Reference) {
		return false
	}
	return true
}

func cloneArtifact(item domainartifact.Artifact) domainartifact.Artifact {
	if item.Metadata != nil {
		out := make(map[string]string, len(item.Metadata))
		for key, value := range item.Metadata {
			out[key] = value
		}
		item.Metadata = out
	}
	return item
}

func (s *Service) recordEventAndAudit(ctx context.Context, subject string, eventType string, action string, actorID string, message string) error {
	if err := s.recordEvent(ctx, subject, eventType, message); err != nil {
		return err
	}
	return s.store.AppendAudit(ctx, subject, audit.AuditLog{
		ID:        newID("audit"),
		ActorID:   actorID,
		Action:    action,
		Subject:   subject,
		CreatedAt: s.now(),
	})
}

func (s *Service) recordEvent(ctx context.Context, subject string, eventType string, message string) error {
	evt := event.Event{
		SpecVersion:     "1.0",
		ID:              newID("evt"),
		Type:            eventType,
		Source:          "nivora/release",
		Subject:         subject,
		Time:            s.now(),
		DataContentType: "application/json",
		Data:            map[string]any{"message": message},
	}
	if err := s.eventBus.Publish(ctx, evt); err != nil {
		return err
	}
	return s.store.AppendEvent(ctx, subject, evt)
}

func newID(prefix string) string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(b[:])
}
