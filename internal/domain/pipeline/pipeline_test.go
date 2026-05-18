package pipeline

import "testing"

func TestStatuses(t *testing.T) {
	if !PipelineRunRunning.Valid() {
		t.Fatal("expected running pipeline status to be valid")
	}
	if PipelineRunStatus("unknown").Valid() {
		t.Fatal("expected unknown pipeline status to be invalid")
	}
	if !JobRunAssigned.Valid() {
		t.Fatal("expected assigned job status to be valid")
	}
	if JobRunStatus("unknown").Valid() {
		t.Fatal("expected unknown job status to be invalid")
	}
}
