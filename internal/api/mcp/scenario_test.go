package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	domainauth "github.com/sevoniva/nivora/internal/domain/auth"
	"gopkg.in/yaml.v3"
)

type mcpScenario struct {
	ID               string      `yaml:"id"`
	Title            string      `yaml:"title"`
	OperatorQuestion string      `yaml:"operator_question"`
	FixtureState     []string    `yaml:"fixture_state"`
	MCP              scenarioMCP `yaml:"mcp"`
	SafeAnswer       safeAnswer  `yaml:"safe_answer"`
	NextSafeChecks   []string    `yaml:"next_safe_checks"`
	BlockedActions   []string    `yaml:"blocked_actions"`
}

type scenarioMCP struct {
	Resources []string           `yaml:"resources"`
	Tools     []scenarioToolCall `yaml:"tools"`
	Prompts   []string           `yaml:"prompts"`
}

type scenarioToolCall struct {
	Name               string         `yaml:"name"`
	Arguments          map[string]any `yaml:"arguments"`
	RequiresFixture    bool           `yaml:"requires_fixture"`
	ExpectMutatedFalse bool           `yaml:"expect_mutated_false"`
}

type safeAnswer struct {
	Facts           []string `yaml:"facts"`
	Inferences      []string `yaml:"inferences"`
	Unknowns        []string `yaml:"unknowns"`
	ForbiddenClaims []string `yaml:"forbidden_claims"`
}

func TestMCPGoldenScenariosCoverSafeControlPlaneWorkflows(t *testing.T) {
	ctx := context.Background()
	scenarios := loadMCPScenarios(t)
	if len(scenarios) < 8 {
		t.Fatalf("expected at least 8 MCP golden scenarios, got %d", len(scenarios))
	}

	catalogServer := newTestMCPServer(t, domainauth.RoleOwner, "token")
	resources, err := catalogServer.ListResources(ctx)
	if err != nil {
		t.Fatalf("ListResources: %v", err)
	}
	tools, err := catalogServer.ListTools(ctx)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	prompts, err := catalogServer.ListPrompts(ctx)
	if err != nil {
		t.Fatalf("ListPrompts: %v", err)
	}
	resourceNames := resourceSet(resources)
	toolNames := toolSet(tools)
	promptNames := promptSet(prompts)

	for _, scenario := range scenarios {
		t.Run(scenario.ID, func(t *testing.T) {
			assertScenarioHasReviewContent(t, scenario)
			for _, resource := range scenario.MCP.Resources {
				if !resourceNames[canonicalScenarioResource(resource)] {
					t.Fatalf("scenario resource %s is not exposed by MCP catalog", resource)
				}
			}
			for _, tool := range scenario.MCP.Tools {
				if !toolNames[tool.Name] {
					t.Fatalf("scenario tool %s is not exposed by MCP catalog", tool.Name)
				}
				if !tool.RequiresFixture {
					result := callScenarioTool(t, ctx, tool)
					if tool.ExpectMutatedFalse {
						assertToolMutatedFalse(t, result)
					}
				}
			}
			for _, prompt := range scenario.MCP.Prompts {
				if !promptNames[prompt] {
					t.Fatalf("scenario prompt %s is not exposed by MCP catalog", prompt)
				}
				result, err := catalogServer.GetPrompt(ctx, prompt, map[string]string{
					"id":              scenario.ID + "-id",
					"subject":         scenario.ID,
					"subjectType":     "deployment",
					"subjectId":       scenario.ID,
					"requestedAction": "apply",
				})
				if err != nil {
					t.Fatalf("GetPrompt(%s): %v", prompt, err)
				}
				text := result.Messages[0].Content.Text
				for _, want := range []string{"Separate facts from inference", "List unknowns", "Never request", "read-only and plan-only"} {
					if !strings.Contains(text, want) {
						t.Fatalf("prompt %s missing safety phrase %q: %s", prompt, want, text)
					}
				}
			}
			admin := newTestMCPServer(t, domainauth.RoleAdmin, "token")
			for _, blocked := range scenario.BlockedActions {
				result, err := admin.CallTool(ctx, blocked, map[string]any{"authorization": "Bearer placeholder"})
				if err != nil {
					t.Fatalf("blocked tool %s returned transport error: %v", blocked, err)
				}
				if !result.IsError || !strings.Contains(result.Content[0].Text, "mcp_action_not_allowed") {
					t.Fatalf("blocked tool %s result = %#v", blocked, result)
				}
				if strings.Contains(result.Content[0].Text, "Bearer placeholder") {
					t.Fatalf("blocked tool %s leaked sensitive argument: %s", blocked, result.Content[0].Text)
				}
			}
		})
	}
}

func TestMCPPlanOnlyToolsReturnMutatedFalse(t *testing.T) {
	ctx := context.Background()
	cases := []scenarioToolCall{
		{Name: "nivora_plan_deployment_local", Arguments: map[string]any{"content": deploymentDefinitionYAML()}, ExpectMutatedFalse: true},
		{Name: "nivora_evaluate_policy_local", Arguments: map[string]any{
			"subjectType": "manifest",
			"subjectId":   "mcp-policy-test",
			"content":     "securityContext:\n  privileged: true\n",
		}, ExpectMutatedFalse: true},
		{Name: "nivora_inspect_artifact", Arguments: map[string]any{"reference": "registry.example.invalid/team/app:latest", "type": "image"}, ExpectMutatedFalse: true},
		{Name: "nivora_inspect_artifact_reference", Arguments: map[string]any{"reference": "registry.example.invalid/team/app@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "type": "image"}, ExpectMutatedFalse: true},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			result := callScenarioTool(t, ctx, tc)
			assertToolMutatedFalse(t, result)
		})
	}
}

func TestMCPScenarioLogPreviewIsTruncatedAndRedacted(t *testing.T) {
	longSensitiveLog := strings.Repeat("safe line\n", 5000) + "Authorization: Bearer placeholder\n"
	body := mustJSON(map[string]any{"logs": truncateLogs([]map[string]any{{"stream": "stderr", "content": longSensitiveLog}})})
	if !strings.Contains(body, `"truncated": true`) {
		t.Fatalf("expected truncated log preview, got %s", body)
	}
	if strings.Contains(body, "Bearer placeholder") || strings.Contains(body, "Authorization: Bearer") {
		t.Fatalf("truncated log preview leaked sensitive content: %s", body)
	}
}

func loadMCPScenarios(t *testing.T) []mcpScenario {
	t.Helper()
	files, err := filepath.Glob("../../../examples/mcp/scenarios/*.yaml")
	if err != nil {
		t.Fatalf("glob scenarios: %v", err)
	}
	var scenarios []mcpScenario
	for _, file := range files {
		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read scenario %s: %v", file, err)
		}
		var scenario mcpScenario
		if err := yaml.Unmarshal(body, &scenario); err != nil {
			t.Fatalf("decode scenario %s: %v", file, err)
		}
		if scenario.ID == "" {
			t.Fatalf("scenario %s missing id", file)
		}
		scenarios = append(scenarios, scenario)
	}
	return scenarios
}

func callScenarioTool(t *testing.T, ctx context.Context, call scenarioToolCall) ToolResult {
	t.Helper()
	role := domainauth.RoleDeveloper
	if call.Name == "nivora_search_audit" {
		role = domainauth.RoleAuditor
	}
	server := newTestMCPServer(t, role, "token")
	result, err := server.CallTool(ctx, call.Name, call.Arguments)
	if err != nil {
		t.Fatalf("%s transport error: %v", call.Name, err)
	}
	if result.IsError {
		t.Fatalf("%s returned tool error: %#v", call.Name, result)
	}
	return result
}

func assertToolMutatedFalse(t *testing.T, result ToolResult) {
	t.Helper()
	if len(result.Content) == 0 {
		t.Fatalf("empty tool result")
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatalf("decode tool payload: %v\n%s", err, result.Content[0].Text)
	}
	mutated, ok := payload["mutated"].(bool)
	if !ok || mutated {
		t.Fatalf("expected mutated=false in %s", result.Content[0].Text)
	}
}

func assertScenarioHasReviewContent(t *testing.T, scenario mcpScenario) {
	t.Helper()
	if scenario.Title == "" || scenario.OperatorQuestion == "" {
		t.Fatalf("scenario %s missing title or operator question", scenario.ID)
	}
	if len(scenario.FixtureState) == 0 || len(scenario.NextSafeChecks) == 0 {
		t.Fatalf("scenario %s missing fixture state or next safe checks", scenario.ID)
	}
	if len(scenario.MCP.Resources)+len(scenario.MCP.Tools)+len(scenario.MCP.Prompts) == 0 {
		t.Fatalf("scenario %s has no MCP evidence sources", scenario.ID)
	}
	if len(scenario.SafeAnswer.Facts) == 0 || len(scenario.SafeAnswer.Inferences) == 0 ||
		len(scenario.SafeAnswer.Unknowns) == 0 || len(scenario.SafeAnswer.ForbiddenClaims) == 0 {
		t.Fatalf("scenario %s missing safe answer sections", scenario.ID)
	}
}

func canonicalScenarioResource(resource string) string {
	replacer := strings.NewReplacer(
		"pipe-demo-001", "{id}",
		"dep-demo-001", "{id}",
		"rexec-demo-001", "{id}",
	)
	return replacer.Replace(resource)
}

func resourceSet(resources []Resource) map[string]bool {
	set := map[string]bool{}
	for _, resource := range resources {
		set[resource.URI] = true
	}
	return set
}

func toolSet(tools []Tool) map[string]bool {
	set := map[string]bool{}
	for _, tool := range tools {
		set[tool.Name] = true
	}
	return set
}

func promptSet(prompts []Prompt) map[string]bool {
	set := map[string]bool{}
	for _, prompt := range prompts {
		set[prompt.Name] = true
	}
	return set
}
