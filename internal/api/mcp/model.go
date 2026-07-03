package mcp

import (
	"encoding/json"
	"time"

	"github.com/sevoniva/nivora/internal/domain/audit"
	domainauth "github.com/sevoniva/nivora/internal/domain/auth"
	"github.com/sevoniva/nivora/internal/infra/config"
	artifactusecase "github.com/sevoniva/nivora/internal/usecase/artifact"
	authusecase "github.com/sevoniva/nivora/internal/usecase/auth"
	complianceusecase "github.com/sevoniva/nivora/internal/usecase/compliance"
	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
	pluginusecase "github.com/sevoniva/nivora/internal/usecase/plugin"
	releaseusecase "github.com/sevoniva/nivora/internal/usecase/releaseorchestration"
	securityusecase "github.com/sevoniva/nivora/internal/usecase/security"
)

const (
	ProtocolVersion = "2025-06-18"

	EventResourceRead   = "mcp.resource.read"
	EventToolCalled     = "mcp.tool.called"
	EventToolDenied     = "mcp.tool.denied"
	EventPromptRendered = "mcp.prompt.rendered"
)

type Services struct {
	Config      config.Config
	Subject     domainauth.Subject
	Auth        *authusecase.Service
	Pipelines   *pipelineusecase.Service
	Deployments *deploymentusecase.Service
	Artifacts   *artifactusecase.Service
	Releases    *releaseusecase.Service
	Security    *securityusecase.Service
	Compliance  *complianceusecase.Service
	Plugins     *pluginusecase.Registry
	Audit       AuditRecorder
}

type AuditRecorder interface {
	RecordMCPAudit(entry audit.AuditLog)
}

type MemoryAuditRecorder struct {
	entries []audit.AuditLog
}

func (r *MemoryAuditRecorder) RecordMCPAudit(entry audit.AuditLog) {
	r.entries = append(r.entries, entry)
}

func (r *MemoryAuditRecorder) Entries() []audit.AuditLog {
	return append([]audit.AuditLog(nil), r.entries...)
}

type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

type ResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType"`
	Text     string `json:"text"`
}

type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"inputSchema,omitempty"`
}

type ToolResult struct {
	Content []ToolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type ToolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type Prompt struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
}

type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

type PromptResult struct {
	Description string          `json:"description,omitempty"`
	Messages    []PromptMessage `json:"messages"`
}

type PromptMessage struct {
	Role    string        `json:"role"`
	Content PromptContent `json:"content"`
}

type PromptContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type OperationError struct {
	Code               string `json:"code"`
	Message            string `json:"message"`
	RequiredFutureGate string `json:"requiredFutureGate,omitempty"`
}

func (e OperationError) Error() string {
	return e.Message
}

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      any           `json:"id,omitempty"`
	Result  any           `json:"result,omitempty"`
	Error   *jsonRPCError `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type auditDecision struct {
	Event    string
	Subject  string
	Scope    string
	Decision string
	Reason   string
}

func newMCPAudit(subject domainauth.Subject, decision auditDecision) audit.AuditLog {
	now := time.Now().UTC()
	return audit.AuditLog{
		ID:          "mcp-audit-" + now.Format("20060102150405.000000000"),
		ActorID:     subject.ID,
		Action:      decision.Event,
		Subject:     decision.Subject,
		SubjectType: "mcp",
		SubjectID:   decision.Subject,
		ScopeType:   decision.Scope,
		Reason:      decision.Reason,
		Metadata: map[string]string{
			"decision": decision.Decision,
		},
		CreatedAt: now,
	}
}
