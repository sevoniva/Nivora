package pipeline

import (
	"context"
	"io"
	"strings"
	"testing"

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
	if err := service.RegisterRunner(context.Background(), domainrunner.Runner{
		ID:        "runner-a",
		Name:      "runner-a",
		Status:    "online",
		Labels:    map[string]string{"tier": "dev"},
		Executors: []string{"shell"},
	}); err != nil {
		t.Fatalf("register runner: %v", err)
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

func hasEvent(events []event.Event, eventType string) bool {
	for _, evt := range events {
		if evt.Type == eventType {
			return true
		}
	}
	return false
}
