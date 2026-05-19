package approval

import "time"

const (
	SubjectRelease    = "release"
	SubjectDeployment = "deployment"
	SubjectPipeline   = "pipeline"

	StatusPending  = "Pending"
	StatusApproved = "Approved"
	StatusRejected = "Rejected"
	StatusCanceled = "Canceled"
	StatusExpired  = "Expired"

	DecisionApprove = "approve"
	DecisionReject  = "reject"
)

type ApprovalRequest struct {
	ID               string                `json:"id" yaml:"id"`
	SubjectType      string                `json:"subjectType" yaml:"subjectType"`
	SubjectID        string                `json:"subjectId" yaml:"subjectId"`
	EnvironmentID    string                `json:"environmentId,omitempty" yaml:"environmentId,omitempty"`
	RequiredByPolicy bool                  `json:"requiredByPolicy" yaml:"requiredByPolicy"`
	Status           string                `json:"status" yaml:"status"`
	RequestedBy      string                `json:"requestedBy,omitempty" yaml:"requestedBy,omitempty"`
	RequestedAt      time.Time             `json:"requestedAt" yaml:"requestedAt"`
	ExpiresAt        *time.Time            `json:"expiresAt,omitempty" yaml:"expiresAt,omitempty"`
	Reason           string                `json:"reason,omitempty" yaml:"reason,omitempty"`
	Participants     []ApprovalParticipant `json:"participants,omitempty" yaml:"participants,omitempty"`
	Decisions        []ApprovalDecision    `json:"decisions,omitempty" yaml:"decisions,omitempty"`
}

type ApprovalDecision struct {
	Approver  string    `json:"approver" yaml:"approver"`
	Decision  string    `json:"decision" yaml:"decision"`
	Comment   string    `json:"comment,omitempty" yaml:"comment,omitempty"`
	DecidedAt time.Time `json:"decidedAt" yaml:"decidedAt"`
}

type ApprovalPolicy struct {
	ID            string   `json:"id" yaml:"id"`
	Name          string   `json:"name" yaml:"name"`
	SubjectType   string   `json:"subjectType,omitempty" yaml:"subjectType,omitempty"`
	EnvironmentID string   `json:"environmentId,omitempty" yaml:"environmentId,omitempty"`
	RequiredRoles []string `json:"requiredRoles,omitempty" yaml:"requiredRoles,omitempty"`
	RequiredCount int      `json:"requiredCount" yaml:"requiredCount"`
}

type ApprovalParticipant struct {
	UserID string `json:"userId,omitempty" yaml:"userId,omitempty"`
	Role   string `json:"role,omitempty" yaml:"role,omitempty"`
}

type ChangeWindow struct {
	ID            string            `json:"id" yaml:"id"`
	Name          string            `json:"name" yaml:"name"`
	EnvironmentID string            `json:"environmentId" yaml:"environmentId"`
	Timezone      string            `json:"timezone" yaml:"timezone"`
	StartTime     string            `json:"startTime" yaml:"startTime"`
	EndTime       string            `json:"endTime" yaml:"endTime"`
	DaysOfWeek    []string          `json:"daysOfWeek,omitempty" yaml:"daysOfWeek,omitempty"`
	Allowed       bool              `json:"allowed" yaml:"allowed"`
	Metadata      map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	CreatedAt     time.Time         `json:"createdAt" yaml:"createdAt"`
	UpdatedAt     time.Time         `json:"updatedAt" yaml:"updatedAt"`
}

type ChangeWindowResult struct {
	WindowID      string    `json:"windowId,omitempty" yaml:"windowId,omitempty"`
	EnvironmentID string    `json:"environmentId" yaml:"environmentId"`
	Allowed       bool      `json:"allowed" yaml:"allowed"`
	Reason        string    `json:"reason" yaml:"reason"`
	EvaluatedAt   time.Time `json:"evaluatedAt" yaml:"evaluatedAt"`
}
