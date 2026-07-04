package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/sevoniva/nivora/internal/infra/config"
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
