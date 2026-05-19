package pipeline

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sevoniva/nivora/internal/domain/event"
)

func BenchmarkCreateQueuedPipelineRun(b *testing.B) {
	ctx := context.Background()
	service := newTestService()
	def := testDefinition(`printf "hello"`)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		def.Metadata.Name = fmt.Sprintf("bench-%d", i)
		if _, err := service.CreateQueued(ctx, CreateRunInput{Definition: def}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAppendLogChunk(b *testing.B) {
	ctx := context.Background()
	store := NewMemoryStore()
	service := NewService(store, NewLocalRunner("bench-runner", &fakeExecutor{calls: make(map[string]int)}), fakeBus{})
	created, err := service.CreateQueued(ctx, CreateRunInput{Definition: testDefinition(`printf "hello"`)})
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := store.AppendLog(ctx, created.Record.Run.ID, event.LogChunk{
			ID:            fmt.Sprintf("log-%d", i),
			PipelineRunID: created.Record.Run.ID,
			Stream:        "stdout",
			Content:       "benchmark log line",
			CreatedAt:     time.Now(),
		}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkQueryPipelineTimeline(b *testing.B) {
	ctx := context.Background()
	service := newTestService()
	created, err := service.CreateAndRun(ctx, CreateRunInput{Definition: testDefinition(`printf "hello"`)})
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := service.Timeline(ctx, created.Record.Run.ID); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRunnerHeartbeat(b *testing.B) {
	ctx := context.Background()
	service := newTestService()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := service.HeartbeatRunner(ctx, "test-runner"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJobClaim(b *testing.B) {
	ctx := context.Background()
	service := newTestService()
	def := testDefinition(`printf "hello"`)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		def.Metadata.Name = fmt.Sprintf("claim-%d", i)
		if _, err := service.CreateQueued(ctx, CreateRunInput{Definition: def}); err != nil {
			b.Fatal(err)
		}
		claim, err := service.ClaimJob(ctx, "test-runner", time.Minute)
		if err != nil {
			b.Fatal(err)
		}
		if _, err := service.UpdateJobStatus(ctx, claim.JobRunID, UpdateJobStatusInput{Status: "Succeeded"}); err != nil {
			b.Fatal(err)
		}
	}
}
