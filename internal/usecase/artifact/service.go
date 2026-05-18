package artifact

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	domainartifact "github.com/sevoniva/nivora/internal/domain/artifact"
	"github.com/sevoniva/nivora/internal/domain/audit"
	"github.com/sevoniva/nivora/internal/domain/event"
	"github.com/sevoniva/nivora/internal/domain/release"
	"github.com/sevoniva/nivora/internal/ports/artifact"
	"github.com/sevoniva/nivora/internal/ports/eventbus"
)

const (
	EventReleaseCreated          = "devops.release.created"
	EventReleaseArtifactBound    = "devops.release.artifact.bound"
	EventArtifactResolved        = "devops.artifact.resolved"
	EventArtifactWarningDetected = "devops.artifact.warning.detected"
)

var ErrReleaseNotFound = errors.New("release not found")

type Service struct {
	store    *MemoryStore
	provider artifact.ArtifactProvider
	eventBus eventbus.EventBus
	now      func() time.Time
}

func NewService(store *MemoryStore, provider artifact.ArtifactProvider, bus eventbus.EventBus) *Service {
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
		return resolution, nil
	}
	digest, err := s.provider.ResolveDigest(ctx, inspection.Reference.Name, inspection.Reference.Normalized)
	if err == nil && digest != "" {
		resolution.Digest = digest
		resolution.Resolved = true
		resolution.Reference.Digest = digest
		resolution.Reference.Immutable = true
		return resolution, nil
	}
	return resolution, nil
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
		Status:              "Created",
		Metadata:            map[string]string{"phase": "2.2"},
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	record := ReleaseRecord{Release: rel}
	for _, item := range input.Definition.Spec.Artifacts {
		inspection, err := s.Inspect(ctx, item.Reference, domainartifact.ArtifactType(item.Type))
		if err != nil {
			return ReleaseRecord{}, err
		}
		resolution, err := s.Resolve(ctx, item.Reference, domainartifact.ArtifactType(item.Type))
		if err != nil {
			return ReleaseRecord{}, err
		}
		artifactID := newID("artifact")
		artifact := domainartifact.Artifact{
			ID:         artifactID,
			Type:       inspection.Reference.Type,
			Name:       item.Name,
			Version:    inspection.Reference.Version,
			Reference:  inspection.Reference.Normalized,
			Digest:     resolution.Digest,
			Registry:   inspection.Reference.Registry,
			Repository: inspection.Reference.Repository,
			Metadata:   item.Metadata,
			CreatedAt:  now,
		}
		bound := release.ReleaseArtifact{
			ID:         newID("relart"),
			ReleaseID:  rel.ID,
			ArtifactID: artifactID,
			Name:       item.Name,
			Type:       string(inspection.Reference.Type),
			Role:       item.Role,
			Required:   item.Required,
			Reference:  inspection.Reference.Normalized,
			Digest:     resolution.Digest,
			Metadata:   item.Metadata,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		record.Artifacts = append(record.Artifacts, artifact)
		record.Bindings = append(record.Bindings, bound)
		record.Inspections = append(record.Inspections, inspection)
		for _, warning := range inspection.Warnings {
			record.Warnings = append(record.Warnings, warning)
		}
	}
	if err := s.store.SaveRelease(ctx, record); err != nil {
		return ReleaseRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, rel.ID, EventReleaseCreated, "Release created", input.ActorID, "Release created"); err != nil {
		return ReleaseRecord{}, err
	}
	for _, binding := range record.Bindings {
		_ = s.recordEventAndAudit(ctx, rel.ID, EventReleaseArtifactBound, "Artifact bound to release", input.ActorID, binding.Reference)
		if binding.Digest != "" {
			_ = s.recordEvent(ctx, rel.ID, EventArtifactResolved, binding.Reference)
		}
	}
	for _, warning := range record.Warnings {
		_ = s.recordEvent(ctx, rel.ID, EventArtifactWarningDetected, warning.Message)
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

func (s *Service) GetRelease(ctx context.Context, id string) (ReleaseRecord, error) {
	return s.store.GetRelease(ctx, id)
}

func (s *Service) ListReleases(ctx context.Context) ([]ReleaseRecord, error) {
	return s.store.ListReleases(ctx)
}

func (s *Service) ReleaseArtifacts(ctx context.Context, id string) ([]release.ReleaseArtifact, error) {
	record, err := s.store.GetRelease(ctx, id)
	if err != nil {
		return nil, err
	}
	return append([]release.ReleaseArtifact(nil), record.Bindings...), nil
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

type MemoryStore struct {
	mu       sync.RWMutex
	releases map[string]ReleaseRecord
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{releases: make(map[string]ReleaseRecord)}
}

func (s *MemoryStore) SaveRelease(ctx context.Context, record ReleaseRecord) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.releases[record.Release.ID] = cloneReleaseRecord(record)
	return nil
}

func (s *MemoryStore) GetRelease(ctx context.Context, id string) (ReleaseRecord, error) {
	select {
	case <-ctx.Done():
		return ReleaseRecord{}, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.releases[id]
	if !ok {
		return ReleaseRecord{}, ErrReleaseNotFound
	}
	return cloneReleaseRecord(record), nil
}

func (s *MemoryStore) ListReleases(ctx context.Context) ([]ReleaseRecord, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	records := make([]ReleaseRecord, 0, len(s.releases))
	for _, record := range s.releases {
		records = append(records, cloneReleaseRecord(record))
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].Release.CreatedAt.Before(records[j].Release.CreatedAt)
	})
	return records, nil
}

func (s *MemoryStore) AppendEvent(ctx context.Context, subject string, evt event.Event) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.releases[subject]
	if !ok {
		return ErrReleaseNotFound
	}
	record.Events = append(record.Events, evt)
	s.releases[subject] = cloneReleaseRecord(record)
	return nil
}

func (s *MemoryStore) AppendAudit(ctx context.Context, subject string, entry audit.AuditLog) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.releases[subject]
	if !ok {
		return ErrReleaseNotFound
	}
	record.Audits = append(record.Audits, entry)
	s.releases[subject] = cloneReleaseRecord(record)
	return nil
}

func cloneReleaseRecord(record ReleaseRecord) ReleaseRecord {
	record.Artifacts = append([]domainartifact.Artifact(nil), record.Artifacts...)
	record.Bindings = append([]release.ReleaseArtifact(nil), record.Bindings...)
	record.Inspections = append([]domainartifact.Inspection(nil), record.Inspections...)
	record.Warnings = append([]domainartifact.Warning(nil), record.Warnings...)
	record.Events = append([]event.Event(nil), record.Events...)
	record.Audits = append([]audit.AuditLog(nil), record.Audits...)
	return record
}
