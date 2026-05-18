package pipeline

import (
	"context"
	"testing"
	"time"

	"github.com/sevoniva/nivora/internal/domain/event"
	domainpipeline "github.com/sevoniva/nivora/internal/domain/pipeline"
)

func TestMemoryStoreLogOrdering(t *testing.T) {
	store := NewMemoryStore()
	now := time.Now()
	record := RunRecord{Run: domainpipeline.PipelineRun{
		ID:        "run-logs",
		Status:    domainpipeline.PipelineRunQueued,
		CreatedAt: now,
		UpdatedAt: now,
	}}
	if err := store.Save(context.Background(), record); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := store.AppendLog(context.Background(), "run-logs", event.LogChunk{ID: "log-b", Content: "second"}); err != nil {
		t.Fatalf("append second: %v", err)
	}
	if err := store.AppendLog(context.Background(), "run-logs", event.LogChunk{ID: "log-a", Content: "first"}); err != nil {
		t.Fatalf("append first: %v", err)
	}
	logs, err := store.LogsByPipelineRun(context.Background(), "run-logs")
	if err != nil {
		t.Fatalf("logs: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("logs = %#v", logs)
	}
	if logs[0].Sequence != 1 || logs[1].Sequence != 2 {
		t.Fatalf("sequences = %d, %d", logs[0].Sequence, logs[1].Sequence)
	}
}
