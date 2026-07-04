package pipeline

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	domainapproval "github.com/sevoniva/nivora/internal/domain/approval"
	"github.com/sevoniva/nivora/internal/domain/audit"
	"github.com/sevoniva/nivora/internal/domain/event"
	domainpipeline "github.com/sevoniva/nivora/internal/domain/pipeline"
	domainrunner "github.com/sevoniva/nivora/internal/domain/runner"
	"github.com/sevoniva/nivora/internal/ports/eventbus"
	"github.com/sevoniva/nivora/internal/ports/executor"
)

const (
	EventPipelineRunCreated    = "devops.pipeline.run.created"
	EventPipelineRunQueued     = "devops.pipeline.run.queued"
	EventPipelineRunPaused     = "devops.pipeline.run.paused"
	EventPipelineRunStarted    = "devops.pipeline.run.started"
	EventPipelineRunCompleted  = "devops.pipeline.run.completed"
	EventPipelineRunFailed     = "devops.pipeline.run.failed"
	EventPipelineRunCanceled   = "devops.pipeline.run.canceled"
	EventJobRunAssigned        = "devops.job.run.assigned"
	EventJobRunStarted         = "devops.job.run.started"
	EventJobRunCompleted       = "devops.job.run.completed"
	EventJobRunFailed          = "devops.job.run.failed"
	EventJobRunRetrying        = "devops.job.run.retrying"
	EventJobRunStatusUpdated   = "devops.job.run.status.updated"
	EventJobRunLogAppended     = "devops.job.run.log.appended"
	EventJobRunCancelRequested = "devops.job.run.cancel_requested"
	EventRunnerRegistered      = "devops.runner.registered"
	EventRunnerTokenRotated    = "devops.runner.token.rotated"
	EventRunnerTokenRevoked    = "devops.runner.token.revoked"
	EventRunnerOffline         = "devops.runner.offline"
	EventRunnerHeartbeat       = "devops.runner.heartbeat"
	EventRunnerJobClaimed      = "devops.runner.job.claimed"
	EventPipelineRunRecovered  = "devops.pipeline.run.recovered"
	EventPipelineRunTimedOut   = "devops.pipeline.run.timeout_reconciled"

	defaultStepTimeout  = 30 * time.Second
	defaultRunLease     = 30 * time.Second
	defaultStaleAfter   = 2 * time.Minute
	defaultTimeoutAfter = 30 * time.Minute
)

var ErrRunTerminal = errors.New("pipeline run is already terminal")

type Service struct {
	store    Store
	runner   Runner
	eventBus eventbus.EventBus
	now      func() time.Time
}

func NewService(store Store, runner Runner, bus eventbus.EventBus) *Service {
	service := &Service{
		store:    store,
		runner:   runner,
		eventBus: bus,
		now:      time.Now,
	}
	_ = service.RegisterRunner(context.Background(), domainrunner.Runner{
		ID:        runner.ID(),
		Name:      runner.ID(),
		Status:    "online",
		Labels:    map[string]string{"runtime": "local"},
		Executors: []string{"shell"},
	})
	return service
}

func (s *Service) CreateQueued(ctx context.Context, input CreateRunInput) (CreateRunResult, error) {
	if err := input.Definition.Validate(); err != nil {
		return CreateRunResult{}, err
	}

	record := s.newRecord(input.Definition)
	record.Pipeline.ProjectID = strings.TrimSpace(input.ProjectID)
	if pipelineID := strings.TrimSpace(input.PipelineID); pipelineID != "" {
		record.Pipeline.ID = pipelineID
		record.Run.PipelineID = pipelineID
	}
	if versionID := strings.TrimSpace(input.PipelineVersionID); versionID != "" {
		record.Run.PipelineVersionID = versionID
	}
	record.Run.CorrelationID = input.CorrelationID
	if err := s.store.Save(ctx, record); err != nil {
		return CreateRunResult{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventPipelineRunCreated, "PipelineRun created", input.ActorID, string(record.Run.Status), "PipelineRun created"); err != nil {
		return CreateRunResult{}, err
	}

	now := s.now()
	if err := transitionPipelineRun(&record.Run, domainpipeline.PipelineRunQueued, now, ""); err != nil {
		return CreateRunResult{}, err
	}
	if err := s.store.Save(ctx, record); err != nil {
		return CreateRunResult{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventPipelineRunQueued, "PipelineRun queued", input.ActorID, string(record.Run.Status), "PipelineRun queued"); err != nil {
		return CreateRunResult{}, err
	}

	finalRecord, err := s.store.Get(ctx, record.Run.ID)
	if err != nil {
		return CreateRunResult{}, err
	}
	return CreateRunResult{Record: finalRecord}, nil
}

func (s *Service) CreateAndRun(ctx context.Context, input CreateRunInput) (CreateRunResult, error) {
	created, err := s.CreateQueued(ctx, input)
	if err != nil {
		return CreateRunResult{}, err
	}
	record, err := s.ProcessRun(ctx, created.Record.Run.ID, input.ActorID)
	if err != nil {
		return CreateRunResult{}, err
	}
	return CreateRunResult{Record: record}, nil
}

func (s *Service) ProcessQueued(ctx context.Context, limit int) ([]RunRecord, error) {
	queued, err := s.store.ListByStatus(ctx, domainpipeline.PipelineRunQueued)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > len(queued) {
		limit = len(queued)
	}
	processed := make([]RunRecord, 0, limit)
	for i := 0; i < limit; i++ {
		leased, err := s.store.AcquirePipelineRunLease(ctx, queued[i].Run.ID, "worker-local", s.now().Add(defaultRunLease), s.now())
		if err != nil {
			return processed, err
		}
		record, err := s.ProcessRun(ctx, leased.Run.ID, "")
		if err != nil {
			return processed, err
		}
		processed = append(processed, record)
	}
	return processed, nil
}

func (s *Service) ProcessRun(ctx context.Context, id string, actorID string) (RunRecord, error) {
	record, err := s.store.Get(ctx, id)
	if err != nil {
		return RunRecord{}, err
	}
	if isTerminalPipelineStatus(record.Run.Status) {
		return record, nil
	}
	if record.Run.Status != domainpipeline.PipelineRunQueued {
		return RunRecord{}, fmt.Errorf("pipeline run %s is %s, not Queued", id, record.Run.Status)
	}

	now := s.now()
	if err := transitionPipelineRun(&record.Run, domainpipeline.PipelineRunRunning, now, ""); err != nil {
		return RunRecord{}, err
	}
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventPipelineRunStarted, "PipelineRun started", actorID, string(record.Run.Status), "PipelineRun started"); err != nil {
		return RunRecord{}, err
	}
	record, err = s.store.Get(ctx, record.Run.ID)
	if err != nil {
		return RunRecord{}, err
	}

	record = s.runStages(ctx, record.Definition, record)
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}

	finalEvent := EventPipelineRunCompleted
	finalMessage := "PipelineRun completed"
	if record.Run.Status == domainpipeline.PipelineRunFailed || record.Run.Status == domainpipeline.PipelineRunTimeout {
		finalEvent = EventPipelineRunFailed
		finalMessage = "PipelineRun failed"
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, finalEvent, finalMessage, actorID, string(record.Run.Status), finalMessage); err != nil {
		return RunRecord{}, err
	}
	return s.store.Get(ctx, record.Run.ID)
}

func (s *Service) Cancel(ctx context.Context, id string, actorID string) (RunRecord, error) {
	record, err := s.store.Get(ctx, id)
	if err != nil {
		return RunRecord{}, err
	}
	if isTerminalPipelineStatus(record.Run.Status) {
		return record, ErrRunTerminal
	}
	now := s.now()
	if err := transitionPipelineRun(&record.Run, domainpipeline.PipelineRunCanceled, now, "canceled by request"); err != nil {
		return RunRecord{}, err
	}
	for stageIndex := range record.Stages {
		if !isTerminalJobStatus(record.Stages[stageIndex].Stage.Status) {
			_ = transitionStageRun(&record.Stages[stageIndex].Stage, domainpipeline.JobRunCanceled, now, "canceled by request")
		}
		for jobIndex := range record.Stages[stageIndex].Jobs {
			if !isTerminalJobStatus(record.Stages[stageIndex].Jobs[jobIndex].Job.Status) {
				_ = transitionJobRun(&record.Stages[stageIndex].Jobs[jobIndex].Job, domainpipeline.JobRunCanceled, now, "canceled by request")
			}
			for stepIndex := range record.Stages[stageIndex].Jobs[jobIndex].Steps {
				if !isTerminalJobStatus(record.Stages[stageIndex].Jobs[jobIndex].Steps[stepIndex].Status) {
					_ = transitionStepRun(&record.Stages[stageIndex].Jobs[jobIndex].Steps[stepIndex], domainpipeline.JobRunCanceled, now, "canceled by request")
				}
			}
		}
	}
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, id, EventPipelineRunCanceled, "PipelineRun canceled", actorID, string(record.Run.Status), "PipelineRun canceled"); err != nil {
		return RunRecord{}, err
	}
	return s.store.Get(ctx, id)
}

func (s *Service) PauseForApproval(ctx context.Context, id string, actorID string, reason string) (RunRecord, error) {
	record, err := s.store.Get(ctx, id)
	if err != nil {
		return RunRecord{}, err
	}
	if isTerminalPipelineStatus(record.Run.Status) {
		return record, ErrRunTerminal
	}
	if record.Run.Status != domainpipeline.PipelineRunQueued && record.Run.Status != domainpipeline.PipelineRunRunning {
		return RunRecord{}, fmt.Errorf("pipeline run %s cannot wait for approval from %s", id, record.Run.Status)
	}
	if reason == "" {
		reason = "approval required"
	}
	if err := transitionPipelineRun(&record.Run, domainpipeline.PipelineRunPaused, s.now(), reason); err != nil {
		return RunRecord{}, err
	}
	record.Run.OwnerID = ""
	record.Run.LeaseExpiresAt = nil
	record.Run.HeartbeatAt = nil
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventPipelineRunPaused, "PipelineRun paused for approval", actorID, string(record.Run.Status), reason); err != nil {
		return RunRecord{}, err
	}
	return s.store.Get(ctx, id)
}

func (s *Service) ApplyApprovalDecision(ctx context.Context, id string, approval domainapproval.ApprovalRequest, actorID string) (RunRecord, error) {
	record, err := s.store.Get(ctx, id)
	if err != nil {
		return RunRecord{}, err
	}
	if record.Run.Status != domainpipeline.PipelineRunPaused {
		return RunRecord{}, fmt.Errorf("pipeline run is not waiting for approval")
	}
	if approval.SubjectType != "" && approval.SubjectType != domainapproval.SubjectPipeline {
		return RunRecord{}, fmt.Errorf("approval subject does not match pipeline run")
	}
	if approval.SubjectID != "" && approval.SubjectID != id {
		return RunRecord{}, fmt.Errorf("approval subject does not match pipeline run")
	}
	now := s.now()
	switch approval.Status {
	case domainapproval.StatusApproved:
		if err := transitionPipelineRun(&record.Run, domainpipeline.PipelineRunQueued, now, "approval approved"); err != nil {
			return RunRecord{}, err
		}
		record.Run.FinishedAt = nil
		record.Run.FailureReason = ""
		if err := s.store.Save(ctx, record); err != nil {
			return RunRecord{}, err
		}
		if err := s.recordEventAndAudit(ctx, record.Run.ID, EventPipelineRunQueued, "Pipeline approval approved", actorID, string(record.Run.Status), "Pipeline approval approved; run returned to queue"); err != nil {
			return RunRecord{}, err
		}
		return s.store.Get(ctx, id)
	case domainapproval.StatusRejected, domainapproval.StatusExpired:
		reason := "approval " + strings.ToLower(approval.Status)
		if err := transitionPipelineRun(&record.Run, domainpipeline.PipelineRunFailed, now, reason); err != nil {
			return RunRecord{}, err
		}
		if err := s.store.Save(ctx, record); err != nil {
			return RunRecord{}, err
		}
		if err := s.recordEventAndAudit(ctx, record.Run.ID, EventPipelineRunFailed, "PipelineRun failed", actorID, string(record.Run.Status), reason); err != nil {
			return RunRecord{}, err
		}
		return s.store.Get(ctx, id)
	case domainapproval.StatusCanceled:
		if err := transitionPipelineRun(&record.Run, domainpipeline.PipelineRunCanceled, now, "approval canceled"); err != nil {
			return RunRecord{}, err
		}
		if err := s.store.Save(ctx, record); err != nil {
			return RunRecord{}, err
		}
		if err := s.recordEventAndAudit(ctx, record.Run.ID, EventPipelineRunCanceled, "PipelineRun canceled", actorID, string(record.Run.Status), "approval canceled"); err != nil {
			return RunRecord{}, err
		}
		return s.store.Get(ctx, id)
	default:
		return RunRecord{}, fmt.Errorf("approval must be Approved, Rejected, Expired, or Canceled")
	}
}

func (s *Service) Get(ctx context.Context, id string) (RunRecord, error) {
	return s.store.Get(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]RunRecord, error) {
	return s.store.List(ctx)
}

func (s *Service) ListFiltered(ctx context.Context, scopeType, scopeID string) ([]RunRecord, error) {
	return s.store.ListFiltered(ctx, scopeType, scopeID)
}

func (s *Service) Logs(ctx context.Context, id string) ([]event.LogChunk, error) {
	return s.store.LogsByPipelineRun(ctx, id)
}

func (s *Service) Events(ctx context.Context, id string) ([]event.Event, error) {
	return s.store.EventsByPipelineRun(ctx, id)
}

func (s *Service) Timeline(ctx context.Context, id string) ([]TimelineEntry, error) {
	events, err := s.Events(ctx, id)
	if err != nil {
		return nil, err
	}
	timeline := make([]TimelineEntry, 0, len(events))
	for _, evt := range events {
		entry := TimelineEntry{
			Type:    evt.Type,
			Time:    evt.Time,
			Subject: evt.Subject,
		}
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

func (s *Service) RegisterRunner(ctx context.Context, runner domainrunner.Runner) error {
	_, err := s.RegisterRunnerWithToken(ctx, runner)
	return err
}

func (s *Service) RegisterRunnerWithToken(ctx context.Context, runner domainrunner.Runner) (RegisterRunnerResult, error) {
	now := s.now()
	if runner.ID == "" {
		runner.ID = newID("runner")
	}
	if runner.Name == "" {
		runner.Name = runner.ID
	}
	if runner.Status == "" {
		runner.Status = "online"
	}
	if len(runner.Executors) == 0 {
		runner.Executors = []string{"shell"}
	}
	if len(runner.Capabilities) == 0 {
		runner.Capabilities = append([]string(nil), runner.Executors...)
	}
	if runner.MaxConcurrency <= 0 {
		runner.MaxConcurrency = 1
	}
	if runner.CreatedAt.IsZero() {
		runner.CreatedAt = now
	}
	runner.UpdatedAt = now
	if runner.LastHeartbeatAt == nil {
		runner.LastHeartbeatAt = &now
	}
	if runner.LastSeenAt == nil {
		runner.LastSeenAt = &now
	}
	token := newRunnerToken(now)
	runner.TokenID = token.TokenID
	runner.TokenHash = hashRunnerToken(token.Token)
	runner.TokenCreatedAt = &now
	if err := s.store.RegisterRunner(ctx, runner); err != nil {
		return RegisterRunnerResult{}, err
	}
	evt := s.newEvent(EventRunnerRegistered, runner.ID, runner.Status, "Runner registered")
	if err := s.eventBus.Publish(ctx, evt); err != nil {
		return RegisterRunnerResult{}, err
	}
	if err := s.appendOutbox(ctx, evt); err != nil {
		return RegisterRunnerResult{}, err
	}
	return RegisterRunnerResult{Runner: runner, Token: token}, nil
}

func (s *Service) HeartbeatRunner(ctx context.Context, runnerID string) (domainrunner.Runner, error) {
	runner, err := s.store.Heartbeat(ctx, runnerID, s.now())
	if err != nil {
		return domainrunner.Runner{}, err
	}
	evt := s.newEvent(EventRunnerHeartbeat, runner.ID, runner.Status, "Runner heartbeat")
	if err := s.eventBus.Publish(ctx, evt); err != nil {
		return domainrunner.Runner{}, err
	}
	if err := s.appendOutbox(ctx, evt); err != nil {
		return domainrunner.Runner{}, err
	}
	return runner, nil
}

func (s *Service) RotateRunnerToken(ctx context.Context, runnerID string) (RegisterRunnerResult, error) {
	now := s.now()
	token := newRunnerToken(now)
	runner, err := s.store.RotateRunnerToken(ctx, runnerID, token.TokenID, hashRunnerToken(token.Token), now)
	if err != nil {
		return RegisterRunnerResult{}, err
	}
	evt := s.newEvent(EventRunnerTokenRotated, runner.ID, runner.Status, "Runner token rotated")
	if err := s.eventBus.Publish(ctx, evt); err != nil {
		return RegisterRunnerResult{}, err
	}
	if err := s.appendOutbox(ctx, evt); err != nil {
		return RegisterRunnerResult{}, err
	}
	return RegisterRunnerResult{Runner: runner, Token: token}, nil
}

func (s *Service) RevokeRunnerToken(ctx context.Context, runnerID string) (domainrunner.Runner, error) {
	runner, err := s.store.RevokeRunnerToken(ctx, runnerID, s.now())
	if err != nil {
		return domainrunner.Runner{}, err
	}
	evt := s.newEvent(EventRunnerTokenRevoked, runner.ID, runner.Status, "Runner token revoked")
	if err := s.eventBus.Publish(ctx, evt); err != nil {
		return domainrunner.Runner{}, err
	}
	if err := s.appendOutbox(ctx, evt); err != nil {
		return domainrunner.Runner{}, err
	}
	return runner, nil
}

func (s *Service) ValidateRunnerToken(ctx context.Context, runnerID string, token string) error {
	if token == "" {
		return ErrRunnerUnauthorized
	}
	runner, err := s.store.GetRunner(ctx, runnerID)
	if err != nil {
		return err
	}
	if runner.TokenRevokedAt != nil || runner.TokenHash == "" {
		return ErrRunnerTokenRevoked
	}
	got := hashRunnerToken(token)
	if subtle.ConstantTimeCompare([]byte(got), []byte(runner.TokenHash)) != 1 {
		return ErrRunnerUnauthorized
	}
	return nil
}

func (s *Service) ValidateRunnerJob(ctx context.Context, runnerID string, jobRunID string) error {
	records, err := s.store.List(ctx)
	if err != nil {
		return err
	}
	for _, record := range records {
		for _, stage := range record.Stages {
			for _, job := range stage.Jobs {
				if job.Job.ID == jobRunID && job.Job.RunnerID == runnerID {
					return nil
				}
			}
		}
	}
	return ErrRunnerUnauthorized
}

func (s *Service) MarkOfflineRunners(ctx context.Context, timeout time.Duration) ([]domainrunner.Runner, error) {
	if timeout <= 0 {
		timeout = time.Minute
	}
	now := s.now()
	runners, err := s.store.MarkOfflineRunners(ctx, now.Add(-timeout), now)
	if err != nil {
		return nil, err
	}
	for _, runner := range runners {
		evt := s.newEvent(EventRunnerOffline, runner.ID, runner.Status, "Runner marked offline")
		if err := s.eventBus.Publish(ctx, evt); err != nil {
			return runners, err
		}
		if err := s.appendOutbox(ctx, evt); err != nil {
			return runners, err
		}
	}
	return runners, nil
}

func (s *Service) ListRunners(ctx context.Context) ([]domainrunner.Runner, error) {
	return s.store.ListRunners(ctx)
}

func (s *Service) GetRunner(ctx context.Context, id string) (domainrunner.Runner, error) {
	return s.store.GetRunner(ctx, id)
}

func (s *Service) ClaimJob(ctx context.Context, runnerID string, lease time.Duration) (JobClaim, error) {
	if lease <= 0 {
		lease = 30 * time.Second
	}
	if _, err := s.store.GetRunner(ctx, runnerID); err != nil {
		return JobClaim{}, err
	}
	claim, err := s.store.ClaimJob(ctx, runnerID, s.now().Add(lease))
	if err != nil {
		return JobClaim{}, err
	}
	evt := s.newEvent(EventRunnerJobClaimed, claim.JobRunID, string(claim.Status), "JobRun claimed by runner")
	if err := s.eventBus.Publish(ctx, evt); err != nil {
		return JobClaim{}, err
	}
	if err := s.store.AppendEvent(ctx, claim.PipelineRunID, evt); err != nil {
		return JobClaim{}, err
	}
	if err := s.appendOutbox(ctx, evt); err != nil {
		return JobClaim{}, err
	}
	return claim, nil
}

func (s *Service) AppendJobLog(ctx context.Context, jobRunID string, input AppendJobLogInput) ([]event.LogChunk, error) {
	if input.PipelineRunID == "" {
		return nil, fmt.Errorf("pipelineRunId is required")
	}
	if input.Stream == "" {
		input.Stream = "system"
	}
	record, err := s.store.Get(ctx, input.PipelineRunID)
	if err != nil {
		return nil, err
	}
	if !recordHasJob(record, jobRunID) {
		return nil, ErrJobNotFound
	}
	log := event.LogChunk{
		ID:            newID("log"),
		PipelineRunID: input.PipelineRunID,
		StageRunID:    input.StageRunID,
		JobRunID:      jobRunID,
		StepRunID:     input.StepRunID,
		Stream:        input.Stream,
		Content:       input.Content,
		CreatedAt:     s.now(),
	}
	if err := s.store.AppendLog(ctx, input.PipelineRunID, log); err != nil {
		return nil, err
	}
	evt := s.newEvent(EventJobRunLogAppended, jobRunID, "log_appended", "JobRun log appended")
	if err := s.eventBus.Publish(ctx, evt); err != nil {
		return nil, err
	}
	if err := s.store.AppendEvent(ctx, input.PipelineRunID, evt); err != nil {
		return nil, err
	}
	if err := s.appendOutbox(ctx, evt); err != nil {
		return nil, err
	}
	return s.store.LogsByJobRun(ctx, jobRunID)
}

func (s *Service) UpdateJobStatus(ctx context.Context, jobRunID string, input UpdateJobStatusInput) (RunRecord, error) {
	if !input.Status.Valid() {
		return RunRecord{}, fmt.Errorf("invalid job status %q", input.Status)
	}
	record, err := s.store.UpdateJobStatus(ctx, jobRunID, input.Status, input.Reason, s.now())
	if err != nil {
		return RunRecord{}, err
	}
	if err := s.recordEvent(ctx, record.Run.ID, EventJobRunStatusUpdated, jobRunID, string(input.Status), "JobRun status updated"); err != nil {
		return RunRecord{}, err
	}
	return s.store.Get(ctx, record.Run.ID)
}

func (s *Service) RequestCancel(ctx context.Context, id string, actorID string) (RunRecord, error) {
	record, err := s.store.RequestCancel(ctx, id, s.now())
	if err != nil {
		return RunRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, id, EventJobRunCancelRequested, "PipelineRun cancel requested", actorID, string(record.Run.Status), "PipelineRun cancel requested"); err != nil {
		return RunRecord{}, err
	}
	return s.store.Get(ctx, id)
}

func (s *Service) PendingOutbox(ctx context.Context, limit int) ([]EventOutboxRecord, error) {
	return s.store.ListPendingOutbox(ctx, limit)
}

func (s *Service) PublishPendingOutbox(ctx context.Context, limit int) (int, error) {
	items, err := s.store.ListPendingOutbox(ctx, limit)
	if err != nil {
		return 0, err
	}
	published := 0
	var firstErr error
	for _, item := range items {
		if err := s.eventBus.Publish(ctx, item.Payload); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			next := s.now().Add(outboxBackoff(item.RetryCount + 1))
			_ = s.store.MarkOutboxFailed(ctx, item.ID, item.RetryCount+1, next, err.Error())
			continue
		}
		if err := s.store.MarkOutboxPublished(ctx, item.ID, s.now()); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		published++
	}
	return published, firstErr
}

func (s *Service) RuntimeStatus(ctx context.Context) (RuntimeRecoverySummary, error) {
	now := s.now()
	queued, err := s.store.ListByStatus(ctx, domainpipeline.PipelineRunQueued)
	if err != nil {
		return RuntimeRecoverySummary{}, err
	}
	stale, err := s.store.ListStaleRunningPipelineRuns(ctx, now.Add(-defaultStaleAfter), 100)
	if err != nil {
		return RuntimeRecoverySummary{}, err
	}
	expiredClaims, err := s.store.ListExpiredJobClaims(ctx, now, 100)
	if err != nil {
		return RuntimeRecoverySummary{}, err
	}
	outbox, err := s.store.ListPendingOutbox(ctx, 100)
	if err != nil {
		return RuntimeRecoverySummary{}, err
	}
	all, err := s.store.List(ctx)
	if err != nil {
		return RuntimeRecoverySummary{}, err
	}
	summary := RuntimeRecoverySummary{
		WorkerID:                 "status",
		QueuedPipelineRuns:       len(queued),
		StaleRunningPipelineRuns: len(stale),
		ExpiredJobClaims:         len(expiredClaims),
		PendingOutboxEvents:      len(outbox),
		CheckedAt:                now,
	}
	for _, item := range outbox {
		if item.Status == "failed" {
			summary.FailedOutboxEvents++
		}
	}
	for _, record := range all {
		if record.Run.CancelRequested && !isTerminalPipelineStatus(record.Run.Status) {
			summary.CancelRequestedPipelineRuns++
		}
		if record.Run.Status == domainpipeline.PipelineRunRunning && record.Run.UpdatedAt.Before(now.Add(-defaultTimeoutAfter)) {
			summary.TimedOutPipelineRuns++
		}
	}
	return summary, nil
}

func (s *Service) ReconcileRuntime(ctx context.Context, options RuntimeRecoveryOptions) (RuntimeRecoverySummary, error) {
	options = defaultRecoveryOptions(options)
	now := s.now()
	summary := RuntimeRecoverySummary{WorkerID: options.WorkerID, CheckedAt: now}

	offline, err := s.MarkOfflineRunners(ctx, time.Minute)
	if err != nil {
		summary.Warnings = append(summary.Warnings, err.Error())
	} else {
		summary.OfflineRunners = len(offline)
	}

	all, err := s.store.List(ctx)
	if err != nil {
		return summary, err
	}
	for _, record := range all {
		if record.Run.CancelRequested && !isTerminalPipelineStatus(record.Run.Status) {
			if _, err := s.Cancel(ctx, record.Run.ID, options.WorkerID); err != nil && !errors.Is(err, ErrRunTerminal) {
				summary.Warnings = append(summary.Warnings, err.Error())
				continue
			}
			summary.CancelRequestedPipelineRuns++
		}
	}

	stale, err := s.store.ListStaleRunningPipelineRuns(ctx, now.Add(-options.StaleAfter), options.ProcessLimit)
	if err != nil {
		return summary, err
	}
	summary.StaleRunningPipelineRuns = len(stale)
	for _, record := range stale {
		if record.Run.Status != domainpipeline.PipelineRunRunning || record.Run.CancelRequested {
			continue
		}
		if record.Run.UpdatedAt.Before(now.Add(-options.TimeoutAfter)) {
			if err := s.timeoutRun(ctx, record, "pipeline run exceeded runtime timeout"); err != nil {
				summary.Warnings = append(summary.Warnings, err.Error())
				continue
			}
			summary.TimedOutPipelineRuns++
			continue
		}
		if err := s.requeueRun(ctx, record, "worker lease expired; run returned to queue"); err != nil {
			summary.Warnings = append(summary.Warnings, err.Error())
			continue
		}
		summary.RecoveredPipelineRuns++
	}

	expiredClaims, err := s.store.ListExpiredJobClaims(ctx, now, options.ProcessLimit)
	if err != nil {
		return summary, err
	}
	summary.ExpiredJobClaims = len(expiredClaims)
	for _, claim := range expiredClaims {
		record, err := s.store.Get(ctx, claim.PipelineRunID)
		if err != nil {
			summary.Warnings = append(summary.Warnings, err.Error())
			continue
		}
		if err := s.requeueRun(ctx, record, "job lease expired; run returned to queue"); err != nil {
			summary.Warnings = append(summary.Warnings, err.Error())
			continue
		}
		summary.RecoveredPipelineRuns++
	}

	queued, err := s.store.ListByStatus(ctx, domainpipeline.PipelineRunQueued)
	if err != nil {
		return summary, err
	}
	summary.QueuedPipelineRuns = len(queued)
	limit := options.ProcessLimit
	if limit <= 0 || limit > len(queued) {
		limit = len(queued)
	}
	for i := 0; i < limit; i++ {
		leased, err := s.store.AcquirePipelineRunLease(ctx, queued[i].Run.ID, options.WorkerID, s.now().Add(options.LeaseDuration), s.now())
		if err != nil {
			summary.Warnings = append(summary.Warnings, err.Error())
			continue
		}
		if _, err := s.ProcessRun(ctx, leased.Run.ID, options.WorkerID); err != nil {
			summary.Warnings = append(summary.Warnings, err.Error())
			continue
		}
		summary.ProcessedPipelineRuns++
	}

	published, err := s.PublishPendingOutbox(ctx, options.OutboxLimit)
	summary.PublishedOutboxEvents = published
	if err != nil {
		summary.Warnings = append(summary.Warnings, err.Error())
	}
	pending, listErr := s.store.ListPendingOutbox(ctx, options.OutboxLimit)
	if listErr == nil {
		summary.PendingOutboxEvents = len(pending)
		for _, item := range pending {
			if item.Status == "failed" {
				summary.FailedOutboxEvents++
			}
		}
	}
	return summary, nil
}

func (s *Service) newRecord(def Definition) RunRecord {
	now := s.now()
	pipelineID := newID("pipe")
	runID := newID("prun")
	record := RunRecord{
		Definition: def,
		Pipeline: domainpipeline.Pipeline{
			ID:        pipelineID,
			Name:      def.Metadata.Name,
			CreatedAt: now,
			UpdatedAt: now,
		},
		Run: domainpipeline.PipelineRun{
			ID:         runID,
			PipelineID: pipelineID,
			Status:     domainpipeline.PipelineRunPending,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
	}
	for _, stage := range def.Spec.Stages {
		stageRunID := newID("stage")
		stageRecord := StageRecord{Stage: domainpipeline.StageRun{
			ID:            stageRunID,
			PipelineRunID: runID,
			Name:          stage.Name,
			Status:        domainpipeline.JobRunPending,
			CreatedAt:     now,
			UpdatedAt:     now,
		}}
		for _, job := range stage.Jobs {
			jobRunID := newID("job")
			jobRecord := JobRecord{Job: domainpipeline.JobRun{
				ID:         jobRunID,
				StageRunID: stageRunID,
				Name:       job.Name,
				Status:     domainpipeline.JobRunPending,
				MaxRetries: job.Retries,
				Attempt:    1,
				CreatedAt:  now,
				UpdatedAt:  now,
			}}
			for i, step := range job.Steps {
				name := step.Name
				if name == "" {
					name = fmt.Sprintf("step-%d", i+1)
				}
				jobRecord.Steps = append(jobRecord.Steps, domainpipeline.StepRun{
					ID:        newID("step"),
					JobRunID:  jobRunID,
					Name:      name,
					Status:    domainpipeline.JobRunPending,
					Attempt:   1,
					CreatedAt: now,
					UpdatedAt: now,
				})
			}
			stageRecord.Jobs = append(stageRecord.Jobs, jobRecord)
		}
		record.Stages = append(record.Stages, stageRecord)
	}
	return record
}

func (s *Service) runStages(ctx context.Context, def Definition, record RunRecord) RunRecord {
	for stageIndex, stage := range def.Spec.Stages {
		started := s.now()
		_ = transitionStageRun(&record.Stages[stageIndex].Stage, domainpipeline.JobRunRunning, started, "")
		for jobIndex, job := range stage.Jobs {
			record.Stages[stageIndex].Jobs[jobIndex].Job.RunnerID = s.selectRunnerID(ctx)
			_ = transitionJobRun(&record.Stages[stageIndex].Jobs[jobIndex].Job, domainpipeline.JobRunAssigned, s.now(), "")
			_ = s.recordRuntimeEvent(ctx, &record, EventJobRunAssigned, record.Stages[stageIndex].Jobs[jobIndex].Job.ID, string(domainpipeline.JobRunAssigned), "JobRun assigned")

			jobSucceeded := false
			for attempt := 1; attempt <= job.Retries+1; attempt++ {
				record.Stages[stageIndex].Jobs[jobIndex].Job.Attempt = attempt
				_ = transitionJobRun(&record.Stages[stageIndex].Jobs[jobIndex].Job, domainpipeline.JobRunRunning, s.now(), "")
				_ = s.recordRuntimeEvent(ctx, &record, EventJobRunStarted, record.Stages[stageIndex].Jobs[jobIndex].Job.ID, string(domainpipeline.JobRunRunning), "JobRun started")
				if s.runJobAttempt(ctx, &record, stageIndex, jobIndex, job, attempt, attempt > job.Retries) {
					jobSucceeded = true
					break
				}
				if attempt <= job.Retries {
					record.Stages[stageIndex].Jobs[jobIndex].Job.FinishedAt = nil
					record.Stages[stageIndex].Jobs[jobIndex].Job.FailureReason = ""
					_ = transitionJobRun(&record.Stages[stageIndex].Jobs[jobIndex].Job, domainpipeline.JobRunAssigned, s.now(), "")
				}
			}
			if !jobSucceeded {
				finished := s.now()
				record.Stages[stageIndex].Stage.Status = domainpipeline.JobRunFailed
				record.Stages[stageIndex].Stage.FinishedAt = &finished
				record.Stages[stageIndex].Stage.UpdatedAt = finished
				record.Stages[stageIndex].Stage.FailureReason = record.Stages[stageIndex].Jobs[jobIndex].Job.FailureReason
				status := domainpipeline.PipelineRunFailed
				if isTimeout(record.Stages[stageIndex].Jobs[jobIndex].Job.FailureReason) {
					status = domainpipeline.PipelineRunTimeout
				}
				_ = transitionPipelineRun(&record.Run, status, finished, record.Stages[stageIndex].Jobs[jobIndex].Job.FailureReason)
				return record
			}
		}
		_ = transitionStageRun(&record.Stages[stageIndex].Stage, domainpipeline.JobRunSucceeded, s.now(), "")
	}
	_ = transitionPipelineRun(&record.Run, domainpipeline.PipelineRunSucceeded, s.now(), "")
	return record
}

func (s *Service) runJobAttempt(ctx context.Context, record *RunRecord, stageIndex int, jobIndex int, job Job, attempt int, finalAttempt bool) bool {
	jobRun := &record.Stages[stageIndex].Jobs[jobIndex].Job
	for stepIndex, step := range job.Steps {
		stepRun := &record.Stages[stageIndex].Jobs[jobIndex].Steps[stepIndex]
		stepRun.Attempt = attempt
		if stepRun.Status == domainpipeline.JobRunFailed || stepRun.Status == domainpipeline.JobRunSucceeded {
			stepRun.Status = domainpipeline.JobRunPending
			stepRun.StartedAt = nil
			stepRun.FinishedAt = nil
			stepRun.FailureReason = ""
		}
		_ = transitionStepRun(stepRun, domainpipeline.JobRunRunning, s.now(), "")
		result, err := s.runner.RunShellStep(ctx, jobRun.ID, step.Run, timeoutFor(job, step))
		for _, log := range s.logChunks(record.Run.ID, record.Stages[stageIndex].Stage.ID, jobRun.ID, stepRun.ID, result, int64(len(record.Logs)+1)) {
			record.Logs = append(record.Logs, log)
		}
		finished := s.now()
		if err != nil || result.ExitCode != 0 {
			reason := failureReason(err, result.ExitCode)
			_ = transitionStepRun(stepRun, domainpipeline.JobRunFailed, finished, reason)
			next := domainpipeline.JobRunRetrying
			if finalAttempt {
				next = domainpipeline.JobRunFailed
			}
			_ = transitionJobRun(jobRun, next, finished, reason)
			eventType := EventJobRunRetrying
			if finalAttempt {
				eventType = EventJobRunFailed
			}
			_ = s.recordRuntimeEvent(ctx, record, eventType, jobRun.ID, string(next), reason)
			return false
		}
		_ = transitionStepRun(stepRun, domainpipeline.JobRunSucceeded, finished, "")
	}
	_ = transitionJobRun(jobRun, domainpipeline.JobRunSucceeded, s.now(), "")
	_ = s.recordRuntimeEvent(ctx, record, EventJobRunCompleted, jobRun.ID, string(domainpipeline.JobRunSucceeded), "JobRun completed")
	return true
}

func (s *Service) selectRunnerID(ctx context.Context) string {
	runner, err := s.store.SelectRunner(ctx, "shell", nil)
	if err != nil {
		return s.runner.ID()
	}
	return runner.ID
}

func (s *Service) logChunks(runID string, stageRunID string, jobRunID string, stepRunID string, result executor.Result, startSequence int64) []event.LogChunk {
	var logs []event.LogChunk
	now := s.now()
	if result.Stdout != "" {
		logs = append(logs, event.LogChunk{
			ID:            newID("log"),
			PipelineRunID: runID,
			StageRunID:    stageRunID,
			JobRunID:      jobRunID,
			StepRunID:     stepRunID,
			Stream:        "stdout",
			Sequence:      startSequence + int64(len(logs)),
			Content:       result.Stdout,
			CreatedAt:     now,
		})
	}
	if result.Stderr != "" {
		logs = append(logs, event.LogChunk{
			ID:            newID("log"),
			PipelineRunID: runID,
			StageRunID:    stageRunID,
			JobRunID:      jobRunID,
			StepRunID:     stepRunID,
			Stream:        "stderr",
			Sequence:      startSequence + int64(len(logs)),
			Content:       result.Stderr,
			CreatedAt:     now,
		})
	}
	return logs
}

func (s *Service) recordEventAndAudit(ctx context.Context, runID string, eventType string, auditAction string, actorID string, status string, message string) error {
	if err := s.recordEvent(ctx, runID, eventType, runID, status, message); err != nil {
		return err
	}
	return s.store.AppendAudit(ctx, runID, audit.AuditLog{
		ID:        newID("audit"),
		ActorID:   actorID,
		Action:    auditAction,
		Subject:   runID,
		CreatedAt: s.now(),
	})
}

func (s *Service) recordRuntimeEvent(ctx context.Context, record *RunRecord, eventType string, subject string, status string, message string) error {
	evt := s.newEvent(eventType, subject, status, message)
	if err := s.eventBus.Publish(ctx, evt); err != nil {
		return err
	}
	if err := s.store.AppendEvent(ctx, record.Run.ID, evt); err != nil {
		return err
	}
	if err := s.appendOutbox(ctx, evt); err != nil {
		return err
	}
	record.Events = append(record.Events, evt)
	return nil
}

func (s *Service) recordEvent(ctx context.Context, runID string, eventType string, subject string, status string, message string) error {
	evt := s.newEvent(eventType, subject, status, message)
	if err := s.eventBus.Publish(ctx, evt); err != nil {
		return err
	}
	if err := s.store.AppendEvent(ctx, runID, evt); err != nil {
		return err
	}
	return s.appendOutbox(ctx, evt)
}

func (s *Service) newEvent(eventType string, subject string, status string, message string) event.Event {
	return event.Event{
		SpecVersion:     "1.0",
		ID:              newID("evt"),
		Type:            eventType,
		Source:          "nivora/pipeline",
		Subject:         subject,
		Time:            s.now(),
		DataContentType: "application/json",
		Data: map[string]any{
			"status":  status,
			"message": message,
		},
	}
}

func (s *Service) appendOutbox(ctx context.Context, evt event.Event) error {
	return s.store.AppendOutbox(ctx, EventOutboxRecord{
		ID:        newID("outbox"),
		EventType: evt.Type,
		Subject:   evt.Subject,
		Payload:   evt,
		Status:    "pending",
		CreatedAt: s.now(),
	})
}

func (s *Service) requeueRun(ctx context.Context, record RunRecord, reason string) error {
	now := s.now()
	record.Run.Status = domainpipeline.PipelineRunQueued
	record.Run.OwnerID = ""
	record.Run.LeaseExpiresAt = nil
	record.Run.HeartbeatAt = nil
	record.Run.FailureReason = ""
	record.Run.UpdatedAt = now
	for stageIndex := range record.Stages {
		stage := &record.Stages[stageIndex].Stage
		if stage.Status == domainpipeline.JobRunRunning || stage.Status == domainpipeline.JobRunAssigned {
			stage.Status = domainpipeline.JobRunPending
			stage.FinishedAt = nil
			stage.FailureReason = ""
			stage.UpdatedAt = now
		}
		for jobIndex := range record.Stages[stageIndex].Jobs {
			job := &record.Stages[stageIndex].Jobs[jobIndex].Job
			if job.Status == domainpipeline.JobRunAssigned || job.Status == domainpipeline.JobRunRunning || job.Status == domainpipeline.JobRunRetrying {
				job.Status = domainpipeline.JobRunPending
				job.RunnerID = ""
				job.LeaseExpiresAt = nil
				job.FinishedAt = nil
				job.FailureReason = ""
				job.UpdatedAt = now
			}
			for stepIndex := range record.Stages[stageIndex].Jobs[jobIndex].Steps {
				step := &record.Stages[stageIndex].Jobs[jobIndex].Steps[stepIndex]
				if step.Status == domainpipeline.JobRunRunning {
					step.Status = domainpipeline.JobRunPending
					step.FinishedAt = nil
					step.FailureReason = ""
					step.UpdatedAt = now
				}
			}
		}
	}
	if err := s.store.Save(ctx, record); err != nil {
		return err
	}
	return s.recordEvent(ctx, record.Run.ID, EventPipelineRunRecovered, record.Run.ID, string(record.Run.Status), reason)
}

func (s *Service) timeoutRun(ctx context.Context, record RunRecord, reason string) error {
	now := s.now()
	if err := transitionPipelineRun(&record.Run, domainpipeline.PipelineRunTimeout, now, reason); err != nil {
		return err
	}
	for stageIndex := range record.Stages {
		if !isTerminalJobStatus(record.Stages[stageIndex].Stage.Status) {
			_ = transitionStageRun(&record.Stages[stageIndex].Stage, domainpipeline.JobRunFailed, now, reason)
		}
		for jobIndex := range record.Stages[stageIndex].Jobs {
			if !isTerminalJobStatus(record.Stages[stageIndex].Jobs[jobIndex].Job.Status) {
				_ = transitionJobRun(&record.Stages[stageIndex].Jobs[jobIndex].Job, domainpipeline.JobRunFailed, now, reason)
			}
			for stepIndex := range record.Stages[stageIndex].Jobs[jobIndex].Steps {
				if !isTerminalJobStatus(record.Stages[stageIndex].Jobs[jobIndex].Steps[stepIndex].Status) {
					_ = transitionStepRun(&record.Stages[stageIndex].Jobs[jobIndex].Steps[stepIndex], domainpipeline.JobRunFailed, now, reason)
				}
			}
		}
	}
	if err := s.store.Save(ctx, record); err != nil {
		return err
	}
	return s.recordEventAndAudit(ctx, record.Run.ID, EventPipelineRunTimedOut, "PipelineRun timeout reconciled", "worker", string(record.Run.Status), reason)
}

func defaultRecoveryOptions(options RuntimeRecoveryOptions) RuntimeRecoveryOptions {
	if options.WorkerID == "" {
		options.WorkerID = "worker-local"
	}
	if options.LeaseDuration <= 0 {
		options.LeaseDuration = defaultRunLease
	}
	if options.StaleAfter <= 0 {
		options.StaleAfter = defaultStaleAfter
	}
	if options.TimeoutAfter <= 0 {
		options.TimeoutAfter = defaultTimeoutAfter
	}
	if options.ProcessLimit <= 0 {
		options.ProcessLimit = 10
	}
	if options.OutboxLimit <= 0 {
		options.OutboxLimit = 100
	}
	return options
}

func outboxBackoff(retry int) time.Duration {
	if retry < 1 {
		retry = 1
	}
	if retry > 5 {
		retry = 5
	}
	return time.Duration(retry) * time.Minute
}

func timeoutFor(job Job, step Step) time.Duration {
	seconds := step.TimeoutSeconds
	if seconds == 0 {
		seconds = job.TimeoutSeconds
	}
	if seconds <= 0 {
		return defaultStepTimeout
	}
	return time.Duration(seconds) * time.Second
}

func failureReason(err error, exitCode int) string {
	if err != nil {
		return err.Error()
	}
	return fmt.Sprintf("command exited with code %d", exitCode)
}

func isTimeout(reason string) bool {
	return reason == context.DeadlineExceeded.Error()
}

func recordHasJob(record RunRecord, jobRunID string) bool {
	for _, stage := range record.Stages {
		for _, job := range stage.Jobs {
			if job.Job.ID == jobRunID {
				return true
			}
		}
	}
	return false
}

func newID(prefix string) string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(b[:])
}

func newRunnerToken(now time.Time) RunnerToken {
	return RunnerToken{TokenID: newID("rtok"), Token: "nvr_runner_" + randomHex(24), IssuedAt: now}
}

func hashRunnerToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func randomHex(size int) string {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}
