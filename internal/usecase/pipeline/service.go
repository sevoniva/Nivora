package pipeline

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/sevoniva/nivora/internal/domain/audit"
	"github.com/sevoniva/nivora/internal/domain/event"
	domainpipeline "github.com/sevoniva/nivora/internal/domain/pipeline"
	"github.com/sevoniva/nivora/internal/ports/eventbus"
	"github.com/sevoniva/nivora/internal/ports/executor"
)

const (
	EventPipelineRunCreated   = "devops.pipeline.run.created"
	EventPipelineRunStarted   = "devops.pipeline.run.started"
	EventPipelineRunCompleted = "devops.pipeline.run.completed"
	EventPipelineRunFailed    = "devops.pipeline.run.failed"

	defaultStepTimeout = 30 * time.Second
)

type Service struct {
	store    Store
	runner   Runner
	eventBus eventbus.EventBus
	now      func() time.Time
}

func NewService(store Store, runner Runner, bus eventbus.EventBus) *Service {
	return &Service{
		store:    store,
		runner:   runner,
		eventBus: bus,
		now:      time.Now,
	}
}

func (s *Service) CreateAndRun(ctx context.Context, input CreateRunInput) (CreateRunResult, error) {
	if err := input.Definition.Validate(); err != nil {
		return CreateRunResult{}, err
	}

	record := s.newRecord(input.Definition)
	if err := s.store.Save(ctx, record); err != nil {
		return CreateRunResult{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventPipelineRunCreated, "PipelineRun created", input.ActorID); err != nil {
		return CreateRunResult{}, err
	}

	record.Run.Status = domainpipeline.PipelineRunQueued
	record.Run.UpdatedAt = s.now()
	if err := s.store.Save(ctx, record); err != nil {
		return CreateRunResult{}, err
	}

	record.Run.Status = domainpipeline.PipelineRunRunning
	started := s.now()
	record.Run.StartedAt = &started
	record.Run.UpdatedAt = started
	if err := s.store.Save(ctx, record); err != nil {
		return CreateRunResult{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventPipelineRunStarted, "PipelineRun started", input.ActorID); err != nil {
		return CreateRunResult{}, err
	}

	record = s.runStages(ctx, input.Definition, record)
	if err := s.store.Save(ctx, record); err != nil {
		return CreateRunResult{}, err
	}

	finalEvent := EventPipelineRunCompleted
	finalMessage := "PipelineRun completed"
	if record.Run.Status == domainpipeline.PipelineRunFailed {
		finalEvent = EventPipelineRunFailed
		finalMessage = "PipelineRun failed"
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, finalEvent, finalMessage, input.ActorID); err != nil {
		return CreateRunResult{}, err
	}

	finalRecord, err := s.store.Get(ctx, record.Run.ID)
	if err != nil {
		return CreateRunResult{}, err
	}
	return CreateRunResult{Record: finalRecord}, nil
}

func (s *Service) Get(ctx context.Context, id string) (RunRecord, error) {
	return s.store.Get(ctx, id)
}

func (s *Service) Logs(ctx context.Context, id string) ([]LogRecord, error) {
	record, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return record.Logs, nil
}

func (s *Service) Events(ctx context.Context, id string) ([]event.Event, error) {
	record, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return record.Events, nil
}

func (s *Service) newRecord(def Definition) RunRecord {
	now := s.now()
	pipelineID := newID("pipe")
	runID := newID("prun")
	record := RunRecord{
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
		stageRecord := StageRecord{
			Stage: domainpipeline.StageRun{
				ID:            stageRunID,
				PipelineRunID: runID,
				Name:          stage.Name,
				Status:        domainpipeline.JobRunPending,
				CreatedAt:     now,
				UpdatedAt:     now,
			},
		}
		for _, job := range stage.Jobs {
			jobRunID := newID("job")
			jobRecord := JobRecord{
				Job: domainpipeline.JobRun{
					ID:         jobRunID,
					StageRunID: stageRunID,
					Name:       job.Name,
					Status:     domainpipeline.JobRunPending,
					RunnerID:   s.runner.ID(),
					CreatedAt:  now,
					UpdatedAt:  now,
				},
			}
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
		record.Stages[stageIndex].Stage.Status = domainpipeline.JobRunRunning
		record.Stages[stageIndex].Stage.StartedAt = &started
		record.Stages[stageIndex].Stage.UpdatedAt = started

		for jobIndex, job := range stage.Jobs {
			record.Stages[stageIndex].Jobs[jobIndex].Job.Status = domainpipeline.JobRunAssigned
			record.Stages[stageIndex].Jobs[jobIndex].Job.UpdatedAt = s.now()
			jobStarted := s.now()
			record.Stages[stageIndex].Jobs[jobIndex].Job.Status = domainpipeline.JobRunRunning
			record.Stages[stageIndex].Jobs[jobIndex].Job.StartedAt = &jobStarted
			record.Stages[stageIndex].Jobs[jobIndex].Job.UpdatedAt = jobStarted

			for stepIndex, step := range job.Steps {
				stepStarted := s.now()
				stepRun := &record.Stages[stageIndex].Jobs[jobIndex].Steps[stepIndex]
				stepRun.Status = domainpipeline.JobRunRunning
				stepRun.StartedAt = &stepStarted
				stepRun.UpdatedAt = stepStarted

				result, err := s.runner.RunShellStep(ctx, record.Stages[stageIndex].Jobs[jobIndex].Job.ID, step.Run, defaultStepTimeout)
				record.Logs = append(record.Logs, s.captureLogs(record.Run.ID, record.Stages[stageIndex].Jobs[jobIndex].Job.ID, stepRun.ID, result)...)

				finished := s.now()
				stepRun.FinishedAt = &finished
				stepRun.UpdatedAt = finished
				if err != nil || result.ExitCode != 0 {
					reason := failureReason(err, result.ExitCode)
					stepRun.Status = domainpipeline.JobRunFailed
					stepRun.FailureReason = reason
					record.Stages[stageIndex].Jobs[jobIndex].Job.Status = domainpipeline.JobRunFailed
					record.Stages[stageIndex].Jobs[jobIndex].Job.FailureReason = reason
					record.Stages[stageIndex].Jobs[jobIndex].Job.FinishedAt = &finished
					record.Stages[stageIndex].Jobs[jobIndex].Job.UpdatedAt = finished
					record.Stages[stageIndex].Stage.Status = domainpipeline.JobRunFailed
					record.Stages[stageIndex].Stage.FailureReason = reason
					record.Stages[stageIndex].Stage.FinishedAt = &finished
					record.Stages[stageIndex].Stage.UpdatedAt = finished
					record.Run.Status = domainpipeline.PipelineRunFailed
					record.Run.FailureReason = reason
					record.Run.FinishedAt = &finished
					record.Run.UpdatedAt = finished
					return record
				}
				stepRun.Status = domainpipeline.JobRunSucceeded
			}

			jobFinished := s.now()
			record.Stages[stageIndex].Jobs[jobIndex].Job.Status = domainpipeline.JobRunSucceeded
			record.Stages[stageIndex].Jobs[jobIndex].Job.FinishedAt = &jobFinished
			record.Stages[stageIndex].Jobs[jobIndex].Job.UpdatedAt = jobFinished
		}
		stageFinished := s.now()
		record.Stages[stageIndex].Stage.Status = domainpipeline.JobRunSucceeded
		record.Stages[stageIndex].Stage.FinishedAt = &stageFinished
		record.Stages[stageIndex].Stage.UpdatedAt = stageFinished
	}

	finished := s.now()
	record.Run.Status = domainpipeline.PipelineRunSucceeded
	record.Run.FinishedAt = &finished
	record.Run.UpdatedAt = finished
	return record
}

func (s *Service) captureLogs(runID string, jobRunID string, stepRunID string, result executor.Result) []LogRecord {
	var logs []LogRecord
	now := s.now()
	if result.Stdout != "" {
		logs = append(logs, LogRecord{
			ID:            newID("log"),
			PipelineRunID: runID,
			JobRunID:      jobRunID,
			StepRunID:     stepRunID,
			Stream:        "stdout",
			Content:       result.Stdout,
			CreatedAt:     now,
		})
	}
	if result.Stderr != "" {
		logs = append(logs, LogRecord{
			ID:            newID("log"),
			PipelineRunID: runID,
			JobRunID:      jobRunID,
			StepRunID:     stepRunID,
			Stream:        "stderr",
			Content:       result.Stderr,
			CreatedAt:     now,
		})
	}
	return logs
}

func (s *Service) recordEventAndAudit(ctx context.Context, runID string, eventType string, auditAction string, actorID string) error {
	now := s.now()
	evt := event.Event{
		SpecVersion:     "1.0",
		ID:              newID("evt"),
		Type:            eventType,
		Source:          "nivora/pipeline",
		Subject:         runID,
		Time:            now,
		DataContentType: "application/json",
		Data: map[string]any{
			"pipelineRunId": runID,
		},
	}
	if err := s.eventBus.Publish(ctx, evt); err != nil {
		return err
	}
	if err := s.store.AppendEvent(ctx, runID, evt); err != nil {
		return err
	}
	return s.store.AppendAudit(ctx, runID, audit.AuditLog{
		ID:        newID("audit"),
		ActorID:   actorID,
		Action:    auditAction,
		Subject:   runID,
		CreatedAt: now,
	})
}

func failureReason(err error, exitCode int) string {
	if err != nil {
		return err.Error()
	}
	return fmt.Sprintf("command exited with code %d", exitCode)
}

func newID(prefix string) string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(b[:])
}
