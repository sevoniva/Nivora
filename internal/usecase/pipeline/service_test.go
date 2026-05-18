package pipeline

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/sevoniva/nivora/internal/domain/event"
	domainpipeline "github.com/sevoniva/nivora/internal/domain/pipeline"
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
	if len(result.Record.Logs) != 1 || result.Record.Logs[0].Content != "hello" {
		t.Fatalf("logs = %#v", result.Record.Logs)
	}
	if len(result.Record.Events) != 3 {
		t.Fatalf("events = %d", len(result.Record.Events))
	}
	if len(result.Record.Audits) != 3 {
		t.Fatalf("audits = %d", len(result.Record.Audits))
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

func newTestService() *Service {
	return NewService(NewMemoryStore(), NewLocalRunner("test-runner", fakeExecutor{}), fakeBus{})
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

type fakeExecutor struct{}

func (e fakeExecutor) Prepare(ctx context.Context, job executor.JobContext) error {
	return nil
}

func (e fakeExecutor) Run(ctx context.Context, command executor.Command) (executor.Result, error) {
	if command.Args[1] == `printf "bad" >&2; exit 7` {
		return executor.Result{ExitCode: 7, Stderr: "bad"}, nil
	}
	return executor.Result{ExitCode: 0, Stdout: "hello"}, nil
}

func (e fakeExecutor) Cancel(ctx context.Context, commandID string) error {
	return nil
}

func (e fakeExecutor) Logs(ctx context.Context, commandID string) (io.ReadCloser, error) {
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
