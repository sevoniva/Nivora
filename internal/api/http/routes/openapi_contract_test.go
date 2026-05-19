package routes

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/sevoniva/nivora/internal/infra/config"
	"gopkg.in/yaml.v3"
)

func TestOpenAPIPathsMatchRegisteredRoutes(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router, ok := newTestRouter(cfg).(chi.Routes)
	if !ok {
		t.Fatal("test router does not expose chi routes")
	}

	registered := map[string]map[string]bool{}
	if err := chi.Walk(router, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		if method == http.MethodHead || method == http.MethodOptions {
			return nil
		}
		route = strings.TrimSuffix(route, "/")
		if route == "" {
			route = "/"
		}
		if registered[route] == nil {
			registered[route] = map[string]bool{}
		}
		registered[route][strings.ToLower(method)] = true
		return nil
	}); err != nil {
		t.Fatalf("walk routes: %v", err)
	}

	openapi := readOpenAPIPaths(t)
	for route, methods := range registered {
		openapiMethods, ok := openapi[route]
		if !ok {
			t.Errorf("registered route %s is missing from OpenAPI", route)
			continue
		}
		for method := range methods {
			if !openapiMethods[method] {
				t.Errorf("registered route %s %s is missing from OpenAPI", strings.ToUpper(method), route)
			}
		}
	}
	for path, methods := range openapi {
		for method := range methods {
			if registered[path] == nil || !registered[path][method] {
				t.Errorf("OpenAPI path %s %s is missing from registered routes", strings.ToUpper(method), path)
			}
		}
	}
}

func TestOpenAPIPlaceholderRouteLabelsMatchRouter(t *testing.T) {
	openapi := readOpenAPIOperations(t)
	placeholders := map[string]bool{}
	for _, group := range placeholderGroups() {
		placeholders["/api/v1"+group.path] = true
	}

	for path, ops := range openapi {
		for method, op := range ops {
			isPlaceholderDoc := strings.Contains(strings.ToLower(op.Summary+" "+op.Description), "placeholder") ||
				strings.Contains(strings.ToLower(op.Summary+" "+op.Description), "not implemented")
			if placeholders[path] && !isPlaceholderDoc {
				t.Fatalf("placeholder route %s %s is not documented as placeholder/not implemented", strings.ToUpper(method), path)
			}
			if !placeholders[path] && isPlaceholderDoc {
				t.Fatalf("implemented route %s %s is documented as placeholder/not implemented", strings.ToUpper(method), path)
			}
		}
	}
}

func TestAllPlaceholderRoutesReturnStructuredNotImplemented(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := newTestRouter(cfg)
	for _, group := range placeholderGroups() {
		path := "/api/v1" + group.path
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotImplemented {
			t.Fatalf("%s status = %d body = %s", path, rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), `"code":"not_implemented"`) || !strings.Contains(rec.Body.String(), `"path":"`+path+`"`) {
			t.Fatalf("%s response is not structured not_implemented: %s", path, rec.Body.String())
		}
	}
}

type openAPIDocument struct {
	Paths map[string]map[string]openAPIOperation `yaml:"paths"`
}

type openAPIOperation struct {
	Summary     string `yaml:"summary"`
	Description string `yaml:"description"`
}

func readOpenAPIPaths(t *testing.T) map[string]map[string]bool {
	t.Helper()
	ops := readOpenAPIOperations(t)
	out := make(map[string]map[string]bool, len(ops))
	for path, methods := range ops {
		out[path] = map[string]bool{}
		for method := range methods {
			out[path][method] = true
		}
	}
	return out
}

func readOpenAPIOperations(t *testing.T) map[string]map[string]openAPIOperation {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(repoRootForRouteContract(t), "api/openapi/openapi.yaml"))
	if err != nil {
		t.Fatalf("read OpenAPI: %v", err)
	}
	var doc openAPIDocument
	if err := yaml.Unmarshal(body, &doc); err != nil {
		t.Fatalf("parse OpenAPI: %v", err)
	}
	if len(doc.Paths) == 0 {
		t.Fatal("OpenAPI has no paths")
	}
	return doc.Paths
}

func repoRootForRouteContract(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file path")
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("cannot find repository root")
		}
		dir = parent
	}
}
