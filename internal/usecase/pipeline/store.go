package pipeline

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/sevoniva/nivora/internal/domain/audit"
	"github.com/sevoniva/nivora/internal/domain/event"
	domainpipeline "github.com/sevoniva/nivora/internal/domain/pipeline"
	domainrunner "github.com/sevoniva/nivora/internal/domain/runner"
)

var (
	ErrRunNotFound               = errors.New("pipeline run not found")
	ErrRunnerNotFound            = errors.New("runner not found")
	ErrRunnerUnauthorized        = errors.New("runner token is invalid")
	ErrRunnerTokenRevoked        = errors.New("runner token is revoked")
	ErrRunnerGroupNotFound       = errors.New("runner group not found")
	ErrRunnerGroupScopeDenied    = errors.New("runner group scope does not allow runner")
	ErrUnsupportedRunnerExecutor = errors.New("runner executor capability is not supported")
	ErrRunnerConcurrencyLimit    = errors.New("runner concurrency limit reached")
	ErrJobNotFound               = errors.New("job run not found")
	ErrNoClaimableJob            = errors.New("no claimable job found")
	ErrOutboxNotFound            = errors.New("event outbox record not found")
)

type PipelineRepository interface {
	Save(ctx context.Context, record RunRecord) error
}

type PipelineRunRepository interface {
	Get(ctx context.Context, id string) (RunRecord, error)
	List(ctx context.Context) ([]RunRecord, error)
	ListFiltered(ctx context.Context, scopeType, scopeID string) ([]RunRecord, error)
	ListByStatus(ctx context.Context, status domainpipeline.PipelineRunStatus) ([]RunRecord, error)
}

type LogRepository interface {
	AppendLog(ctx context.Context, runID string, log event.LogChunk) error
	LogsByPipelineRun(ctx context.Context, runID string) ([]event.LogChunk, error)
	LogsByJobRun(ctx context.Context, jobRunID string) ([]event.LogChunk, error)
}

type JobRepository interface {
	ClaimJob(ctx context.Context, runnerID string, leaseUntil time.Time) (JobClaim, error)
	UpdateJobStatus(ctx context.Context, jobRunID string, status domainpipeline.JobRunStatus, reason string, at time.Time) (RunRecord, error)
	RequestCancel(ctx context.Context, pipelineRunID string, at time.Time) (RunRecord, error)
	AcquirePipelineRunLease(ctx context.Context, id string, ownerID string, leaseUntil time.Time, at time.Time) (RunRecord, error)
	HeartbeatPipelineRunLease(ctx context.Context, id string, ownerID string, leaseUntil time.Time, at time.Time) (RunRecord, error)
	ListStaleRunningPipelineRuns(ctx context.Context, olderThan time.Time, limit int) ([]RunRecord, error)
	ListExpiredJobClaims(ctx context.Context, now time.Time, limit int) ([]JobClaim, error)
}

type EventRepository interface {
	AppendEvent(ctx context.Context, runID string, evt event.Event) error
	EventsByPipelineRun(ctx context.Context, runID string) ([]event.Event, error)
}

type EventOutboxRepository interface {
	AppendOutbox(ctx context.Context, item EventOutboxRecord) error
	ListPendingOutbox(ctx context.Context, limit int) ([]EventOutboxRecord, error)
	MarkOutboxPublished(ctx context.Context, id string, at time.Time) error
	MarkOutboxFailed(ctx context.Context, id string, retryCount int, nextAttemptAt time.Time, reason string) error
}

type AuditRepository interface {
	AppendAudit(ctx context.Context, runID string, entry audit.AuditLog) error
	AuditBySubject(ctx context.Context, subject string) ([]audit.AuditLog, error)
}

type PipelineMetadataRepository interface {
	SaveArtifact(ctx context.Context, artifact PipelineArtifact) error
	ArtifactsByPipelineRun(ctx context.Context, runID string) ([]PipelineArtifact, error)
	SaveCacheEntry(ctx context.Context, entry PipelineCacheEntry) error
	CacheEntriesByPipelineRun(ctx context.Context, runID string) ([]PipelineCacheEntry, error)
	SaveAnnotation(ctx context.Context, annotation StepAnnotation) error
	AnnotationsByPipelineRun(ctx context.Context, runID string) ([]StepAnnotation, error)
	SaveStepSummary(ctx context.Context, summary StepSummary) error
	StepSummariesByPipelineRun(ctx context.Context, runID string) ([]StepSummary, error)
}

type RunnerRepository interface {
	SaveRunnerGroup(ctx context.Context, group domainrunner.RunnerGroup) error
	GetRunnerGroup(ctx context.Context, id string) (domainrunner.RunnerGroup, error)
	ListRunnerGroups(ctx context.Context) ([]domainrunner.RunnerGroup, error)
	RegisterRunner(ctx context.Context, runner domainrunner.Runner) error
	Heartbeat(ctx context.Context, runnerID string, at time.Time) (domainrunner.Runner, error)
	GetRunner(ctx context.Context, id string) (domainrunner.Runner, error)
	ListRunners(ctx context.Context) ([]domainrunner.Runner, error)
	SelectRunner(ctx context.Context, executor string, labels map[string]string) (domainrunner.Runner, error)
	RotateRunnerToken(ctx context.Context, runnerID string, tokenID string, tokenHash string, at time.Time) (domainrunner.Runner, error)
	RevokeRunnerToken(ctx context.Context, runnerID string, at time.Time) (domainrunner.Runner, error)
	CountActiveJobs(ctx context.Context, runnerID string) (int, error)
	CountActiveJobsByRunnerGroup(ctx context.Context, groupID string) (int, error)
	MarkOfflineRunners(ctx context.Context, cutoff time.Time, at time.Time) ([]domainrunner.Runner, error)
}

type Store interface {
	PipelineRepository
	PipelineRunRepository
	LogRepository
	JobRepository
	EventRepository
	EventOutboxRepository
	AuditRepository
	PipelineMetadataRepository
	RunnerRepository
}

type MemoryStore struct {
	mu          sync.RWMutex
	runs        map[string]RunRecord
	runners     map[string]domainrunner.Runner
	groups      map[string]domainrunner.RunnerGroup
	outbox      map[string]EventOutboxRecord
	nextSeq     map[string]int64
	artifacts   map[string]PipelineArtifact
	caches      map[string]PipelineCacheEntry
	annotations map[string]StepAnnotation
	summaries   map[string]StepSummary
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		runs:        make(map[string]RunRecord),
		runners:     make(map[string]domainrunner.Runner),
		groups:      make(map[string]domainrunner.RunnerGroup),
		outbox:      make(map[string]EventOutboxRecord),
		nextSeq:     make(map[string]int64),
		artifacts:   make(map[string]PipelineArtifact),
		caches:      make(map[string]PipelineCacheEntry),
		annotations: make(map[string]StepAnnotation),
		summaries:   make(map[string]StepSummary),
	}
}

func (s *MemoryStore) Save(ctx context.Context, record RunRecord) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.runs[record.Run.ID]; ok {
		if record.Logs == nil {
			record.Logs = existing.Logs
		}
		if record.Events == nil {
			record.Events = existing.Events
		}
		if record.Audits == nil {
			record.Audits = existing.Audits
		}
		if record.Artifacts == nil {
			record.Artifacts = existing.Artifacts
		}
		if record.Caches == nil {
			record.Caches = existing.Caches
		}
		if record.Annotations == nil {
			record.Annotations = existing.Annotations
		}
		if record.Summaries == nil {
			record.Summaries = existing.Summaries
		}
	}
	s.runs[record.Run.ID] = cloneRecord(record)
	return nil
}

func (s *MemoryStore) Get(ctx context.Context, id string) (RunRecord, error) {
	select {
	case <-ctx.Done():
		return RunRecord{}, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.runs[id]
	if !ok {
		return RunRecord{}, ErrRunNotFound
	}
	return cloneRecord(record), nil
}

func (s *MemoryStore) List(ctx context.Context) ([]RunRecord, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	records := make([]RunRecord, 0, len(s.runs))
	for _, record := range s.runs {
		records = append(records, cloneRecord(record))
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].Run.CreatedAt.Before(records[j].Run.CreatedAt)
	})
	return records, nil
}

func (s *MemoryStore) ListFiltered(ctx context.Context, scopeType, scopeID string) ([]RunRecord, error) {
	all, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	if scopeType == "" {
		return all, nil
	}
	var filtered []RunRecord
	for _, r := range all {
		if scopeType == "project" && r.Pipeline.ProjectID == scopeID {
			filtered = append(filtered, r)
		}
	}
	return filtered, nil
}

func (s *MemoryStore) ListByStatus(ctx context.Context, status domainpipeline.PipelineRunStatus) ([]RunRecord, error) {
	records, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	filtered := records[:0]
	for _, record := range records {
		if record.Run.Status == status {
			filtered = append(filtered, record)
		}
	}
	return filtered, nil
}

func (s *MemoryStore) AppendLog(ctx context.Context, runID string, log event.LogChunk) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.runs[runID]
	if !ok {
		return ErrRunNotFound
	}
	s.nextSeq[runID]++
	log.Sequence = s.nextSeq[runID]
	record.Logs = append(record.Logs, log)
	s.runs[runID] = record
	return nil
}

func (s *MemoryStore) ClaimJob(ctx context.Context, runnerID string, leaseUntil time.Time) (JobClaim, error) {
	select {
	case <-ctx.Done():
		return JobClaim{}, ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	runner, ok := s.runners[runnerID]
	if !ok {
		return JobClaim{}, ErrRunnerNotFound
	}
	group := domainrunner.RunnerGroup{}
	if runner.GroupID != "" {
		var ok bool
		group, ok = s.groups[runner.GroupID]
		if !ok {
			return JobClaim{}, ErrRunnerGroupNotFound
		}
	}
	if runner.Status != "online" {
		return JobClaim{}, ErrRunnerNotFound
	}
	active := activeJobsInRecords(s.runs, runnerID)
	if runner.MaxConcurrency > 0 && active >= runner.MaxConcurrency {
		return JobClaim{}, ErrRunnerConcurrencyLimit
	}
	if group.ID != "" && group.MaxConcurrency > 0 && activeJobsInGroupRecords(s.runs, s.runners, group.ID) >= group.MaxConcurrency {
		return JobClaim{}, ErrRunnerConcurrencyLimit
	}
	runIDs := make([]string, 0, len(s.runs))
	for id := range s.runs {
		runIDs = append(runIDs, id)
	}
	sort.Strings(runIDs)
	for _, runID := range runIDs {
		record := s.runs[runID]
		if !runnerCanClaimRecord(runner, group, record) {
			continue
		}
		if record.Run.Status != domainpipeline.PipelineRunQueued && record.Run.Status != domainpipeline.PipelineRunRunning {
			continue
		}
		for stageIndex := range record.Stages {
			for jobIndex := range record.Stages[stageIndex].Jobs {
				job := &record.Stages[stageIndex].Jobs[jobIndex].Job
				executor := domainrunner.ExecutorShell
				jobLabels := map[string]string(nil)
				if stageIndex < len(record.Definition.Spec.Stages) && jobIndex < len(record.Definition.Spec.Stages[stageIndex].Jobs) {
					specJob := record.Definition.Spec.Stages[stageIndex].Jobs[jobIndex]
					executor = normalizeRecordExecutor(specJob.Executor)
					jobLabels = specJob.Labels
				}
				if !domainrunner.IsSupportedExecutorCapability(executor) {
					continue
				}
				if group.ID != "" && len(group.Executors) > 0 && !executorListContains(group.Executors, executor) {
					continue
				}
				if !runnerSupportsExecutor(runner, executor) {
					continue
				}
				if !labelsMatch(runner.Labels, jobLabels) {
					continue
				}
				claimable := job.Status == domainpipeline.JobRunPending || job.Status == domainpipeline.JobRunRetrying
				leaseExpired := job.Status == domainpipeline.JobRunAssigned && job.LeaseExpiresAt != nil && job.LeaseExpiresAt.Before(now)
				if !claimable && !leaseExpired {
					continue
				}
				stage := &record.Stages[stageIndex].Stage
				if record.Run.Status == domainpipeline.PipelineRunQueued {
					record.Run.Status = domainpipeline.PipelineRunRunning
					record.Run.StartedAt = &now
					record.Run.UpdatedAt = now
				}
				if stage.Status == domainpipeline.JobRunPending {
					stage.Status = domainpipeline.JobRunRunning
					stage.StartedAt = &now
					stage.UpdatedAt = now
				}
				job.Status = domainpipeline.JobRunAssigned
				job.RunnerID = runnerID
				job.LeaseExpiresAt = &leaseUntil
				job.UpdatedAt = now
				if job.Attempt <= 0 {
					job.Attempt = 1
				}
				claim := buildJobClaim(record, stageIndex, jobIndex, runnerID, leaseUntil)
				s.runs[runID] = cloneRecord(record)
				return claim, nil
			}
		}
		s.runs[runID] = cloneRecord(record)
	}
	return JobClaim{}, ErrNoClaimableJob
}

func runnerCanClaimRecord(runner domainrunner.Runner, group domainrunner.RunnerGroup, record RunRecord) bool {
	if projectID := runner.Labels["projectId"]; projectID != "" && record.Pipeline.ProjectID != projectID {
		return false
	}
	if environmentID := runner.Labels["environmentId"]; environmentID != "" {
		recordEnvironmentID := firstNonEmpty(record.Pipeline.Labels["environmentId"], record.Pipeline.Metadata["environmentId"])
		if recordEnvironmentID != environmentID {
			return false
		}
	}
	if group.ProjectID != "" && record.Pipeline.ProjectID != group.ProjectID {
		return false
	}
	if len(group.EnvironmentIDs) > 0 {
		recordEnvironmentID := firstNonEmpty(record.Pipeline.Labels["environmentId"], record.Pipeline.Metadata["environmentId"])
		if !contains(group.EnvironmentIDs, recordEnvironmentID) {
			return false
		}
	}
	return true
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func (s *MemoryStore) UpdateJobStatus(ctx context.Context, jobRunID string, status domainpipeline.JobRunStatus, reason string, at time.Time) (RunRecord, error) {
	select {
	case <-ctx.Done():
		return RunRecord{}, ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for runID, record := range s.runs {
		for stageIndex := range record.Stages {
			for jobIndex := range record.Stages[stageIndex].Jobs {
				job := &record.Stages[stageIndex].Jobs[jobIndex].Job
				if job.ID != jobRunID {
					continue
				}
				job.Status = status
				job.FailureReason = reason
				job.UpdatedAt = at
				if status == domainpipeline.JobRunRunning && job.StartedAt == nil {
					job.StartedAt = &at
				}
				if isTerminalJobStatus(status) {
					job.FinishedAt = &at
					job.LeaseExpiresAt = nil
				}
				record.Stages[stageIndex].Stage.UpdatedAt = at
				record = updatePipelineStatusFromJobs(record, at)
				s.runs[runID] = cloneRecord(record)
				return cloneRecord(record), nil
			}
		}
	}
	return RunRecord{}, ErrJobNotFound
}

func (s *MemoryStore) RequestCancel(ctx context.Context, pipelineRunID string, at time.Time) (RunRecord, error) {
	select {
	case <-ctx.Done():
		return RunRecord{}, ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.runs[pipelineRunID]
	if !ok {
		return RunRecord{}, ErrRunNotFound
	}
	record.Run.CancelRequested = true
	record.Run.UpdatedAt = at
	s.runs[pipelineRunID] = cloneRecord(record)
	return cloneRecord(record), nil
}

func (s *MemoryStore) AcquirePipelineRunLease(ctx context.Context, id string, ownerID string, leaseUntil time.Time, at time.Time) (RunRecord, error) {
	select {
	case <-ctx.Done():
		return RunRecord{}, ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.runs[id]
	if !ok {
		return RunRecord{}, ErrRunNotFound
	}
	record.Run.OwnerID = ownerID
	record.Run.LeaseExpiresAt = &leaseUntil
	record.Run.HeartbeatAt = &at
	record.Run.UpdatedAt = at
	record.Run.Attempt++
	if record.Run.Attempt <= 0 {
		record.Run.Attempt = 1
	}
	s.runs[id] = cloneRecord(record)
	return cloneRecord(record), nil
}

func (s *MemoryStore) HeartbeatPipelineRunLease(ctx context.Context, id string, ownerID string, leaseUntil time.Time, at time.Time) (RunRecord, error) {
	select {
	case <-ctx.Done():
		return RunRecord{}, ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.runs[id]
	if !ok {
		return RunRecord{}, ErrRunNotFound
	}
	if record.Run.OwnerID != "" && record.Run.OwnerID != ownerID {
		return RunRecord{}, errors.New("pipeline run lease is owned by another worker")
	}
	record.Run.OwnerID = ownerID
	record.Run.LeaseExpiresAt = &leaseUntil
	record.Run.HeartbeatAt = &at
	record.Run.UpdatedAt = at
	s.runs[id] = cloneRecord(record)
	return cloneRecord(record), nil
}

func (s *MemoryStore) ListStaleRunningPipelineRuns(ctx context.Context, olderThan time.Time, limit int) ([]RunRecord, error) {
	records, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	var stale []RunRecord
	for _, record := range records {
		expiredLease := record.Run.LeaseExpiresAt != nil && record.Run.LeaseExpiresAt.Before(olderThan)
		staleTimestamp := record.Run.UpdatedAt.Before(olderThan)
		if record.Run.Status == domainpipeline.PipelineRunRunning && (expiredLease || staleTimestamp) {
			stale = append(stale, record)
		}
	}
	if limit > 0 && len(stale) > limit {
		stale = stale[:limit]
	}
	return stale, nil
}

func (s *MemoryStore) ListExpiredJobClaims(ctx context.Context, now time.Time, limit int) ([]JobClaim, error) {
	records, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	var claims []JobClaim
	for _, record := range records {
		for stageIndex := range record.Stages {
			for jobIndex := range record.Stages[stageIndex].Jobs {
				job := record.Stages[stageIndex].Jobs[jobIndex].Job
				if job.Status != domainpipeline.JobRunAssigned || job.LeaseExpiresAt == nil || !job.LeaseExpiresAt.Before(now) {
					continue
				}
				claims = append(claims, buildJobClaim(record, stageIndex, jobIndex, job.RunnerID, *job.LeaseExpiresAt))
				if limit > 0 && len(claims) >= limit {
					return claims, nil
				}
			}
		}
	}
	return claims, nil
}

func (s *MemoryStore) LogsByPipelineRun(ctx context.Context, runID string) ([]event.LogChunk, error) {
	record, err := s.Get(ctx, runID)
	if err != nil {
		return nil, err
	}
	return sortLogs(record.Logs), nil
}

func (s *MemoryStore) LogsByJobRun(ctx context.Context, jobRunID string) ([]event.LogChunk, error) {
	records, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	var logs []event.LogChunk
	for _, record := range records {
		for _, log := range record.Logs {
			if log.JobRunID == jobRunID {
				logs = append(logs, log)
			}
		}
	}
	return sortLogs(logs), nil
}

func (s *MemoryStore) AppendEvent(ctx context.Context, runID string, evt event.Event) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.runs[runID]
	if !ok {
		return ErrRunNotFound
	}
	record.Events = append(record.Events, evt)
	s.runs[runID] = record
	return nil
}

func (s *MemoryStore) AppendOutbox(ctx context.Context, item EventOutboxRecord) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.outbox[item.ID] = item
	return nil
}

func (s *MemoryStore) ListPendingOutbox(ctx context.Context, limit int) ([]EventOutboxRecord, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]EventOutboxRecord, 0, len(s.outbox))
	now := time.Now()
	for _, item := range s.outbox {
		if item.Status == "pending" || (item.Status == "failed" && (item.NextAttemptAt == nil || !item.NextAttemptAt.After(now))) {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return append([]EventOutboxRecord(nil), items...), nil
}

func (s *MemoryStore) MarkOutboxPublished(ctx context.Context, id string, at time.Time) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.outbox[id]
	if !ok {
		return ErrOutboxNotFound
	}
	item.Status = "published"
	item.PublishedAt = &at
	item.LastError = ""
	s.outbox[id] = item
	return nil
}

func (s *MemoryStore) MarkOutboxFailed(ctx context.Context, id string, retryCount int, nextAttemptAt time.Time, reason string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.outbox[id]
	if !ok {
		return ErrOutboxNotFound
	}
	item.Status = "failed"
	item.RetryCount = retryCount
	item.NextAttemptAt = &nextAttemptAt
	item.LastError = reason
	s.outbox[id] = item
	return nil
}

func (s *MemoryStore) EventsByPipelineRun(ctx context.Context, runID string) ([]event.Event, error) {
	record, err := s.Get(ctx, runID)
	if err != nil {
		return nil, err
	}
	events := append([]event.Event(nil), record.Events...)
	sort.Slice(events, func(i, j int) bool {
		return events[i].Time.Before(events[j].Time)
	})
	return events, nil
}

func (s *MemoryStore) AppendAudit(ctx context.Context, runID string, entry audit.AuditLog) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.runs[runID]
	if !ok {
		return ErrRunNotFound
	}
	record.Audits = append(record.Audits, entry)
	s.runs[runID] = record
	return nil
}

func (s *MemoryStore) AuditBySubject(ctx context.Context, subject string) ([]audit.AuditLog, error) {
	records, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	var entries []audit.AuditLog
	for _, record := range records {
		for _, entry := range record.Audits {
			if entry.Subject == subject {
				entries = append(entries, entry)
			}
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].CreatedAt.Before(entries[j].CreatedAt)
	})
	return entries, nil
}

func (s *MemoryStore) SaveArtifact(ctx context.Context, artifact PipelineArtifact) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.runs[artifact.PipelineRunID]
	if !ok {
		return ErrRunNotFound
	}
	artifact.Metadata = cloneMap(artifact.Metadata)
	s.artifacts[artifact.ID] = artifact
	record.Artifacts = upsertArtifact(record.Artifacts, artifact)
	s.runs[artifact.PipelineRunID] = cloneRecord(record)
	return nil
}

func (s *MemoryStore) ArtifactsByPipelineRun(ctx context.Context, runID string) ([]PipelineArtifact, error) {
	if _, err := s.Get(ctx, runID); err != nil {
		return nil, err
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var artifacts []PipelineArtifact
	for _, artifact := range s.artifacts {
		if artifact.PipelineRunID == runID {
			artifacts = append(artifacts, cloneArtifact(artifact))
		}
	}
	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].CreatedAt.Before(artifacts[j].CreatedAt) })
	if artifacts == nil {
		artifacts = []PipelineArtifact{}
	}
	return artifacts, nil
}

func (s *MemoryStore) SaveCacheEntry(ctx context.Context, entry PipelineCacheEntry) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.runs[entry.PipelineRunID]; !ok {
		return ErrRunNotFound
	}
	entry.RestoreKeys = append([]string(nil), entry.RestoreKeys...)
	entry.Metadata = cloneMap(entry.Metadata)
	s.caches[entry.ID] = entry
	record := s.runs[entry.PipelineRunID]
	record.Caches = upsertCacheEntry(record.Caches, entry)
	s.runs[entry.PipelineRunID] = cloneRecord(record)
	return nil
}

func (s *MemoryStore) CacheEntriesByPipelineRun(ctx context.Context, runID string) ([]PipelineCacheEntry, error) {
	if _, err := s.Get(ctx, runID); err != nil {
		return nil, err
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var entries []PipelineCacheEntry
	for _, entry := range s.caches {
		if entry.PipelineRunID == runID {
			entries = append(entries, cloneCacheEntry(entry))
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].CreatedAt.Before(entries[j].CreatedAt) })
	if entries == nil {
		entries = []PipelineCacheEntry{}
	}
	return entries, nil
}

func (s *MemoryStore) SaveAnnotation(ctx context.Context, annotation StepAnnotation) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.runs[annotation.PipelineRunID]
	if !ok {
		return ErrRunNotFound
	}
	annotation.Metadata = cloneMap(annotation.Metadata)
	s.annotations[annotation.ID] = annotation
	record.Annotations = upsertAnnotation(record.Annotations, annotation)
	s.runs[annotation.PipelineRunID] = cloneRecord(record)
	return nil
}

func (s *MemoryStore) AnnotationsByPipelineRun(ctx context.Context, runID string) ([]StepAnnotation, error) {
	if _, err := s.Get(ctx, runID); err != nil {
		return nil, err
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var annotations []StepAnnotation
	for _, annotation := range s.annotations {
		if annotation.PipelineRunID == runID {
			annotations = append(annotations, cloneAnnotation(annotation))
		}
	}
	sort.Slice(annotations, func(i, j int) bool { return annotations[i].CreatedAt.Before(annotations[j].CreatedAt) })
	if annotations == nil {
		annotations = []StepAnnotation{}
	}
	return annotations, nil
}

func (s *MemoryStore) SaveStepSummary(ctx context.Context, summary StepSummary) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.runs[summary.PipelineRunID]
	if !ok {
		return ErrRunNotFound
	}
	summary.Metadata = cloneMap(summary.Metadata)
	s.summaries[summary.ID] = summary
	record.Summaries = upsertSummary(record.Summaries, summary)
	s.runs[summary.PipelineRunID] = cloneRecord(record)
	return nil
}

func (s *MemoryStore) StepSummariesByPipelineRun(ctx context.Context, runID string) ([]StepSummary, error) {
	if _, err := s.Get(ctx, runID); err != nil {
		return nil, err
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var summaries []StepSummary
	for _, summary := range s.summaries {
		if summary.PipelineRunID == runID {
			summaries = append(summaries, cloneSummary(summary))
		}
	}
	sort.Slice(summaries, func(i, j int) bool { return summaries[i].CreatedAt.Before(summaries[j].CreatedAt) })
	if summaries == nil {
		summaries = []StepSummary{}
	}
	return summaries, nil
}

func (s *MemoryStore) RegisterRunner(ctx context.Context, runner domainrunner.Runner) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runners[runner.ID] = cloneRunner(runner)
	return nil
}

func (s *MemoryStore) SaveRunnerGroup(ctx context.Context, group domainrunner.RunnerGroup) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.groups[group.ID] = cloneRunnerGroup(group)
	return nil
}

func (s *MemoryStore) GetRunnerGroup(ctx context.Context, id string) (domainrunner.RunnerGroup, error) {
	select {
	case <-ctx.Done():
		return domainrunner.RunnerGroup{}, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	group, ok := s.groups[id]
	if !ok {
		return domainrunner.RunnerGroup{}, ErrRunnerGroupNotFound
	}
	return cloneRunnerGroup(group), nil
}

func (s *MemoryStore) ListRunnerGroups(ctx context.Context) ([]domainrunner.RunnerGroup, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	groups := make([]domainrunner.RunnerGroup, 0, len(s.groups))
	for _, group := range s.groups {
		groups = append(groups, cloneRunnerGroup(group))
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].ID < groups[j].ID })
	return groups, nil
}

func (s *MemoryStore) Heartbeat(ctx context.Context, runnerID string, at time.Time) (domainrunner.Runner, error) {
	select {
	case <-ctx.Done():
		return domainrunner.Runner{}, ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	runner, ok := s.runners[runnerID]
	if !ok {
		return domainrunner.Runner{}, ErrRunnerNotFound
	}
	runner.Status = "online"
	runner.LastHeartbeatAt = &at
	runner.LastSeenAt = &at
	runner.UpdatedAt = at
	s.runners[runnerID] = runner
	return cloneRunner(runner), nil
}

func (s *MemoryStore) GetRunner(ctx context.Context, id string) (domainrunner.Runner, error) {
	select {
	case <-ctx.Done():
		return domainrunner.Runner{}, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	runner, ok := s.runners[id]
	if !ok {
		return domainrunner.Runner{}, ErrRunnerNotFound
	}
	return cloneRunner(runner), nil
}

func (s *MemoryStore) ListRunners(ctx context.Context) ([]domainrunner.Runner, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	runners := make([]domainrunner.Runner, 0, len(s.runners))
	for _, runner := range s.runners {
		runner.ActiveJobs = activeJobsInRecords(s.runs, runner.ID)
		runners = append(runners, cloneRunner(runner))
	}
	sort.Slice(runners, func(i, j int) bool {
		return runners[i].ID < runners[j].ID
	})
	return runners, nil
}

func (s *MemoryStore) SelectRunner(ctx context.Context, executor string, labels map[string]string) (domainrunner.Runner, error) {
	executor = normalizeRecordExecutor(executor)
	if !domainrunner.IsSupportedExecutorCapability(executor) {
		return domainrunner.Runner{}, ErrRunnerNotFound
	}
	runners, err := s.ListRunners(ctx)
	if err != nil {
		return domainrunner.Runner{}, err
	}
	for _, runner := range runners {
		if runner.Status != "online" {
			continue
		}
		if !runnerSupportsExecutor(runner, executor) {
			continue
		}
		if !labelsMatch(runner.Labels, labels) {
			continue
		}
		active, err := s.CountActiveJobs(ctx, runner.ID)
		if err != nil {
			return domainrunner.Runner{}, err
		}
		if runner.MaxConcurrency > 0 && active >= runner.MaxConcurrency {
			continue
		}
		runner.ActiveJobs = active
		return runner, nil
	}
	return domainrunner.Runner{}, ErrRunnerNotFound
}

func (s *MemoryStore) RotateRunnerToken(ctx context.Context, runnerID string, tokenID string, tokenHash string, at time.Time) (domainrunner.Runner, error) {
	select {
	case <-ctx.Done():
		return domainrunner.Runner{}, ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	runner, ok := s.runners[runnerID]
	if !ok {
		return domainrunner.Runner{}, ErrRunnerNotFound
	}
	runner.TokenID = tokenID
	runner.TokenHash = tokenHash
	runner.TokenRevokedAt = nil
	if runner.TokenCreatedAt == nil {
		runner.TokenCreatedAt = &at
	} else {
		runner.TokenRotatedAt = &at
	}
	runner.UpdatedAt = at
	s.runners[runnerID] = runner
	return cloneRunner(runner), nil
}

func (s *MemoryStore) RevokeRunnerToken(ctx context.Context, runnerID string, at time.Time) (domainrunner.Runner, error) {
	select {
	case <-ctx.Done():
		return domainrunner.Runner{}, ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	runner, ok := s.runners[runnerID]
	if !ok {
		return domainrunner.Runner{}, ErrRunnerNotFound
	}
	runner.TokenHash = ""
	runner.TokenRevokedAt = &at
	runner.UpdatedAt = at
	s.runners[runnerID] = runner
	return cloneRunner(runner), nil
}

func (s *MemoryStore) CountActiveJobs(ctx context.Context, runnerID string) (int, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return activeJobsInRecords(s.runs, runnerID), nil
}

func (s *MemoryStore) CountActiveJobsByRunnerGroup(ctx context.Context, groupID string) (int, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return activeJobsInGroupRecords(s.runs, s.runners, groupID), nil
}

func (s *MemoryStore) MarkOfflineRunners(ctx context.Context, cutoff time.Time, at time.Time) ([]domainrunner.Runner, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var offline []domainrunner.Runner
	for id, runner := range s.runners {
		if runner.Status != "online" || runner.LastHeartbeatAt == nil || !runner.LastHeartbeatAt.Before(cutoff) {
			continue
		}
		runner.Status = "offline"
		runner.UpdatedAt = at
		s.runners[id] = runner
		offline = append(offline, cloneRunner(runner))
	}
	return offline, nil
}

func cloneRecord(record RunRecord) RunRecord {
	record.Definition.Spec.Stages = cloneSpecStages(record.Definition.Spec.Stages)
	record.Stages = append([]StageRecord(nil), record.Stages...)
	for i := range record.Stages {
		record.Stages[i].Jobs = append([]JobRecord(nil), record.Stages[i].Jobs...)
		for j := range record.Stages[i].Jobs {
			steps := record.Stages[i].Jobs[j].Steps
			record.Stages[i].Jobs[j].Steps = append([]domainpipeline.StepRun(nil), steps...)
		}
	}
	record.Logs = append([]event.LogChunk(nil), record.Logs...)
	record.Events = append([]event.Event(nil), record.Events...)
	record.Audits = append([]audit.AuditLog(nil), record.Audits...)
	record.Artifacts = cloneArtifacts(record.Artifacts)
	record.Caches = cloneCacheEntries(record.Caches)
	record.Annotations = cloneAnnotations(record.Annotations)
	record.Summaries = cloneSummaries(record.Summaries)
	return record
}

func cloneSpecStages(stages []Stage) []Stage {
	out := append([]Stage(nil), stages...)
	for i := range out {
		out[i].Jobs = append([]Job(nil), out[i].Jobs...)
		for j := range out[i].Jobs {
			out[i].Jobs[j].Labels = cloneMap(out[i].Jobs[j].Labels)
			out[i].Jobs[j].Steps = append([]Step(nil), out[i].Jobs[j].Steps...)
		}
	}
	return out
}

func cloneRunner(runner domainrunner.Runner) domainrunner.Runner {
	runner.Labels = cloneMap(runner.Labels)
	runner.Executors = append([]string(nil), runner.Executors...)
	runner.Capabilities = append([]string(nil), runner.Capabilities...)
	return runner
}

func cloneRunnerGroup(group domainrunner.RunnerGroup) domainrunner.RunnerGroup {
	group.Labels = cloneMap(group.Labels)
	group.EnvironmentIDs = append([]string(nil), group.EnvironmentIDs...)
	group.Executors = append([]string(nil), group.Executors...)
	return group
}

func upsertArtifact(items []PipelineArtifact, item PipelineArtifact) []PipelineArtifact {
	for i := range items {
		if items[i].ID == item.ID {
			items[i] = cloneArtifact(item)
			return items
		}
	}
	return append(items, cloneArtifact(item))
}

func upsertCacheEntry(items []PipelineCacheEntry, item PipelineCacheEntry) []PipelineCacheEntry {
	for i := range items {
		if items[i].ID == item.ID {
			items[i] = cloneCacheEntry(item)
			return items
		}
	}
	return append(items, cloneCacheEntry(item))
}

func upsertAnnotation(items []StepAnnotation, item StepAnnotation) []StepAnnotation {
	for i := range items {
		if items[i].ID == item.ID {
			items[i] = cloneAnnotation(item)
			return items
		}
	}
	return append(items, cloneAnnotation(item))
}

func upsertSummary(items []StepSummary, item StepSummary) []StepSummary {
	for i := range items {
		if items[i].ID == item.ID {
			items[i] = cloneSummary(item)
			return items
		}
	}
	return append(items, cloneSummary(item))
}

func cloneArtifacts(items []PipelineArtifact) []PipelineArtifact {
	out := append([]PipelineArtifact(nil), items...)
	for i := range out {
		out[i] = cloneArtifact(out[i])
	}
	return out
}

func cloneArtifact(item PipelineArtifact) PipelineArtifact {
	item.Metadata = cloneMap(item.Metadata)
	return item
}

func cloneCacheEntry(item PipelineCacheEntry) PipelineCacheEntry {
	item.RestoreKeys = append([]string(nil), item.RestoreKeys...)
	item.Metadata = cloneMap(item.Metadata)
	return item
}

func cloneCacheEntries(items []PipelineCacheEntry) []PipelineCacheEntry {
	out := append([]PipelineCacheEntry(nil), items...)
	for i := range out {
		out[i] = cloneCacheEntry(out[i])
	}
	return out
}

func cloneAnnotations(items []StepAnnotation) []StepAnnotation {
	out := append([]StepAnnotation(nil), items...)
	for i := range out {
		out[i] = cloneAnnotation(out[i])
	}
	return out
}

func cloneAnnotation(item StepAnnotation) StepAnnotation {
	item.Metadata = cloneMap(item.Metadata)
	return item
}

func cloneSummaries(items []StepSummary) []StepSummary {
	out := append([]StepSummary(nil), items...)
	for i := range out {
		out[i] = cloneSummary(out[i])
	}
	return out
}

func cloneSummary(item StepSummary) StepSummary {
	item.Metadata = cloneMap(item.Metadata)
	return item
}

func activeJobsInRecords(records map[string]RunRecord, runnerID string) int {
	active := 0
	for _, record := range records {
		for _, stage := range record.Stages {
			for _, job := range stage.Jobs {
				if job.Job.RunnerID == runnerID && (job.Job.Status == domainpipeline.JobRunAssigned || job.Job.Status == domainpipeline.JobRunRunning) {
					active++
				}
			}
		}
	}
	return active
}

func activeJobsInGroupRecords(records map[string]RunRecord, runners map[string]domainrunner.Runner, groupID string) int {
	if groupID == "" {
		return 0
	}
	active := 0
	for _, record := range records {
		for _, stage := range record.Stages {
			for _, job := range stage.Jobs {
				runner := runners[job.Job.RunnerID]
				if runner.GroupID != groupID {
					continue
				}
				if job.Job.Status == domainpipeline.JobRunAssigned || job.Job.Status == domainpipeline.JobRunRunning {
					active++
				}
			}
		}
	}
	return active
}

func buildJobClaim(record RunRecord, stageIndex int, jobIndex int, runnerID string, leaseUntil time.Time) JobClaim {
	job := record.Stages[stageIndex].Jobs[jobIndex]
	stepIDs := make([]string, 0, len(job.Steps))
	commands := make([]string, 0, len(job.Steps))
	if stageIndex < len(record.Definition.Spec.Stages) && jobIndex < len(record.Definition.Spec.Stages[stageIndex].Jobs) {
		for _, step := range record.Definition.Spec.Stages[stageIndex].Jobs[jobIndex].Steps {
			commands = append(commands, step.Run)
		}
	}
	for _, step := range job.Steps {
		stepIDs = append(stepIDs, step.ID)
	}
	return JobClaim{
		PipelineRunID:   record.Run.ID,
		StageRunID:      record.Stages[stageIndex].Stage.ID,
		JobRunID:        job.Job.ID,
		StepRunIDs:      stepIDs,
		RunnerID:        runnerID,
		Executor:        "shell",
		Commands:        commands,
		Attempt:         job.Job.Attempt,
		LeaseExpiresAt:  leaseUntil,
		CancelRequested: record.Run.CancelRequested,
		Status:          job.Job.Status,
	}
}

func updatePipelineStatusFromJobs(record RunRecord, at time.Time) RunRecord {
	allSucceeded := true
	anyFailed := false
	for stageIndex := range record.Stages {
		stageSucceeded := true
		for jobIndex := range record.Stages[stageIndex].Jobs {
			status := record.Stages[stageIndex].Jobs[jobIndex].Job.Status
			if status == domainpipeline.JobRunFailed || status == domainpipeline.JobRunCanceled {
				anyFailed = true
				stageSucceeded = false
				allSucceeded = false
			}
			if status != domainpipeline.JobRunSucceeded && status != domainpipeline.JobRunSkipped {
				stageSucceeded = false
				allSucceeded = false
			}
		}
		if anyFailed {
			record.Stages[stageIndex].Stage.Status = domainpipeline.JobRunFailed
			record.Stages[stageIndex].Stage.FinishedAt = &at
		} else if stageSucceeded {
			record.Stages[stageIndex].Stage.Status = domainpipeline.JobRunSucceeded
			record.Stages[stageIndex].Stage.FinishedAt = &at
		}
	}
	if anyFailed {
		record.Run.Status = domainpipeline.PipelineRunFailed
		record.Run.FinishedAt = &at
		record.Run.UpdatedAt = at
		return record
	}
	if allSucceeded {
		record.Run.Status = domainpipeline.PipelineRunSucceeded
		record.Run.FinishedAt = &at
		record.Run.UpdatedAt = at
	}
	return record
}

func sortLogs(logs []event.LogChunk) []event.LogChunk {
	logs = append([]event.LogChunk(nil), logs...)
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].Sequence < logs[j].Sequence
	})
	return logs
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func normalizeRecordExecutor(executor string) string {
	normalized := domainrunner.NormalizeExecutorCapability(executor)
	if normalized == "" {
		return domainrunner.ExecutorShell
	}
	return normalized
}

func executorListContains(values []string, executor string) bool {
	executor = domainrunner.NormalizeExecutorCapability(executor)
	if !domainrunner.IsSupportedExecutorCapability(executor) {
		return false
	}
	for _, value := range values {
		if domainrunner.NormalizeExecutorCapability(value) == executor && domainrunner.IsSupportedExecutorCapability(value) {
			return true
		}
	}
	return false
}

func runnerSupportsExecutor(runner domainrunner.Runner, executor string) bool {
	return executorListContains(runner.Executors, executor) || executorListContains(runner.Capabilities, executor)
}

func labelsMatch(have map[string]string, want map[string]string) bool {
	for key, value := range want {
		if have[key] != value {
			return false
		}
	}
	return true
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
