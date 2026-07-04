package handlers

import (
	"testing"
	"time"

	domainevent "github.com/sevoniva/nivora/internal/domain/event"
)

func TestBuildAggregateTimelineRedactsSensitiveEventAndLogFields(t *testing.T) {
	now := time.Now().UTC()
	timeline := buildAggregateTimeline(
		[]domainevent.Event{
			{
				ID:      "evt-1",
				Type:    "devops.test",
				Source:  "test",
				Subject: "subject-1",
				Time:    now,
				Data: map[string]any{
					"message":       "bearer secret-token-value",
					"status":        "Succeeded",
					"authorization": "Bearer raw-token",
					"nested": map[string]any{
						"password": "raw-password",
					},
				},
			},
		},
		[]domainevent.LogChunk{
			{
				ID:            "log-1",
				PipelineRunID: "prun-1",
				Stream:        "stdout",
				Sequence:      1,
				Content:       "printed bearer secret-token-value",
				CreatedAt:     now.Add(time.Second),
			},
		},
	)
	if len(timeline) != 2 {
		t.Fatalf("timeline length = %d", len(timeline))
	}
	if timeline[0].Message != "[REDACTED]" {
		t.Fatalf("event message was not redacted: %#v", timeline[0])
	}
	if timeline[0].Data["authorization"] != "[REDACTED]" {
		t.Fatalf("authorization was not redacted: %#v", timeline[0].Data)
	}
	nested, ok := timeline[0].Data["nested"].(map[string]any)
	if !ok || nested["password"] != "[REDACTED]" {
		t.Fatalf("nested secret was not redacted: %#v", timeline[0].Data)
	}
	if timeline[1].Message != "[REDACTED]" {
		t.Fatalf("log message was not redacted: %#v", timeline[1])
	}
}
