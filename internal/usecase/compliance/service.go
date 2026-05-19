package compliance

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sevoniva/nivora/internal/domain/audit"
	domaincompliance "github.com/sevoniva/nivora/internal/domain/compliance"
	"github.com/sevoniva/nivora/internal/domain/event"
	"github.com/sevoniva/nivora/internal/infra/crypto"
	approvalusecase "github.com/sevoniva/nivora/internal/usecase/approval"
	artifactusecase "github.com/sevoniva/nivora/internal/usecase/artifact"
	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
	releaseusecase "github.com/sevoniva/nivora/internal/usecase/releaseorchestration"
	securityusecase "github.com/sevoniva/nivora/internal/usecase/security"
)

type Service struct {
	pipelines   *pipelineusecase.Service
	deployments *deploymentusecase.Service
	artifacts   *artifactusecase.Service
	releases    *releaseusecase.Service
	security    *securityusecase.Service
	approvals   *approvalusecase.Service
	now         func() time.Time

	mu        sync.RWMutex
	retention map[string]domaincompliance.RetentionPolicy
}

func NewService(pipelines *pipelineusecase.Service, deployments *deploymentusecase.Service, artifacts *artifactusecase.Service, releases *releaseusecase.Service, security *securityusecase.Service, approvals *approvalusecase.Service) *Service {
	return &Service{
		pipelines:   pipelines,
		deployments: deployments,
		artifacts:   artifacts,
		releases:    releases,
		security:    security,
		approvals:   approvals,
		now:         time.Now,
		retention:   make(map[string]domaincompliance.RetentionPolicy),
	}
}

func (s *Service) SearchAudit(ctx context.Context, input AuditSearchInput) (domaincompliance.AuditSearchResult, error) {
	entries, err := s.collectAudits(ctx)
	if err != nil {
		return domaincompliance.AuditSearchResult{}, err
	}
	filtered := make([]audit.AuditLog, 0, len(entries))
	for _, entry := range entries {
		if input.Subject != "" && !strings.Contains(entry.Subject, input.Subject) {
			continue
		}
		if input.ActorID != "" && entry.ActorID != input.ActorID {
			continue
		}
		if input.Action != "" && !strings.Contains(strings.ToLower(entry.Action), strings.ToLower(input.Action)) {
			continue
		}
		if input.ScopeType != "" && entry.ScopeType != input.ScopeType {
			continue
		}
		if input.ScopeID != "" && entry.ScopeID != input.ScopeID {
			continue
		}
		if input.CorrelationID != "" && entry.CorrelationID != input.CorrelationID {
			continue
		}
		filtered = append(filtered, sanitizeAudit(entry))
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].CreatedAt.Before(filtered[j].CreatedAt) })
	return domaincompliance.AuditSearchResult{Items: filtered, Count: len(filtered)}, nil
}

func (s *Service) EvidenceBundle(ctx context.Context, input EvidenceInput) (domaincompliance.EvidenceBundle, error) {
	bundle := domaincompliance.EvidenceBundle{
		ID:          "evb-" + input.SubjectType + "-" + input.SubjectID,
		SubjectType: input.SubjectType,
		SubjectID:   input.SubjectID,
		Summary:     fmt.Sprintf("Evidence bundle for %s %s", input.SubjectType, input.SubjectID),
		GeneratedAt: s.now(),
	}
	switch input.SubjectType {
	case "pipeline", "pipelineRun":
		record, err := s.pipelines.Get(ctx, input.SubjectID)
		if err != nil {
			return bundle, err
		}
		bundle.Events = record.Events
		bundle.Audits = record.Audits
		bundle.LogReferences = logReferences(input.SubjectID, record.Logs)
	case "deployment", "deploymentRun":
		record, err := s.deployments.Get(ctx, input.SubjectID)
		if err != nil {
			return bundle, err
		}
		bundle.Release = sanitizeAny(record.Release)
		bundle.Artifacts = anySlice(record.Artifacts)
		bundle.PolicyResults = []any{sanitizeAny(record.Policy)}
		if len(record.Security.Scan.Findings) > 0 {
			bundle.SecurityFindings = anySlice(record.Security.Scan.Findings)
		}
		if record.Approval.ID != "" {
			bundle.Approvals = []any{sanitizeAny(record.Approval)}
		}
		bundle.DeploymentPlans = []any{sanitizeAny(record.Plan)}
		bundle.Events = record.Events
		bundle.Audits = record.Audits
		bundle.LogReferences = logReferences(input.SubjectID, record.Logs)
	case "release":
		record, err := s.artifacts.GetRelease(ctx, input.SubjectID)
		if err != nil {
			return bundle, err
		}
		bundle.Release = sanitizeAny(record.Release)
		bundle.Artifacts = append(anySlice(record.Artifacts), anySlice(record.Bindings)...)
		bundle.Events = record.Events
		bundle.Audits = record.Audits
	case "security", "securityScan":
		record, err := s.security.Get(ctx, input.SubjectID)
		if err != nil {
			return bundle, err
		}
		bundle.SecurityFindings = anySlice(record.Scan.Findings)
		bundle.PolicyResults = []any{sanitizeAny(record.Policy)}
		bundle.Events = record.Events
		bundle.Audits = record.Audits
	default:
		audits, err := s.SearchAudit(ctx, AuditSearchInput{Subject: input.SubjectID})
		if err != nil {
			return bundle, err
		}
		bundle.Audits = audits.Items
	}
	bundle.Events = sanitizeEvents(bundle.Events)
	for i, entry := range bundle.Audits {
		bundle.Audits[i] = sanitizeAudit(entry)
	}
	return bundle, nil
}

func (s *Service) ExportMarkdown(bundle domaincompliance.EvidenceBundle) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Evidence Bundle\n\n")
	fmt.Fprintf(&b, "- Subject: `%s/%s`\n", bundle.SubjectType, bundle.SubjectID)
	fmt.Fprintf(&b, "- Generated: `%s`\n", bundle.GeneratedAt.Format(time.RFC3339))
	fmt.Fprintf(&b, "- Audits: `%d`\n", len(bundle.Audits))
	fmt.Fprintf(&b, "- Events: `%d`\n", len(bundle.Events))
	fmt.Fprintf(&b, "- Log references: `%d`\n\n", len(bundle.LogReferences))
	if len(bundle.Audits) > 0 {
		fmt.Fprintf(&b, "## Audit\n\n")
		for _, entry := range bundle.Audits {
			fmt.Fprintf(&b, "- `%s` %s `%s` actor=`%s`\n", entry.CreatedAt.Format(time.RFC3339), entry.Action, entry.Subject, entry.ActorID)
		}
	}
	if len(bundle.SecurityFindings) > 0 {
		fmt.Fprintf(&b, "\n## Security Findings\n\n")
		fmt.Fprintf(&b, "%d finding(s) included. See JSON export for structured detail.\n", len(bundle.SecurityFindings))
	}
	return b.String()
}

func (s *Service) RetentionPolicy(ctx context.Context, scopeType string, scopeID string) (domaincompliance.RetentionPolicy, error) {
	if err := ctx.Err(); err != nil {
		return domaincompliance.RetentionPolicy{}, err
	}
	key := retentionKey(scopeType, scopeID)
	s.mu.RLock()
	defer s.mu.RUnlock()
	if policy, ok := s.retention[key]; ok {
		return policy, nil
	}
	return defaultRetention(scopeType, scopeID, s.now()), nil
}

func (s *Service) SetRetentionPolicy(ctx context.Context, input RetentionInput) (domaincompliance.RetentionPolicy, error) {
	if err := ctx.Err(); err != nil {
		return domaincompliance.RetentionPolicy{}, err
	}
	policy := defaultRetention(input.ScopeType, input.ScopeID, s.now())
	policy.LogDays = override(policy.LogDays, input.LogDays)
	policy.AuditDays = override(policy.AuditDays, input.AuditDays)
	policy.EventDays = override(policy.EventDays, input.EventDays)
	policy.EvidenceDays = override(policy.EvidenceDays, input.EvidenceDays)
	policy.ImmutableAudit = true
	s.mu.Lock()
	defer s.mu.Unlock()
	s.retention[retentionKey(input.ScopeType, input.ScopeID)] = policy
	return policy, nil
}

func (s *Service) collectAudits(ctx context.Context) ([]audit.AuditLog, error) {
	var entries []audit.AuditLog
	if s.pipelines != nil {
		records, err := s.pipelines.List(ctx)
		if err != nil {
			return nil, err
		}
		for _, record := range records {
			entries = append(entries, record.Audits...)
		}
	}
	if s.deployments != nil {
		records, err := s.deployments.List(ctx)
		if err != nil {
			return nil, err
		}
		for _, record := range records {
			entries = append(entries, record.Audits...)
		}
	}
	if s.releases != nil {
		records, err := s.releases.ListExecutions(ctx, "")
		if err != nil {
			return nil, err
		}
		for _, record := range records {
			entries = append(entries, record.Audits...)
		}
	}
	if s.security != nil {
		records, err := s.security.List(ctx)
		if err != nil {
			return nil, err
		}
		for _, record := range records {
			entries = append(entries, record.Audits...)
		}
	}
	if s.approvals != nil {
		entries = append(entries, s.approvalsAudits(ctx)...)
	}
	if s.artifacts != nil {
		records, err := s.artifacts.ListReleases(ctx)
		if err != nil {
			return nil, err
		}
		for _, record := range records {
			entries = append(entries, record.Audits...)
		}
	}
	return entries, nil
}

func (s *Service) approvalsAudits(ctx context.Context) []audit.AuditLog {
	_ = ctx
	if s.approvals == nil {
		return nil
	}
	// Approval service exposes audit through its backing record methods in tests and local runtime.
	// The public service intentionally keeps the list surface small, so approval evidence is included
	// through subject-specific bundles when approval records are attached to deployments/releases.
	return nil
}

func logReferences(subjectID string, logs []event.LogChunk) []domaincompliance.LogReference {
	refs := make([]domaincompliance.LogReference, 0, len(logs))
	for _, log := range logs {
		refs = append(refs, domaincompliance.LogReference{ID: log.ID, SubjectID: subjectID, Stream: log.Stream, Sequence: log.Sequence, CreatedAt: log.CreatedAt})
	}
	return refs
}

func anySlice[T any](values []T) []any {
	out := make([]any, 0, len(values))
	for _, value := range values {
		out = append(out, sanitizeAny(value))
	}
	return out
}

func sanitizeAudit(entry audit.AuditLog) audit.AuditLog {
	entry.Reason = crypto.RedactString(entry.Reason)
	entry.Before = crypto.RedactMap(entry.Before)
	entry.After = crypto.RedactMap(entry.After)
	entry.Metadata = crypto.RedactMap(entry.Metadata)
	return entry
}

func sanitizeEvents(events []event.Event) []event.Event {
	out := append([]event.Event(nil), events...)
	for i, evt := range out {
		evt.Data = sanitizeAnyMap(evt.Data)
		out[i] = evt
	}
	return out
}

func sanitizeAny(value any) any {
	body, err := json.Marshal(value)
	if err != nil {
		return value
	}
	var decoded any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return value
	}
	return sanitizeAnyValue(decoded)
}

func sanitizeAnyMap(values map[string]any) map[string]any {
	if values == nil {
		return nil
	}
	out := make(map[string]any, len(values))
	for key, value := range values {
		if crypto.IsSensitiveKey(key) {
			out[key] = "[REDACTED]"
			continue
		}
		out[key] = sanitizeAnyValue(value)
	}
	return out
}

func sanitizeAnyValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return sanitizeAnyMap(typed)
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = sanitizeAnyValue(item)
		}
		return out
	case string:
		return crypto.RedactString(typed)
	default:
		return value
	}
}

func defaultRetention(scopeType string, scopeID string, now time.Time) domaincompliance.RetentionPolicy {
	if scopeType == "" {
		scopeType = "global"
	}
	return domaincompliance.RetentionPolicy{
		ID:             "retention-" + retentionKey(scopeType, scopeID),
		ScopeType:      scopeType,
		ScopeID:        scopeID,
		LogDays:        30,
		AuditDays:      365,
		EventDays:      180,
		EvidenceDays:   365,
		ImmutableAudit: true,
		UpdatedAt:      now,
	}
}

func retentionKey(scopeType string, scopeID string) string {
	if scopeType == "" {
		scopeType = "global"
	}
	return scopeType + ":" + scopeID
}

func override(current int, next int) int {
	if next == 0 {
		return current
	}
	return next
}
