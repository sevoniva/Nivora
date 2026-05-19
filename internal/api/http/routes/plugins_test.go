package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sevoniva/nivora/internal/infra/config"
)

func TestPluginRoutes(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := newTestRouter(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plugins", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "executor-shell") {
		t.Fatalf("list body missing shell plugin: %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/plugins/artifact-oci", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get status = %d body = %s", rec.Code, rec.Body.String())
	}
	var plugin struct {
		Name         string `json:"name"`
		Protocol     string `json:"protocol"`
		Capabilities []struct {
			Name string `json:"name"`
		} `json:"capabilities"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &plugin); err != nil {
		t.Fatalf("decode plugin: %v", err)
	}
	if plugin.Name != "artifact-oci" || plugin.Protocol != "builtin" || len(plugin.Capabilities) == 0 {
		t.Fatalf("unexpected plugin = %#v", plugin)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/plugins/artifact-oci/capabilities", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("capabilities status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "artifact.resolve_digest") {
		t.Fatalf("capabilities body = %s", rec.Body.String())
	}
}

func TestMissingPluginRoute(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := newTestRouter(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plugins/missing", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "plugin_not_found") {
		t.Fatalf("body = %s", rec.Body.String())
	}
}
