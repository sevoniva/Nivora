package mcp

import (
	"context"
	"os"
	"strings"
	"testing"

	domainauth "github.com/sevoniva/nivora/internal/domain/auth"
)

func TestMCPPermissionMatrixCoversCatalogs(t *testing.T) {
	matrix := loadMCPPermissionMatrix(t)
	server := newTestMCPServer(t, domainauth.RoleOwner, "token")

	resources, err := server.ListResources(context.Background())
	if err != nil {
		t.Fatalf("ListResources: %v", err)
	}
	for _, resource := range resources {
		assertMatrixRow(t, matrix, resource.URI, "resource", "no", "no")
	}

	tools, err := server.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	for _, tool := range tools {
		assertMatrixRow(t, matrix, tool.Name, "tool", "no", "no")
	}
	for name := range blockedActionTools {
		row := assertMatrixRow(t, matrix, name, "tool", "no", "yes")
		if row["Status"] != "denied" || row["Audit Event"] != EventToolDenied {
			t.Fatalf("%s matrix row should be denied with %s: %#v", name, EventToolDenied, row)
		}
	}

	prompts, err := server.ListPrompts(context.Background())
	if err != nil {
		t.Fatalf("ListPrompts: %v", err)
	}
	for _, prompt := range prompts {
		assertMatrixRow(t, matrix, prompt.Name, "prompt", "no", "no")
	}
}

func TestMCPPermissionMatrixDocumentsAuditResources(t *testing.T) {
	matrix := loadMCPPermissionMatrix(t)
	auditResource := matrix["nivora://audit/search"]
	if auditResource["Required Permission"] != domainauth.PermissionAuditRead {
		t.Fatalf("audit resource permission = %#v", auditResource)
	}
	auditTool := matrix["nivora_search_audit"]
	if auditTool["Required Permission"] != domainauth.PermissionAuditRead {
		t.Fatalf("audit tool permission = %#v", auditTool)
	}
}

func loadMCPPermissionMatrix(t *testing.T) map[string]map[string]string {
	t.Helper()
	body, err := os.ReadFile("../../../docs/security/MCP_PERMISSION_MATRIX.md")
	if err != nil {
		t.Fatalf("read matrix: %v", err)
	}
	lines := strings.Split(string(body), "\n")
	var headers []string
	rows := map[string]map[string]string{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") || strings.Contains(line, "---") {
			continue
		}
		parts := splitMarkdownRow(line)
		if len(parts) == 0 {
			continue
		}
		if parts[0] == "Name" {
			headers = parts
			continue
		}
		if len(headers) == 0 || len(parts) != len(headers) {
			continue
		}
		row := map[string]string{}
		for i, header := range headers {
			row[header] = parts[i]
		}
		rows[row["Name"]] = row
	}
	if len(rows) == 0 {
		t.Fatalf("empty MCP permission matrix")
	}
	return rows
}

func splitMarkdownRow(line string) []string {
	line = strings.Trim(line, "|")
	raw := strings.Split(line, "|")
	parts := make([]string, 0, len(raw))
	for _, part := range raw {
		parts = append(parts, strings.TrimSpace(part))
	}
	return parts
}

func assertMatrixRow(t *testing.T, matrix map[string]map[string]string, name string, typ string, runnerAllowed string, mutates string) map[string]string {
	t.Helper()
	row, ok := matrix[name]
	if !ok {
		t.Fatalf("%s missing from MCP permission matrix", name)
	}
	if row["Type"] != typ {
		t.Fatalf("%s type = %q, want %q", name, row["Type"], typ)
	}
	if row["Runner Token Allowed"] != runnerAllowed {
		t.Fatalf("%s runner token allowed = %q, want %q", name, row["Runner Token Allowed"], runnerAllowed)
	}
	if row["Mutates State"] != mutates {
		t.Fatalf("%s mutates state = %q, want %q", name, row["Mutates State"], mutates)
	}
	return row
}
