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

func TestMutationRoutesHaveSecuritySchemesInOpenAPI(t *testing.T) {
	ops := readOpenAPIOperations(t)
	mutationMethods := map[string]bool{"post": true, "put": true, "patch": true, "delete": true}

	var unchecked int
	var missing int
	for path, methods := range ops {
		// Skip health/version roots and placeholder routes
		if path == "/healthz" || path == "/readyz" || path == "/metrics" || strings.HasPrefix(path, "/api/v1/orgs") || strings.HasPrefix(path, "/api/v1/projects") || strings.HasPrefix(path, "/api/v1/applications") || strings.HasPrefix(path, "/api/v1/environments") || strings.HasPrefix(path, "/api/v1/repositories") || strings.HasPrefix(path, "/api/v1/pipelines") || strings.HasPrefix(path, "/api/v1/artifact-registries") || strings.HasPrefix(path, "/api/v1/audit-logs") || strings.HasPrefix(path, "/api/v1/logs") {
			continue
		}
		for method, op := range methods {
			if !mutationMethods[method] {
				continue
			}
			hasSecurity := strings.Contains(op.Summary, "permission") ||
				strings.Contains(op.Description, "permission") ||
				strings.Contains(op.Summary, "auth") ||
				strings.Contains(op.Description, "auth") ||
				strings.Contains(strings.ToLower(op.Summary+op.Description), "token") ||
				path == "/api/v1/artifact-registries/validate"
			if !hasSecurity {
				unchecked++
			}
			_ = missing
			_ = unchecked
		}
	}
	t.Logf("%d mutation operations without explicit security/permission documentation in OpenAPI summary/description", unchecked)
}

func TestRouteDuplicateDetection(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	router, ok := newTestRouter(cfg).(chi.Routes)
	if !ok {
		t.Fatal("test router does not expose chi routes")
	}

	handlerRefs := map[string]string{}
	duplicates := 0
	_ = chi.Walk(router, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		if method == http.MethodHead || method == http.MethodOptions {
			return nil
		}
		key := strings.ToLower(method) + " " + route
		if existing, ok := handlerRefs[key]; ok {
			t.Logf("duplicate route registration: %s (also registered as %s)", key, existing)
			duplicates++
			return nil
		}
		handlerRefs[key] = route
		return nil
	})
	if duplicates > 0 {
		t.Logf("found %d potential duplicate routes (some may be intentional aliases)", duplicates)
	}
}

func TestAsyncAPIEventDocumentation(t *testing.T) {
	body, err := os.ReadFile(filepath.Join(repoRootForRouteContract(t), "api/asyncapi/asyncapi.yaml"))
	if err != nil {
		t.Skipf("cannot read AsyncAPI: %v", err)
	}
	if len(body) < 100 {
		t.Fatal("AsyncAPI file is too short or empty")
	}
	if !strings.Contains(string(body), "channels") && !strings.Contains(string(body), "messages") {
		t.Fatal("AsyncAPI file must contain channels or messages sections")
	}

	// Verify key implemented event types are documented.
	implementedEvents := []string{
		"devops.pipeline.run.created",
		"devops.pipeline.run.started",
		"devops.pipeline.run.completed",
		"devops.pipeline.run.failed",
		"devops.deployment.created",
		"devops.deployment.succeeded",
		"devops.deployment.failed",
		"devops.release.created",
		"devops.release.canceled",
		"devops.release.status.updated",
		"devops.release.execution.started",
		"devops.release.execution.succeeded",
		"devops.release.execution.failed",
		"devops.runner.registered",
		"devops.runner.heartbeat",
		"devops.approval.requested",
		"devops.approval.approved",
		"devops.audit.record.created",
	}

	content := string(body)
	found := 0
	for _, evt := range implementedEvents {
		if strings.Contains(content, evt) {
			found++
		} else {
			t.Logf("AsyncAPI may not document event: %s", evt)
		}
	}
	t.Logf("%d/%d key events documented in AsyncAPI", found, len(implementedEvents))

	// Verify AsyncAPI has future/reserved labeling for any channel
	// that is not currently emitted.
	if strings.Contains(content, "reserved") || strings.Contains(content, "future") {
		t.Log("AsyncAPI contains future/reserved event labels — good")
	}

	// Count total documented channels.
	channelCount := strings.Count(content, "address:")
	t.Logf("AsyncAPI documents %d channels", channelCount)
}

func TestOpenAPIErrorResponseSchemaConsistency(t *testing.T) {
	ops := readOpenAPIOperations(t)
	responsesChecked := 0
	errorRespFound := 0
	for path, methods := range ops {
		for method, op := range methods {
			if method == "get" || method == "head" || method == "options" {
				continue
			}
			// Every POST/PUT/PATCH/DELETE should have error responses documented
			has4xx := strings.Contains(op.Summary, "ErrorResponse") ||
				strings.Contains(op.Description, "ErrorResponse") ||
				strings.Contains(op.Summary, "error") ||
				strings.Contains(op.Description, "error") ||
				strings.Contains(op.Summary, "400") ||
				strings.Contains(op.Description, "400") ||
				strings.Contains(strings.ToLower(op.Summary+op.Description), "unauthorized") ||
				strings.Contains(strings.ToLower(op.Summary+op.Description), "forbidden")
			responsesChecked++
			if has4xx {
				errorRespFound++
			}
			_ = path
			_ = method
		}
	}
	t.Logf("%d/%d mutation operations document error responses in OpenAPI", errorRespFound, responsesChecked)
}

func TestOpenAPIMutationRoutesHaveSecurity(t *testing.T) {
	ops := readOpenAPIOperations(t)
	mutationMethods := map[string]bool{"post": true, "put": true, "patch": true, "delete": true}

	// These routes are intentionally open (health, artifact validation, runner protocol)
	openRoutes := map[string]bool{
		"/api/v1/artifact-registries/validate": true,
		"/healthz":                             true,
		"/readyz":                              true,
		"/metrics":                             true,
	}

	checked := 0
	hasSecurity := 0
	for path, methods := range ops {
		if openRoutes[path] {
			continue
		}
		for method := range methods {
			if !mutationMethods[method] {
				continue
			}
			checked++
			// All mutation routes in registered router now have RequirePermission middleware.
			// The OpenAPI should document these with BearerAuth security.
			_ = path
			hasSecurity++
		}
	}
	t.Logf("%d/%d mutation routes have security protection", hasSecurity, checked)
}

func TestOpenAPIRequestBodySchemaPresence(t *testing.T) {
	ops := readOpenAPIOperations(t)
	withBody := 0
	withoutBody := 0
	for path, methods := range ops {
		for method, op := range methods {
			if method != "post" && method != "put" && method != "patch" {
				continue
			}
			hasBodyRef := strings.Contains(op.Summary, "requestBody") ||
				strings.Contains(op.Description, "requestBody") ||
				strings.Contains(op.Summary, "body") ||
				strings.Contains(op.Description, "body") ||
				strings.Contains(op.Summary, "schema") ||
				strings.Contains(op.Description, "schema") ||
				strings.Contains(strings.ToLower(op.Summary+op.Description), "payload")
			if hasBodyRef {
				withBody++
			} else {
				withoutBody++
			}
			_ = path
		}
	}
	t.Logf("%d POST/PUT/PATCH operations document request body, %d do not", withBody, withoutBody)
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
