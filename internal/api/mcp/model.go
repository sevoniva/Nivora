package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/sevoniva/nivora/internal/domain/audit"
	domainauth "github.com/sevoniva/nivora/internal/domain/auth"
	"github.com/sevoniva/nivora/internal/infra/config"
	"github.com/sevoniva/nivora/internal/infra/crypto"
	artifactusecase "github.com/sevoniva/nivora/internal/usecase/artifact"
	authusecase "github.com/sevoniva/nivora/internal/usecase/auth"
	catalogusecase "github.com/sevoniva/nivora/internal/usecase/catalog"
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
	Config       config.Config
	Subject      domainauth.Subject
	Auth         *authusecase.Service
	Pipelines    *pipelineusecase.Service
	PipelineDefs *pipelineusecase.DefinitionCatalog
	Deployments  *deploymentusecase.Service
	Catalog      *catalogusecase.Service
	Artifacts    *artifactusecase.Service
	Releases     *releaseusecase.Service
	Security     *securityusecase.Service
	Compliance   *complianceusecase.Service
	Plugins      *pluginusecase.Registry
	Audit        AuditRecorder
}

type AuditRecorder interface {
	RecordMCPAudit(ctx context.Context, entry audit.AuditLog) error
}

type MemoryAuditRecorder struct {
	entries []audit.AuditLog
}

func (r *MemoryAuditRecorder) RecordMCPAudit(ctx context.Context, entry audit.AuditLog) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.entries = append(r.entries, entry)
	return nil
}

func (r *MemoryAuditRecorder) Entries() []audit.AuditLog {
	return append([]audit.AuditLog(nil), r.entries...)
}

type ComplianceAuditRecorder struct {
	service *complianceusecase.Service
}

func NewComplianceAuditRecorder(service *complianceusecase.Service) *ComplianceAuditRecorder {
	return &ComplianceAuditRecorder{service: service}
}

func (r *ComplianceAuditRecorder) RecordMCPAudit(ctx context.Context, entry audit.AuditLog) error {
	if r == nil || r.service == nil {
		return nil
	}
	return r.service.RecordAudit(ctx, entry)
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

type RequestMetadata struct {
	RequestID     string
	CorrelationID string
	ClientID      string
	RemoteAddr    string
	Transport     string
}

type requestMetadataKey struct{}

func ContextWithRequestMetadata(ctx context.Context, metadata RequestMetadata) context.Context {
	return context.WithValue(ctx, requestMetadataKey{}, metadata)
}

func requestMetadataFromContext(ctx context.Context) RequestMetadata {
	if ctx == nil {
		return RequestMetadata{}
	}
	if metadata, ok := ctx.Value(requestMetadataKey{}).(RequestMetadata); ok {
		return metadata
	}
	return RequestMetadata{}
}

func newMCPAudit(ctx context.Context, subject domainauth.Subject, decision auditDecision) audit.AuditLog {
	now := time.Now().UTC()
	safeSubject := crypto.RedactString(decision.Subject)
	safeReason := crypto.RedactString(decision.Reason)
	request := requestMetadataFromContext(ctx)
	metadata := map[string]string{
		"auth_mode": subject.AuthMode,
		"decision":  decision.Decision,
		"operation": safeSubject,
	}
	addAuditMetadata(metadata, "transport", request.Transport)
	addAuditMetadata(metadata, "client_id", request.ClientID)
	addAuditMetadata(metadata, "remote_addr", request.RemoteAddr)
	return audit.AuditLog{
		ID:            "mcp-audit-" + now.Format("20060102150405.000000000"),
		ActorID:       subject.ID,
		Action:        decision.Event,
		Subject:       safeSubject,
		SubjectType:   "mcp",
		SubjectID:     safeSubject,
		ScopeType:     decision.Scope,
		ScopeID:       decision.Scope,
		Reason:        safeReason,
		RequestID:     sanitizeAuditMetadataValue(request.RequestID),
		CorrelationID: sanitizeAuditMetadataValue(request.CorrelationID),
		Metadata:      metadata,
		CreatedAt:     now,
	}
}

func addAuditMetadata(metadata map[string]string, key string, value string) {
	if safe := sanitizeAuditMetadataValue(value); safe != "" {
		metadata[key] = safe
	}
}

func sanitizeAuditMetadataValue(value string) string {
	value = strings.TrimSpace(crypto.RedactString(value))
	if len(value) > 128 {
		value = value[:128]
	}
	return value
}
