package pipeline

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	domainapproval "github.com/sevoniva/nivora/internal/domain/approval"
	"github.com/sevoniva/nivora/internal/domain/event"
	domainpipeline "github.com/sevoniva/nivora/internal/domain/pipeline"
	domainrunner "github.com/sevoniva/nivora/internal/domain/runner"
	"github.com/sevoniva/nivora/internal/ports/executor"
)

func TestCreateAndRunSuccess(t *testing.T) {
	service := newTestService()
	result, err := service.CreateAndRun(context.Background(), CreateRunInput{Definition: testDefinition(`printf "hello"`)})
	if err != nil {
		t.Fatalf("create and run: %v", err)
	}
	if result.Record.Run.Status != domainpipeline.PipelineRunSucceeded {
		t.Fatalf("status = %s", result.Record.Run.Status)
	}
	if result.Record.Run.StartedAt == nil || result.Record.Run.FinishedAt == nil {
		t.Fatal("expected start and finish timestamps")
	}
	if result.Record.Stages[0].Stage.Status != domainpipeline.JobRunSucceeded {
		t.Fatalf("stage status = %s", result.Record.Stages[0].Stage.Status)
	}
	if result.Record.Stages[0].Jobs[0].Steps[0].Status != domainpipeline.JobRunSucceeded {
		t.Fatalf("step status = %s", result.Record.Stages[0].Jobs[0].Steps[0].Status)
	}
	if len(result.Record.Logs) != 1 || result.Record.Logs[0].Content != "hello" {
		t.Fatalf("logs = %#v", result.Record.Logs)
	}
	if len(result.Record.Events) != 7 {
		t.Fatalf("events = %d", len(result.Record.Events))
	}
	if len(result.Record.Audits) != 4 {
		t.Fatalf("audits = %d", len(result.Record.Audits))
	}
	if result.Record.Logs[0].Sequence != 1 {
		t.Fatalf("log sequence = %d", result.Record.Logs[0].Sequence)
	}
}

func TestCreateQueuedPersistsProjectScope(t *testing.T) {
	service := newTestService()
	result, err := service.CreateQueued(context.Background(), CreateRunInput{
		Definition: testDefinition(`printf "hello"`),
		ProjectID:  " project-a ",
	})
	if err != nil {
		t.Fatalf("create queued: %v", err)
	}
	if result.Record.Pipeline.ProjectID != "project-a" {
		t.Fatalf("project id = %q", result.Record.Pipeline.ProjectID)
	}

	own, err := service.ListFiltered(context.Background(), "project", "project-a")
	if err != nil {
		t.Fatalf("list own project: %v", err)
	}
	if len(own) != 1 || own[0].Run.ID != result.Record.Run.ID {
		t.Fatalf("own project list = %#v", own)
	}
	other, err := service.ListFiltered(context.Background(), "project", "project-b")
	if err != nil {
		t.Fatalf("list other project: %v", err)
	}
	if len(other) != 0 {
		t.Fatalf("other project list should be empty, got %#v", other)
	}
}

func TestCreateAndRunFailure(t *testing.T) {
	service := newTestService()
	result, err := service.CreateAndRun(context.Background(), CreateRunInput{Definition: testDefinition(`printf "bad" >&2; exit 7`)})
	if err != nil {
		t.Fatalf("create and run: %v", err)
	}
	if result.Record.Run.Status != domainpipeline.PipelineRunFailed {
		t.Fatalf("status = %s", result.Record.Run.Status)
	}
	if result.Record.Run.FailureReason == "" {
		t.Fatal("expected failure reason")
	}
	if len(result.Record.Logs) != 1 || result.Record.Logs[0].Stream != "stderr" || result.Record.Logs[0].Content != "bad" {
		t.Fatalf("logs = %#v", result.Record.Logs)
	}
	if got := result.Record.Events[len(result.Record.Events)-1].Type; got != EventPipelineRunFailed {
		t.Fatalf("last event = %s", got)
	}
}

func TestCreateAndRunRetrySuccess(t *testing.T) {
	service := newTestService()
	def := testDefinition(`flaky`)
	def.Spec.Stages[0].Jobs[0].Retries = 1
	result, err := service.CreateAndRun(context.Background(), CreateRunInput{Definition: def})
	if err != nil {
		t.Fatalf("create and run: %v", err)
	}
	if result.Record.Run.Status != domainpipeline.PipelineRunSucceeded {
		t.Fatalf("status = %s", result.Record.Run.Status)
	}
	job := result.Record.Stages[0].Jobs[0].Job
	if job.Attempt != 2 {
		t.Fatalf("attempt = %d", job.Attempt)
	}
	if !hasEvent(result.Record.Events, EventJobRunRetrying) {
		t.Fatal("expected retrying event")
	}
}

func TestCreateAndRunTimeout(t *testing.T) {
	service := newTestService()
	def := testDefinition(`timeout`)
	def.Spec.Stages[0].Jobs[0].Steps[0].TimeoutSeconds = 1
	result, err := service.CreateAndRun(context.Background(), CreateRunInput{Definition: def})
	if err != nil {
		t.Fatalf("create and run: %v", err)
	}
	if result.Record.Run.Status != domainpipeline.PipelineRunTimeout {
		t.Fatalf("status = %s", result.Record.Run.Status)
	}
}

func TestCancelQueuedPipelineRun(t *testing.T) {
	service := newTestService()
	created, err := service.CreateQueued(context.Background(), CreateRunInput{Definition: testDefinition(`printf "hello"`)})
	if err != nil {
		t.Fatalf("create queued: %v", err)
	}
	canceled, err := service.Cancel(context.Background(), created.Record.Run.ID, "tester")
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if canceled.Run.Status != domainpipeline.PipelineRunCanceled {
		t.Fatalf("status = %s", canceled.Run.Status)
	}
	if !hasEvent(canceled.Events, EventPipelineRunCanceled) {
		t.Fatal("expected canceled event")
	}
}

func TestApprovalApprovedRequeuesPausedPipelineRun(t *testing.T) {
	service := newTestService()
	created, err := service.CreateQueued(context.Background(), CreateRunInput{Definition: testDefinition(`printf "hello"`)})
	if err != nil {
		t.Fatalf("create queued: %v", err)
	}
	paused, err := service.PauseForApproval(context.Background(), created.Record.Run.ID, "policy", "manual approval required")
	if err != nil {
		t.Fatalf("pause for approval: %v", err)
	}
	if paused.Run.Status != domainpipeline.PipelineRunPaused {
		t.Fatalf("paused status = %s", paused.Run.Status)
	}
	if !hasEvent(paused.Events, EventPipelineRunPaused) {
		t.Fatalf("expected paused event: %#v", paused.Events)
	}
	resumed, err := service.ApplyApprovalDecision(context.Background(), created.Record.Run.ID, domainapproval.ApprovalRequest{
		SubjectType: domainapproval.SubjectPipeline,
		SubjectID:   created.Record.Run.ID,
		Status:      domainapproval.StatusApproved,
	}, "reviewer")
	if err != nil {
		t.Fatalf("apply approval: %v", err)
	}
	if resumed.Run.Status != domainpipeline.PipelineRunQueued {
		t.Fatalf("resumed status = %s", resumed.Run.Status)
	}
	if resumed.Run.FailureReason != "" {
		t.Fatalf("failure reason should be cleared, got %q", resumed.Run.FailureReason)
	}
	if !hasEvent(resumed.Events, EventPipelineRunQueued) {
		t.Fatalf("expected queued event: %#v", resumed.Events)
	}
}

func TestApprovalRejectedFailsPausedPipelineRun(t *testing.T) {
	service := newTestService()
	created, err := service.CreateQueued(context.Background(), CreateRunInput{Definition: testDefinition(`printf "hello"`)})
	if err != nil {
		t.Fatalf("create queued: %v", err)
	}
	if _, err := service.PauseForApproval(context.Background(), created.Record.Run.ID, "policy", "manual approval required"); err != nil {
		t.Fatalf("pause for approval: %v", err)
	}
	failed, err := service.ApplyApprovalDecision(context.Background(), created.Record.Run.ID, domainapproval.ApprovalRequest{
		SubjectType: domainapproval.SubjectPipeline,
		SubjectID:   created.Record.Run.ID,
		Status:      domainapproval.StatusRejected,
	}, "reviewer")
	if err != nil {
		t.Fatalf("apply rejection: %v", err)
	}
	if failed.Run.Status != domainpipeline.PipelineRunFailed || !strings.Contains(failed.Run.FailureReason, "approval rejected") {
		t.Fatalf("failed run = %#v", failed.Run)
	}
	if !hasEvent(failed.Events, EventPipelineRunFailed) {
		t.Fatalf("expected failed event: %#v", failed.Events)
	}
}

func TestApprovalSubjectMismatchDoesNotResumePipelineRun(t *testing.T) {
	service := newTestService()
	created, err := service.CreateQueued(context.Background(), CreateRunInput{Definition: testDefinition(`printf "hello"`)})
	if err != nil {
		t.Fatalf("create queued: %v", err)
	}
	if _, err := service.PauseForApproval(context.Background(), created.Record.Run.ID, "policy", "manual approval required"); err != nil {
		t.Fatalf("pause for approval: %v", err)
	}
	_, err = service.ApplyApprovalDecision(context.Background(), created.Record.Run.ID, domainapproval.ApprovalRequest{
		SubjectType: domainapproval.SubjectDeployment,
		SubjectID:   created.Record.Run.ID,
		Status:      domainapproval.StatusApproved,
	}, "reviewer")
	if err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("expected subject type mismatch error, got %v", err)
	}

	_, err = service.ApplyApprovalDecision(context.Background(), created.Record.Run.ID, domainapproval.ApprovalRequest{
		SubjectType: domainapproval.SubjectPipeline,
		SubjectID:   "other-run",
		Status:      domainapproval.StatusApproved,
	}, "reviewer")
	if err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("expected subject mismatch error, got %v", err)
	}
}

func TestProcessQueuedAndTimeline(t *testing.T) {
	service := newTestService()
	created, err := service.CreateQueued(context.Background(), CreateRunInput{Definition: testDefinition(`printf "hello"`)})
	if err != nil {
		t.Fatalf("create queued: %v", err)
	}
	processed, err := service.ProcessQueued(context.Background(), 1)
	if err != nil {
		t.Fatalf("process queued: %v", err)
	}
	if len(processed) != 1 || processed[0].Run.ID != created.Record.Run.ID {
		t.Fatalf("processed = %#v", processed)
	}
	timeline, err := service.Timeline(context.Background(), created.Record.Run.ID)
	if err != nil {
		t.Fatalf("timeline: %v", err)
	}
	if len(timeline) == 0 || timeline[0].Type != EventPipelineRunCreated {
		t.Fatalf("timeline = %#v", timeline)
	}
}

func TestRunnerRegistrationHeartbeatAndSelection(t *testing.T) {
	service := newTestService()
	result, err := service.RegisterRunnerWithToken(context.Background(), domainrunner.Runner{
		ID:        "runner-a",
		Name:      "runner-a",
		Status:    "online",
		Labels:    map[string]string{"tier": "dev"},
		Executors: []string{"shell"},
	})
	if err != nil {
		t.Fatalf("register runner: %v", err)
	}
	if result.Token.Token == "" || result.Runner.TokenHash != hashRunnerToken(result.Token.Token) {
		t.Fatalf("token result = %#v runner = %#v", result.Token, result.Runner)
	}
	if err := service.ValidateRunnerToken(context.Background(), "runner-a", "wrong"); !errors.Is(err, ErrRunnerUnauthorized) {
		t.Fatalf("wrong token error = %v", err)
	}
	if err := service.ValidateRunnerToken(context.Background(), "runner-a", result.Token.Token); err != nil {
		t.Fatalf("validate token: %v", err)
	}
	rotated, err := service.RotateRunnerToken(context.Background(), "runner-a")
	if err != nil {
		t.Fatalf("rotate token: %v", err)
	}
	if rotated.Token.Token == result.Token.Token {
		t.Fatal("expected rotated token to change")
	}
	if err := service.ValidateRunnerToken(context.Background(), "runner-a", result.Token.Token); !errors.Is(err, ErrRunnerUnauthorized) {
		t.Fatalf("old token error = %v", err)
	}
	if _, err := service.RevokeRunnerToken(context.Background(), "runner-a"); err != nil {
		t.Fatalf("revoke token: %v", err)
	}
	if err := service.ValidateRunnerToken(context.Background(), "runner-a", rotated.Token.Token); !errors.Is(err, ErrRunnerTokenRevoked) {
		t.Fatalf("revoked token error = %v", err)
	}
	runner, err := service.HeartbeatRunner(context.Background(), "runner-a")
	if err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
	if runner.LastHeartbeatAt == nil {
		t.Fatal("expected heartbeat timestamp")
	}
	runners, err := service.ListRunners(context.Background())
	if err != nil {
		t.Fatalf("list runners: %v", err)
	}
	if len(runners) < 2 {
		t.Fatalf("runners = %#v", runners)
	}
	stale := runner
	past := time.Now().Add(-2 * time.Minute)
	stale.LastHeartbeatAt = &past
	stale.Status = "online"
	if err := service.store.RegisterRunner(context.Background(), stale); err != nil {
		t.Fatalf("save stale runner: %v", err)
	}
	offline, err := service.MarkOfflineRunners(context.Background(), time.Minute)
	if err != nil {
		t.Fatalf("mark offline: %v", err)
	}
	if len(offline) != 1 || offline[0].ID != "runner-a" {
		t.Fatalf("offline = %#v", offline)
	}
}

func TestRunnerClaimRespectsConcurrency(t *testing.T) {
	service := newTestService()
	result, err := service.RegisterRunnerWithToken(context.Background(), domainrunner.Runner{
		ID:             "runner-limited",
		Name:           "runner-limited",
		Status:         "online",
		Executors:      []string{"shell"},
		MaxConcurrency: 1,
	})
	if err != nil {
		t.Fatalf("register runner: %v", err)
	}
	if err := service.ValidateRunnerToken(context.Background(), result.Runner.ID, result.Token.Token); err != nil {
		t.Fatalf("validate token: %v", err)
	}
	if _, err := service.CreateQueued(context.Background(), CreateRunInput{Definition: testDefinition(`printf "one"`)}); err != nil {
		t.Fatalf("create queued one: %v", err)
	}
	if _, err := service.CreateQueued(context.Background(), CreateRunInput{Definition: testDefinition(`printf "two"`)}); err != nil {
		t.Fatalf("create queued two: %v", err)
	}
	if _, err := service.ClaimJob(context.Background(), "runner-limited", time.Minute); err != nil {
		t.Fatalf("claim first: %v", err)
	}
	if _, err := service.ClaimJob(context.Background(), "runner-limited", time.Minute); !errors.Is(err, ErrRunnerConcurrencyLimit) {
		t.Fatalf("second claim error = %v", err)
	}
}

func TestRunnerClaimLogStatusCancelAndOutbox(t *testing.T) {
	service := newTestService()
	created, err := service.CreateQueued(context.Background(), CreateRunInput{Definition: testDefinition(`printf "hello"`)})
	if err != nil {
		t.Fatalf("create queued: %v", err)
	}
	claim, err := service.ClaimJob(context.Background(), "test-runner", time.Minute)
	if err != nil {
		t.Fatalf("claim job: %v", err)
	}
	if claim.PipelineRunID != created.Record.Run.ID || claim.JobRunID == "" || claim.RunnerID != "test-runner" {
		t.Fatalf("claim = %#v", claim)
	}
	logs, err := service.AppendJobLog(context.Background(), claim.JobRunID, AppendJobLogInput{
		PipelineRunID: claim.PipelineRunID,
		StageRunID:    claim.StageRunID,
		StepRunID:     claim.StepRunIDs[0],
		Stream:        "stdout",
		Content:       "hello from runner",
	})
	if err != nil {
		t.Fatalf("append job log: %v", err)
	}
	if len(logs) != 1 || logs[0].Sequence != 1 {
		t.Fatalf("logs = %#v", logs)
	}
	updated, err := service.UpdateJobStatus(context.Background(), claim.JobRunID, UpdateJobStatusInput{Status: domainpipeline.JobRunRunning})
	if err != nil {
		t.Fatalf("update status running: %v", err)
	}
	if updated.Stages[0].Jobs[0].Job.Status != domainpipeline.JobRunRunning {
		t.Fatalf("job = %#v", updated.Stages[0].Jobs[0].Job)
	}
	canceled, err := service.RequestCancel(context.Background(), claim.PipelineRunID, "tester")
	if err != nil {
		t.Fatalf("request cancel: %v", err)
	}
	if !canceled.Run.CancelRequested {
		t.Fatalf("cancel requested not set: %#v", canceled.Run)
	}
	pending, err := service.PendingOutbox(context.Background(), 100)
	if err != nil {
		t.Fatalf("pending outbox: %v", err)
	}
	if len(pending) == 0 {
		t.Fatal("expected pending outbox records")
	}
	published, err := service.PublishPendingOutbox(context.Background(), 100)
	if err != nil {
		t.Fatalf("publish outbox: %v", err)
	}
	if published == 0 {
		t.Fatal("expected published outbox records")
	}
}

func TestReconcileRuntimeRecoversExpiredLease(t *testing.T) {
	service := newTestService()
	created, err := service.CreateQueued(context.Background(), CreateRunInput{Definition: testDefinition(`printf "hello"`)})
	if err != nil {
		t.Fatalf("create queued: %v", err)
	}
	past := time.Now().Add(-10 * time.Minute)
	record := created.Record
	record.Run.Status = domainpipeline.PipelineRunRunning
	record.Run.OwnerID = "dead-worker"
	record.Run.LeaseExpiresAt = &past
	record.Run.UpdatedAt = past
	record.Stages[0].Stage.Status = domainpipeline.JobRunRunning
	record.Stages[0].Jobs[0].Job.Status = domainpipeline.JobRunAssigned
	record.Stages[0].Jobs[0].Job.RunnerID = "dead-runner"
	record.Stages[0].Jobs[0].Job.LeaseExpiresAt = &past
	if err := service.store.Save(context.Background(), record); err != nil {
		t.Fatalf("save stale run: %v", err)
	}
	summary, err := service.ReconcileRuntime(context.Background(), RuntimeRecoveryOptions{
		WorkerID:     "worker-recovery",
		StaleAfter:   time.Minute,
		TimeoutAfter: time.Hour,
		ProcessLimit: 1,
	})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if summary.RecoveredPipelineRuns == 0 {
		t.Fatalf("summary = %#v", summary)
	}
	recovered, err := service.Get(context.Background(), record.Run.ID)
	if err != nil {
		t.Fatalf("get recovered: %v", err)
	}
	if recovered.Run.Status != domainpipeline.PipelineRunSucceeded {
		t.Fatalf("status = %s", recovered.Run.Status)
	}
	if !hasEvent(recovered.Events, EventPipelineRunRecovered) {
		t.Fatal("expected recovered event")
	}
}

func TestReconcileRuntimeCancellationAndTimeout(t *testing.T) {
	service := newTestService()
	cancelRun, err := service.CreateQueued(context.Background(), CreateRunInput{Definition: testDefinition(`printf "hello"`)})
	if err != nil {
		t.Fatalf("create cancel run: %v", err)
	}
	if _, err := service.RequestCancel(context.Background(), cancelRun.Record.Run.ID, "tester"); err != nil {
		t.Fatalf("request cancel: %v", err)
	}
	timeoutRun, err := service.CreateQueued(context.Background(), CreateRunInput{Definition: testDefinition(`printf "hello"`)})
	if err != nil {
		t.Fatalf("create timeout run: %v", err)
	}
	stale := timeoutRun.Record
	past := time.Now().Add(-2 * time.Hour)
	stale.Run.Status = domainpipeline.PipelineRunRunning
	stale.Run.UpdatedAt = past
	if err := service.store.Save(context.Background(), stale); err != nil {
		t.Fatalf("save stale timeout: %v", err)
	}
	summary, err := service.ReconcileRuntime(context.Background(), RuntimeRecoveryOptions{
		WorkerID:     "worker-recovery",
		StaleAfter:   time.Minute,
		TimeoutAfter: time.Minute,
		ProcessLimit: 10,
	})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if summary.CancelRequestedPipelineRuns != 1 || summary.TimedOutPipelineRuns != 1 {
		t.Fatalf("summary = %#v", summary)
	}
	canceled, err := service.Get(context.Background(), cancelRun.Record.Run.ID)
	if err != nil {
		t.Fatalf("get canceled: %v", err)
	}
	if canceled.Run.Status != domainpipeline.PipelineRunCanceled {
		t.Fatalf("cancel status = %s", canceled.Run.Status)
	}
	timedOut, err := service.Get(context.Background(), stale.Run.ID)
	if err != nil {
		t.Fatalf("get timeout: %v", err)
	}
	if timedOut.Run.Status != domainpipeline.PipelineRunTimeout {
		t.Fatalf("timeout status = %s", timedOut.Run.Status)
	}
}

func TestPublishPendingOutboxFailureSchedulesRetry(t *testing.T) {
	store := NewMemoryStore()
	service := NewService(store, NewLocalRunner("test-runner", &fakeExecutor{calls: make(map[string]int)}), fakeBus{})
	created, err := service.CreateQueued(context.Background(), CreateRunInput{Definition: testDefinition(`printf "hello"`)})
	if err != nil {
		t.Fatalf("create queued: %v", err)
	}
	_, _ = service.ProcessRun(context.Background(), created.Record.Run.ID, "")
	service.eventBus = failingBus{}
	published, err := service.PublishPendingOutbox(context.Background(), 100)
	if err == nil {
		t.Fatal("expected publish error")
	}
	if published != 0 {
		t.Fatalf("published = %d", published)
	}
	foundFailed := false
	store.mu.RLock()
	defer store.mu.RUnlock()
	for _, item := range store.outbox {
		if item.Status == "failed" && item.RetryCount > 0 && item.NextAttemptAt != nil && item.LastError != "" {
			foundFailed = true
			break
		}
	}
	if !foundFailed {
		t.Fatalf("outbox = %#v", store.outbox)
	}
}

func newTestService() *Service {
	return NewService(NewMemoryStore(), NewLocalRunner("test-runner", &fakeExecutor{calls: make(map[string]int)}), fakeBus{})
}

func testDefinition(command string) Definition {
	return Definition{
		APIVersion: "nivora.io/v1alpha1",
		Kind:       "Pipeline",
		Metadata:   Metadata{Name: "test"},
		Spec: Spec{Stages: []Stage{{
			Name: "build",
			Jobs: []Job{{
				Name:     "job",
				Executor: "shell",
				Steps: []Step{{
					Name: "step",
					Run:  command,
				}},
			}},
		}}},
	}
}

type fakeExecutor struct {
	calls map[string]int
}

func (e *fakeExecutor) Prepare(ctx context.Context, job executor.JobContext) error {
	return nil
}

func (e *fakeExecutor) Run(ctx context.Context, command executor.Command) (executor.Result, error) {
	script := command.Args[1]
	e.calls[script]++
	switch script {
	case `printf "bad" >&2; exit 7`:
		return executor.Result{ExitCode: 7, Stderr: "bad"}, nil
	case "flaky":
		if e.calls[script] == 1 {
			return executor.Result{ExitCode: 1, Stderr: "try again"}, nil
		}
		return executor.Result{ExitCode: 0, Stdout: "hello"}, nil
	case "timeout":
		return executor.Result{ExitCode: -1, Stderr: "deadline"}, context.DeadlineExceeded
	}
	return executor.Result{ExitCode: 0, Stdout: "hello"}, nil
}

func (e *fakeExecutor) Cancel(ctx context.Context, commandID string) error {
	return nil
}

func (e *fakeExecutor) Logs(ctx context.Context, commandID string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}

type fakeBus struct{}

func (b fakeBus) Publish(ctx context.Context, evt event.Event) error {
	return nil
}

func (b fakeBus) Subscribe(ctx context.Context, eventType string) (<-chan event.Event, error) {
	ch := make(chan event.Event)
	close(ch)
	return ch, nil
}

type failingBus struct{}

func (b failingBus) Publish(ctx context.Context, evt event.Event) error {
	return errors.New("publish failed")
}

func (b failingBus) Subscribe(ctx context.Context, eventType string) (<-chan event.Event, error) {
	ch := make(chan event.Event)
	close(ch)
	return ch, nil
}

func hasEvent(events []event.Event, eventType string) bool {
	for _, evt := range events {
		if evt.Type == eventType {
			return true
		}
	}
	return false
}
