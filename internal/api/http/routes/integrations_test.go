package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sevoniva/nivora/internal/infra/config"
)

func TestListIntegrationsRoute(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := newTestRouter(cfg)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Integrations []struct {
			Name                   string `json:"name"`
			Type                   string `json:"type"`
			Maturity               string `json:"maturity"`
			AdapterKind            string `json:"adapterKind"`
			Boundary               string `json:"boundary"`
			CredentialMode         string `json:"credentialMode"`
			NetworkAccess          string `json:"networkAccess"`
			SafeByDefault          bool   `json:"safeByDefault"`
			DefaultMutation        bool   `json:"defaultMutation"`
			MutatesExternalSystems bool   `json:"mutatesExternalSystems"`
		} `json:"integrations"`
		Count    int      `json:"count"`
		Warnings []string `json:"warnings"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode integrations: %v", err)
	}
	if body.Count == 0 || len(body.Integrations) == 0 {
		t.Fatalf("empty integration index: %#v", body)
	}
	if len(body.Warnings) == 0 {
		t.Fatalf("expected maturity warning")
	}
	if strings.Contains(rec.Body.String(), "not_implemented") {
		t.Fatalf("integration index still looks like placeholder: %s", rec.Body.String())
	}
	var foundArgo bool
	for _, integration := range body.Integrations {
		if integration.AdapterKind == "" || integration.Boundary == "" || integration.CredentialMode == "" || integration.NetworkAccess == "" {
			t.Fatalf("integration missing boundary metadata: %#v", integration)
		}
		if integration.DefaultMutation || integration.MutatesExternalSystems {
			t.Fatalf("integration route exposes default mutation: %#v", integration)
		}
		if integration.Name == "executor-argocd" {
			foundArgo = true
			if integration.Maturity != "experimental" {
				t.Fatalf("argocd maturity = %q", integration.Maturity)
			}
			if integration.Boundary != "guarded-action" || integration.AdapterKind != "noop" || integration.CredentialMode != "credential_ref_only" {
				t.Fatalf("argocd integration boundary = %#v", integration)
			}
		}
	}
	if !foundArgo {
		t.Fatalf("executor-argocd integration missing: %#v", body.Integrations)
	}
}
