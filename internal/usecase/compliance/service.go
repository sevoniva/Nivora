package compliance

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/sevoniva/nivora/internal/domain/audit"
	domaincompliance "github.com/sevoniva/nivora/internal/domain/compliance"
	"github.com/sevoniva/nivora/internal/domain/event"
	"github.com/sevoniva/nivora/internal/domain/tenant"
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
	store       Store
	now         func() time.Time
}

type SubjectScopeResult struct {
	ScopeType string
	ScopeID   string
	Verified  bool
}

func NewService(pipelines *pipelineusecase.Service, deployments *deploymentusecase.Service, artifacts *artifactusecase.Service, releases *releaseusecase.Service, security *securityusecase.Service, approvals *approvalusecase.Service) *Service {
	return NewServiceWithStore(NewMemoryStore(), pipelines, deployments, artifacts, releases, security, approvals)
}

func NewServiceWithStore(store Store, pipelines *pipelineusecase.Service, deployments *deploymentusecase.Service, artifacts *artifactusecase.Service, releases *releaseusecase.Service, security *securityusecase.Service, approvals *approvalusecase.Service) *Service {
	if store == nil {
		store = NewMemoryStore()
	}
	return &Service{
		pipelines:   pipelines,
		deployments: deployments,
		artifacts:   artifacts,
		releases:    releases,
		security:    security,
		approvals:   approvals,
		store:       store,
		now:         time.Now,
	}
}

func (s *Service) SearchAudit(ctx context.Context, input AuditSearchInput) (domaincompliance.AuditSearchResult, error) {
	entries, err := s.collectAudits(ctx)
	if err != nil {
		return domaincompliance.AuditSearchResult{}, err
	}
	storedEntries, err := s.store.SearchAuditLogs(ctx, input)
	if err != nil {
		return domaincompliance.AuditSearchResult{}, err
	}
	entries = append(entries, storedEntries...)
	filtered := make([]audit.AuditLog, 0, len(entries))
	for _, entry := range entries {
		if input.Subject != "" && !strings.Contains(entry.Subject, input.Subject) {
			continue
		}
		if input.SubjectType != "" && entry.SubjectType != input.SubjectType {
			continue
		}
		if input.SubjectID != "" && entry.SubjectID != input.SubjectID {
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
		if input.RequestID != "" && entry.RequestID != input.RequestID {
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

func (s *Service) RecordAudit(ctx context.Context, entry audit.AuditLog) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if entry.ID == "" {
		entry.ID = "audit-" + time.Now().UTC().Format("20060102150405.000000000")
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = s.now()
	}
	if entry.SubjectType == "" {
		entry.SubjectType = "control-plane"
	}
	entry = sanitizeAudit(entry)
	return s.store.AppendAuditLog(ctx, entry)
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
	scope, err := s.SubjectScope(ctx, input.SubjectType, input.SubjectID)
	if err != nil {
		return domaincompliance.EvidenceBundle{}, err
	}
	if scope.Verified {
		bundle.ScopeType = scope.ScopeType
		bundle.ScopeID = scope.ScopeID
	}
	if err := s.store.SaveEvidenceBundle(ctx, bundle); err != nil {
		return domaincompliance.EvidenceBundle{}, err
	}
	return bundle, nil
}

func (s *Service) SubjectScope(ctx context.Context, subjectType string, subjectID string) (SubjectScopeResult, error) {
	if err := ctx.Err(); err != nil {
		return SubjectScopeResult{}, err
	}
	switch strings.TrimSpace(subjectType) {
	case "pipeline", "pipelineRun":
		if s.pipelines == nil {
			return SubjectScopeResult{}, nil
		}
		record, err := s.pipelines.Get(ctx, subjectID)
		if err != nil {
			return SubjectScopeResult{}, err
		}
		if record.Pipeline.ProjectID == "" {
			return SubjectScopeResult{}, nil
		}
		return SubjectScopeResult{ScopeType: tenant.ScopeProject, ScopeID: record.Pipeline.ProjectID, Verified: true}, nil
	case "deployment", "deploymentRun":
		if s.deployments == nil {
			return SubjectScopeResult{}, nil
		}
		record, err := s.deployments.Get(ctx, subjectID)
		if err != nil {
			return SubjectScopeResult{}, err
		}
		projectID := record.Environment.ProjectID
		if projectID == "" {
			projectID = record.Target.ProjectID
		}
		if projectID != "" {
			return SubjectScopeResult{ScopeType: tenant.ScopeProject, ScopeID: projectID, Verified: true}, nil
		}
		if record.Run.EnvironmentID != "" {
			return SubjectScopeResult{ScopeType: tenant.ScopeEnvironment, ScopeID: record.Run.EnvironmentID, Verified: true}, nil
		}
		return SubjectScopeResult{}, nil
	case "releaseExecution", "release_execution":
		if s.releases == nil {
			return SubjectScopeResult{}, nil
		}
		record, err := s.releases.GetExecution(ctx, subjectID)
		if err != nil {
			return SubjectScopeResult{}, err
		}
		for _, target := range record.Plan.Targets {
			if target.ProjectID != "" {
				return SubjectScopeResult{ScopeType: tenant.ScopeProject, ScopeID: target.ProjectID, Verified: true}, nil
			}
		}
		if record.Execution.EnvironmentID != "" {
			return SubjectScopeResult{ScopeType: tenant.ScopeEnvironment, ScopeID: record.Execution.EnvironmentID, Verified: true}, nil
		}
		return SubjectScopeResult{}, nil
	case "security", "securityScan":
		if s.security == nil {
			return SubjectScopeResult{}, nil
		}
		record, err := s.security.Get(ctx, subjectID)
		if err != nil {
			return SubjectScopeResult{}, err
		}
		if string(record.Scan.SubjectType) == subjectType && record.Scan.SubjectID == subjectID {
			return SubjectScopeResult{}, nil
		}
		return s.SubjectScope(ctx, string(record.Scan.SubjectType), record.Scan.SubjectID)
	default:
		return SubjectScopeResult{}, nil
	}
}

func (s *Service) GetEvidenceBundle(ctx context.Context, id string) (domaincompliance.EvidenceBundle, error) {
	if err := ctx.Err(); err != nil {
		return domaincompliance.EvidenceBundle{}, err
	}
	return s.store.GetEvidenceBundle(ctx, id)
}

func (s *Service) SearchEvidenceBundles(ctx context.Context, subjectType string, subjectID string) ([]domaincompliance.EvidenceBundle, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	bundles, err := s.store.SearchEvidenceBundles(ctx, subjectType, subjectID)
	if err != nil {
		return nil, err
	}
	for i := range bundles {
		bundles[i].Events = sanitizeEvents(bundles[i].Events)
		for j, entry := range bundles[i].Audits {
			bundles[i].Audits[j] = sanitizeAudit(entry)
		}
	}
	return bundles, nil
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
	policy, err := s.store.GetRetentionPolicy(ctx, scopeType, scopeID)
	if err == nil {
		return policy, nil
	}
	if err != ErrRetentionPolicyNotFound {
		return domaincompliance.RetentionPolicy{}, err
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
	if err := s.store.SaveRetentionPolicy(ctx, policy); err != nil {
		return domaincompliance.RetentionPolicy{}, err
	}
	return policy, nil
}

type AuditChainVerifyResult struct {
	Valid         bool   `json:"valid"`
	FirstBrokenID string `json:"firstBrokenId,omitempty"`
	Message       string `json:"message,omitempty"`
}

func (s *Service) VerifyAuditChain(ctx context.Context, scopeType, scopeID string) (AuditChainVerifyResult, error) {
	valid, firstBroken, err := s.store.VerifyAuditChain(ctx, scopeType, scopeID)
	if err != nil {
		return AuditChainVerifyResult{Valid: false, Message: err.Error()}, nil
	}
	result := AuditChainVerifyResult{Valid: valid, FirstBrokenID: firstBroken}
	if valid {
		result.Message = "audit chain verified"
	} else if firstBroken != "" {
		result.Message = "audit chain verification failed at record " + firstBroken
	} else {
		result.Message = "no audit records found"
	}
	return result, nil
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
