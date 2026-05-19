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
	ErrRunNotFound    = errors.New("pipeline run not found")
	ErrRunnerNotFound = errors.New("runner not found")
	ErrJobNotFound    = errors.New("job run not found")
	ErrNoClaimableJob = errors.New("no claimable job found")
	ErrOutboxNotFound = errors.New("event outbox record not found")
)

type PipelineRepository interface {
	Save(ctx context.Context, record RunRecord) error
}

type PipelineRunRepository interface {
	Get(ctx context.Context, id string) (RunRecord, error)
	List(ctx context.Context) ([]RunRecord, error)
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
}

type EventRepository interface {
	AppendEvent(ctx context.Context, runID string, evt event.Event) error
	EventsByPipelineRun(ctx context.Context, runID string) ([]event.Event, error)
}

type EventOutboxRepository interface {
	AppendOutbox(ctx context.Context, item EventOutboxRecord) error
	ListPendingOutbox(ctx context.Context, limit int) ([]EventOutboxRecord, error)
	MarkOutboxPublished(ctx context.Context, id string, at time.Time) error
}

type AuditRepository interface {
	AppendAudit(ctx context.Context, runID string, entry audit.AuditLog) error
	AuditBySubject(ctx context.Context, subject string) ([]audit.AuditLog, error)
}

type RunnerRepository interface {
	RegisterRunner(ctx context.Context, runner domainrunner.Runner) error
	Heartbeat(ctx context.Context, runnerID string, at time.Time) (domainrunner.Runner, error)
	GetRunner(ctx context.Context, id string) (domainrunner.Runner, error)
	ListRunners(ctx context.Context) ([]domainrunner.Runner, error)
	SelectRunner(ctx context.Context, executor string, labels map[string]string) (domainrunner.Runner, error)
}

type Store interface {
	PipelineRepository
	PipelineRunRepository
	LogRepository
	JobRepository
	EventRepository
	EventOutboxRepository
	AuditRepository
	RunnerRepository
}

type MemoryStore struct {
	mu      sync.RWMutex
	runs    map[string]RunRecord
	runners map[string]domainrunner.Runner
	outbox  map[string]EventOutboxRecord
	nextSeq map[string]int64
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		runs:    make(map[string]RunRecord),
		runners: make(map[string]domainrunner.Runner),
		outbox:  make(map[string]EventOutboxRecord),
		nextSeq: make(map[string]int64),
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
	runIDs := make([]string, 0, len(s.runs))
	for id := range s.runs {
		runIDs = append(runIDs, id)
	}
	sort.Strings(runIDs)
	for _, runID := range runIDs {
		record := s.runs[runID]
		if record.Run.Status != domainpipeline.PipelineRunQueued && record.Run.Status != domainpipeline.PipelineRunRunning {
			continue
		}
		if record.Run.Status == domainpipeline.PipelineRunQueued {
			record.Run.Status = domainpipeline.PipelineRunRunning
			record.Run.StartedAt = &now
			record.Run.UpdatedAt = now
		}
		for stageIndex := range record.Stages {
			stage := &record.Stages[stageIndex].Stage
			if stage.Status == domainpipeline.JobRunPending {
				stage.Status = domainpipeline.JobRunRunning
				stage.StartedAt = &now
				stage.UpdatedAt = now
			}
			for jobIndex := range record.Stages[stageIndex].Jobs {
				job := &record.Stages[stageIndex].Jobs[jobIndex].Job
				claimable := job.Status == domainpipeline.JobRunPending || job.Status == domainpipeline.JobRunRetrying
				leaseExpired := job.Status == domainpipeline.JobRunAssigned && job.LeaseExpiresAt != nil && job.LeaseExpiresAt.Before(now)
				if !claimable && !leaseExpired {
					continue
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
	for _, item := range s.outbox {
		if item.Status == "pending" {
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
		runners = append(runners, cloneRunner(runner))
	}
	sort.Slice(runners, func(i, j int) bool {
		return runners[i].ID < runners[j].ID
	})
	return runners, nil
}

func (s *MemoryStore) SelectRunner(ctx context.Context, executor string, labels map[string]string) (domainrunner.Runner, error) {
	runners, err := s.ListRunners(ctx)
	if err != nil {
		return domainrunner.Runner{}, err
	}
	for _, runner := range runners {
		if runner.Status != "online" {
			continue
		}
		if !contains(runner.Executors, executor) {
			continue
		}
		if !labelsMatch(runner.Labels, labels) {
			continue
		}
		return runner, nil
	}
	return domainrunner.Runner{}, ErrRunnerNotFound
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
	return record
}

func cloneSpecStages(stages []Stage) []Stage {
	out := append([]Stage(nil), stages...)
	for i := range out {
		out[i].Jobs = append([]Job(nil), out[i].Jobs...)
		for j := range out[i].Jobs {
			out[i].Jobs[j].Steps = append([]Step(nil), out[i].Jobs[j].Steps...)
		}
	}
	return out
}

func cloneRunner(runner domainrunner.Runner) domainrunner.Runner {
	runner.Labels = cloneMap(runner.Labels)
	runner.Executors = append([]string(nil), runner.Executors...)
	return runner
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
