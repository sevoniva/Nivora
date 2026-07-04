package releaseorchestration

import (
	"context"
	"time"
)

type RuntimeRecoveryAction struct {
	SubjectType    string    `json:"subjectType"`
	SubjectID      string    `json:"subjectId"`
	Status         string    `json:"status"`
	Reason         string    `json:"reason,omitempty"`
	SafeNextAction string    `json:"safeNextAction"`
	Automatic      bool      `json:"automatic"`
	ObservedAt     time.Time `json:"observedAt"`
}

type RuntimeRecoverySummary struct {
	NonTerminalReleaseExecutions int                     `json:"nonTerminalReleaseExecutions"`
	StaleReleaseExecutions       int                     `json:"staleReleaseExecutions"`
	Actions                      []RuntimeRecoveryAction `json:"actions,omitempty"`
	Warnings                     []string                `json:"warnings,omitempty"`
}

func (s *Service) RuntimeStatus(ctx context.Context, staleAfter time.Duration, limit int) (RuntimeRecoverySummary, error) {
	if staleAfter <= 0 {
		staleAfter = 5 * time.Minute
	}
	if limit <= 0 {
		limit = 100
	}
	now := s.now()
	nonTerminal, err := s.store.ListNonTerminalReleaseExecutions(ctx, limit)
	if err != nil {
		return RuntimeRecoverySummary{}, err
	}
	stale, err := s.store.ListStaleReleaseExecutions(ctx, now.Add(-staleAfter), limit)
	if err != nil {
		return RuntimeRecoverySummary{}, err
	}
	summary := RuntimeRecoverySummary{
		NonTerminalReleaseExecutions: len(nonTerminal),
		StaleReleaseExecutions:       len(stale),
	}
	for _, record := range stale {
		summary.Actions = append(summary.Actions, RuntimeRecoveryAction{
			SubjectType:    "releaseExecution",
			SubjectID:      record.Execution.ID,
			Status:         string(record.Execution.Status),
			Reason:         record.Execution.Reason,
			SafeNextAction: "Inspect target executions and deployment run IDs before manually resuming or canceling.",
			Automatic:      false,
			ObservedAt:     now,
		})
	}
	return summary, nil
}
