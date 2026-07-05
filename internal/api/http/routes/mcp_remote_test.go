package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	apimiddleware "github.com/sevoniva/nivora/internal/api/http/middleware"
	"github.com/sevoniva/nivora/internal/infra/config"
	complianceusecase "github.com/sevoniva/nivora/internal/usecase/compliance"
)

func TestRemoteMCPDisabledByDefault(t *testing.T) {
	router := newTestRouter(config.Default())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mcp/rpc", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), "mcp_remote_disabled") {
		t.Fatalf("disabled remote MCP status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestRemoteMCPRequiresBearerIdentity(t *testing.T) {
	cfg := config.Default()
	cfg.MCP.Enabled = true
	cfg.MCP.Mode = "http"
	router := newTestRouter(cfg)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/mcp/rpc", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), "mcp_bearer_required") {
		t.Fatalf("anonymous/dev remote MCP status = %d body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/mcp/rpc", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`))
	req.Header.Set("X-Nivora-Runner-Token", "nvr_runner_placeholder")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "mcp_runner_token_denied") {
		t.Fatalf("runner-token remote MCP status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestRemoteMCPJSONRPCWithBearerToken(t *testing.T) {
	cfg := remoteMCPTestConfig(t)
	router := newTestRouter(cfg)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/mcp/rpc", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"resources/list"}`))
	req.Header.Set("Authorization", "Bearer remote-mcp-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("remote MCP status = %d body = %s", rec.Code, rec.Body.String())
	}
	var response struct {
		Result struct {
			Resources []struct {
				URI string `json:"uri"`
			} `json:"resources"`
		} `json:"result"`
		Error any `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode remote MCP response: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("remote MCP error = %#v body = %s", response.Error, rec.Body.String())
	}
	found := false
	for _, resource := range response.Result.Resources {
		if resource.URI == "nivora://runtime/recovery" {
			found = true
		}
	}
	if !found {
		t.Fatalf("runtime recovery resource missing: %s", rec.Body.String())
	}
}

func TestRemoteMCPRecordsAuditAttribution(t *testing.T) {
	cfg := remoteMCPTestConfig(t)
	router, complianceService := newTestRouterWithCompliance(cfg)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/mcp/rpc", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"nivora_status"}}`))
	req.Header.Set("Authorization", "Bearer remote-mcp-token")
	req.Header.Set(apimiddleware.HeaderRequestID, "req-mcp-123")
	req.Header.Set(apimiddleware.HeaderCorrelationID, "corr-mcp-456")
	req.Header.Set("X-Nivora-MCP-Client", "codex-test-client")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"result"`) {
		t.Fatalf("remote MCP status call = %d body = %s", rec.Code, rec.Body.String())
	}

	audits, err := complianceService.SearchAudit(req.Context(), complianceusecase.AuditSearchInput{
		ActorID: "service-account",
		Action:  "mcp.tool.called",
	})
	if err != nil {
		t.Fatalf("search MCP audit: %v", err)
	}
	if len(audits.Items) != 1 {
		t.Fatalf("expected one MCP audit record, got %#v", audits.Items)
	}
	entry := audits.Items[0]
	if entry.SubjectType != "mcp" || entry.SubjectID != "nivora_status" || entry.Metadata["decision"] != "allowed" || entry.Metadata["operation"] != "nivora_status" {
		t.Fatalf("unexpected MCP audit entry = %#v", entry)
	}
	if entry.RequestID != "req-mcp-123" || entry.CorrelationID != "corr-mcp-456" {
		t.Fatalf("remote MCP audit missing request metadata: %#v", entry)
	}
	if entry.Metadata["transport"] != "http" || entry.Metadata["client_id"] != "codex-test-client" || entry.Metadata["remote_addr"] == "" {
		t.Fatalf("remote MCP audit missing client metadata: %#v", entry.Metadata)
	}
	entryBody, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal MCP audit entry: %v", err)
	}
	body := rec.Body.String() + string(entryBody)
	if strings.Contains(body, "remote-mcp-token") {
		t.Fatalf("remote MCP audit or response leaked bearer token: %s", body)
	}
}

func TestRemoteMCPRecordsDeniedAuditAttribution(t *testing.T) {
	cfg := config.Default()
	cfg.MCP.Enabled = true
	cfg.MCP.Mode = "http"
	router, complianceService := newTestRouterWithCompliance(cfg)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/mcp/rpc", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`))
	req.Header.Set(apimiddleware.HeaderRequestID, "req-mcp-denied")
	req.Header.Set(apimiddleware.HeaderCorrelationID, "corr-mcp-denied")
	req.Header.Set("X-Nivora-MCP-Client", "codex-denied-client")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), "mcp_bearer_required") {
		t.Fatalf("remote MCP dev denial status = %d body = %s", rec.Code, rec.Body.String())
	}

	audits, err := complianceService.SearchAudit(req.Context(), complianceusecase.AuditSearchInput{
		ActorID:   "local-admin",
		Action:    "mcp.tool.denied",
		SubjectID: "remote_mcp_auth",
	})
	if err != nil {
		t.Fatalf("search denied MCP audit: %v", err)
	}
	if len(audits.Items) != 1 {
		t.Fatalf("expected one denied MCP audit record, got %#v", audits.Items)
	}
	entry := audits.Items[0]
	if entry.Metadata["decision"] != "denied" || entry.Metadata["client_id"] != "codex-denied-client" || entry.Metadata["transport"] != "http" {
		t.Fatalf("denied MCP audit metadata = %#v", entry)
	}
	if entry.RequestID != "req-mcp-denied" || entry.CorrelationID != "corr-mcp-denied" {
		t.Fatalf("denied MCP audit missing request metadata: %#v", entry)
	}
}

func TestRemoteMCPRecordsRunnerTokenDenialWithoutLeakage(t *testing.T) {
	cfg := config.Default()
	cfg.MCP.Enabled = true
	cfg.MCP.Mode = "http"
	router, complianceService := newTestRouterWithCompliance(cfg)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/mcp/rpc", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`))
	req.Header.Set("Authorization", "Bearer nvr_runner_should_not_leak")
	req.Header.Set("X-Nivora-MCP-Client", "runner-denied-client")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "mcp_runner_token_denied") {
		t.Fatalf("remote MCP runner-token denial status = %d body = %s", rec.Code, rec.Body.String())
	}

	audits, err := complianceService.SearchAudit(req.Context(), complianceusecase.AuditSearchInput{
		ActorID:   "runner-token",
		Action:    "mcp.tool.denied",
		SubjectID: "remote_mcp_auth",
	})
	if err != nil {
		t.Fatalf("search runner-token denial MCP audit: %v", err)
	}
	if len(audits.Items) != 1 {
		t.Fatalf("expected one runner-token denial MCP audit record, got %#v", audits.Items)
	}
	body, err := json.Marshal(audits.Items[0])
	if err != nil {
		t.Fatalf("marshal denied MCP audit: %v", err)
	}
	combined := rec.Body.String() + string(body)
	if strings.Contains(combined, "nvr_runner_should_not_leak") {
		t.Fatalf("remote MCP runner-token denial leaked token: %s", combined)
	}
	if audits.Items[0].Metadata["decision"] != "denied" || audits.Items[0].Metadata["client_id"] != "runner-denied-client" {
		t.Fatalf("runner-token denial audit metadata = %#v", audits.Items[0])
	}
}

func TestRemoteMCPActionAndBodyLimit(t *testing.T) {
	cfg := remoteMCPTestConfig(t)
	cfg.MCP.MaxRequestBytes = 64
	router := newTestRouter(cfg)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/mcp/rpc", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"nivora_apply_deployment","arguments":{"authorization":"Bearer should-not-leak"}}}`))
	req.Header.Set("Authorization", "Bearer remote-mcp-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "mcp_request_too_large") {
		t.Fatalf("large remote MCP request status = %d body = %s", rec.Code, rec.Body.String())
	}

	cfg.MCP.MaxRequestBytes = 1024
	router = newTestRouter(cfg)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/mcp/rpc", strings.NewReader(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"nivora_apply_deployment","arguments":{"authorization":"Bearer should-not-leak"}}}`))
	req.Header.Set("Authorization", "Bearer remote-mcp-token")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "mcp_action_not_allowed") {
		t.Fatalf("blocked remote MCP action status = %d body = %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "should-not-leak") {
		t.Fatalf("remote MCP blocked action leaked sensitive argument: %s", rec.Body.String())
	}
}

func TestRemoteMCPStructuredJSONRPCErrors(t *testing.T) {
	cfg := remoteMCPTestConfig(t)
	router := newTestRouter(cfg)

	cases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "unknown method",
			body: `{"jsonrpc":"2.0","id":1,"method":"unknown/method"}`,
			want: "mcp_method_not_found",
		},
		{
			name: "unknown resource",
			body: `{"jsonrpc":"2.0","id":2,"method":"resources/read","params":{"uri":"nivora://missing"}}`,
			want: "mcp_resource_not_found",
		},
		{
			name: "unknown tool",
			body: `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"nivora_missing_tool","arguments":{"token":"should-not-leak"}}}`,
			want: "mcp_tool_not_found",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/mcp/rpc", strings.NewReader(tc.body))
			req.Header.Set("Authorization", "Bearer remote-mcp-token")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), tc.want) {
				t.Fatalf("%s status = %d body = %s", tc.name, rec.Code, rec.Body.String())
			}
			if strings.Contains(rec.Body.String(), "should-not-leak") {
				t.Fatalf("%s leaked sensitive argument: %s", tc.name, rec.Body.String())
			}
		})
	}
}

func TestRemoteMCPRateLimitAndResponseCap(t *testing.T) {
	cfg := remoteMCPTestConfig(t)
	cfg.MCP.MaxRequestsPerMinute = 1
	router := newTestRouter(cfg)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/mcp/rpc", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`))
	req.Header.Set("Authorization", "Bearer remote-mcp-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "mcp_rate_limited") {
		t.Fatalf("first rate-limited MCP request status = %d body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/mcp/rpc", strings.NewReader(`{"jsonrpc":"2.0","id":2,"method":"initialize"}`))
	req.Header.Set("Authorization", "Bearer remote-mcp-token")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "mcp_rate_limited") {
		t.Fatalf("second rate-limited MCP request status = %d body = %s", rec.Code, rec.Body.String())
	}

	cfg = remoteMCPTestConfig(t)
	cfg.MCP.MaxResponseBytes = 320
	router = newTestRouter(cfg)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/mcp/rpc", strings.NewReader(`{"jsonrpc":"2.0","id":3,"method":"resources/list"}`))
	req.Header.Set("Authorization", "Bearer remote-mcp-token")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "mcp_response_too_large") {
		t.Fatalf("capped remote MCP response status = %d body = %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "nivora://capabilities/current") {
		t.Fatalf("capped remote MCP response leaked full catalog: %s", rec.Body.String())
	}
}

func remoteMCPTestConfig(t *testing.T) config.Config {
	t.Helper()
	t.Setenv("NIVORA_TEST_REMOTE_MCP_TOKEN", "remote-mcp-token")
	cfg := config.Default()
	cfg.MCP.Enabled = true
	cfg.MCP.Mode = "http"
	cfg.Auth.Enabled = true
	cfg.Auth.Mode = "token"
	cfg.Auth.StaticTokenEnv = "NIVORA_TEST_REMOTE_MCP_TOKEN"
	if os.Getenv(cfg.Auth.StaticTokenEnv) == "" {
		t.Fatal("test token env was not set")
	}
	return cfg
}
