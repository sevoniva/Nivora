package repository

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

func applyDetection(snapshot *RepositorySnapshot) {
	languages := map[string]struct{}{}
	frameworks := map[string]struct{}{}
	buildTools := map[string]struct{}{}
	packageManagers := map[string]struct{}{}
	deploymentFiles := map[string]struct{}{}
	workflowFiles := map[string]struct{}{}
	securityFiles := map[string]struct{}{}
	warnings := append([]string(nil), snapshot.Warnings...)

	for _, file := range snapshot.Files {
		path := filepath.ToSlash(file.Path)
		name := filepath.Base(path)
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".go":
			languages["Go"] = struct{}{}
		case ".js", ".jsx":
			languages["JavaScript"] = struct{}{}
		case ".ts", ".tsx":
			languages["TypeScript"] = struct{}{}
		case ".py":
			languages["Python"] = struct{}{}
		case ".java":
			languages["Java"] = struct{}{}
		case ".tf":
			frameworks["Terraform"] = struct{}{}
			deploymentFiles[path] = struct{}{}
		}

		switch name {
		case "go.mod":
			languages["Go"] = struct{}{}
			buildTools["go"] = struct{}{}
			packageManagers["go modules"] = struct{}{}
		case "package.json":
			languages["Node.js"] = struct{}{}
			buildTools["npm"] = struct{}{}
			packageManagers["npm"] = struct{}{}
		case "pnpm-lock.yaml":
			packageManagers["pnpm"] = struct{}{}
		case "yarn.lock":
			packageManagers["yarn"] = struct{}{}
		case "package-lock.json":
			packageManagers["npm"] = struct{}{}
		case "vite.config.ts", "vite.config.js", "vite.config.mjs":
			frameworks["Vite"] = struct{}{}
		case "pom.xml":
			languages["Java"] = struct{}{}
			buildTools["maven"] = struct{}{}
			packageManagers["maven"] = struct{}{}
		case "build.gradle", "build.gradle.kts":
			languages["Java"] = struct{}{}
			buildTools["gradle"] = struct{}{}
			packageManagers["gradle"] = struct{}{}
		case "requirements.txt":
			languages["Python"] = struct{}{}
			packageManagers["pip"] = struct{}{}
		case "pyproject.toml":
			languages["Python"] = struct{}{}
			packageManagers["python"] = struct{}{}
		case "Cargo.toml":
			languages["Rust"] = struct{}{}
			packageManagers["cargo"] = struct{}{}
		case "Dockerfile":
			buildTools["docker"] = struct{}{}
			deploymentFiles[path] = struct{}{}
		case "docker-compose.yaml", "docker-compose.yml":
			deploymentFiles[path] = struct{}{}
		case "Chart.yaml":
			if strings.Contains(path, "chart") || hasSibling(*snapshot, path, "values.yaml") || hasSibling(*snapshot, path, "templates") {
				frameworks["Helm"] = struct{}{}
				deploymentFiles[path] = struct{}{}
			}
		case ".gitlab-ci.yml":
			workflowFiles[path] = struct{}{}
			frameworks["GitLab CI"] = struct{}{}
		case "Makefile":
			buildTools["make"] = struct{}{}
		case "trivy.yaml", ".trivyignore", "cosign.pub", "security-policy.yaml":
			securityFiles[path] = struct{}{}
		}

		if strings.HasPrefix(path, ".github/workflows/") && (strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")) {
			workflowFiles[path] = struct{}{}
			frameworks["GitHub Actions workflow import signal"] = struct{}{}
		}
		if strings.HasPrefix(path, ".nivora/workflows/") && (strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")) {
			workflowFiles[path] = struct{}{}
			frameworks["Nivora Workflow"] = struct{}{}
		}
		if looksLikeKubernetesManifest(path) {
			deploymentFiles[path] = struct{}{}
		}
		if strings.Contains(strings.ToLower(path), "argocd") || strings.Contains(strings.ToLower(path), "gitops") {
			frameworks["GitOps"] = struct{}{}
			deploymentFiles[path] = struct{}{}
		}
		if name == ".env" || strings.HasSuffix(name, ".env") {
			warnings = append(warnings, fmt.Sprintf("secret-like environment file %q detected; values were not read", path))
		}
	}

	if hasFile(*snapshot, "package.json") && (hasSuffix(*snapshot, ".tsx") || hasSuffix(*snapshot, ".jsx")) {
		frameworks["React"] = struct{}{}
	}
	if hasFile(*snapshot, "go.mod") && hasPrefix(*snapshot, "cmd/") {
		frameworks["Go CLI/service"] = struct{}{}
	}

	snapshot.DetectedLanguages = sortedKeys(languages)
	snapshot.DetectedFrameworks = sortedKeys(frameworks)
	snapshot.DetectedBuildTools = sortedKeys(buildTools)
	snapshot.DetectedPackageManagers = sortedKeys(packageManagers)
	snapshot.DetectedDeploymentFiles = sortedKeys(deploymentFiles)
	snapshot.DetectedWorkflowFiles = sortedKeys(workflowFiles)
	snapshot.DetectedSecurityFiles = sortedKeys(securityFiles)
	snapshot.Warnings = dedupeSorted(warnings)
}

func buildCandidates(snapshot RepositorySnapshot) []CommandCandidate {
	var out []CommandCandidate
	if hasFile(snapshot, "go.mod") {
		out = append(out, CommandCandidate{Name: "go build", Command: "go build ./...", Source: "detection:go.mod"})
	}
	if hasFile(snapshot, "package.json") {
		out = append(out, CommandCandidate{Name: "node build", Command: "npm run build", Source: "detection:package.json"})
	}
	if hasFile(snapshot, "pom.xml") {
		out = append(out, CommandCandidate{Name: "maven package", Command: "mvn package", Source: "detection:pom.xml"})
	}
	if hasFile(snapshot, "build.gradle") || hasFile(snapshot, "build.gradle.kts") {
		out = append(out, CommandCandidate{Name: "gradle build", Command: "./gradlew build", Source: "detection:build.gradle"})
	}
	if hasFile(snapshot, "Makefile") {
		out = append(out, CommandCandidate{Name: "make build", Command: "make build", Source: "detection:Makefile"})
	}
	return out
}

func testCandidates(snapshot RepositorySnapshot) []CommandCandidate {
	var out []CommandCandidate
	if hasFile(snapshot, "go.mod") {
		out = append(out, CommandCandidate{Name: "go test", Command: "go test ./...", Source: "detection:go.mod"})
	}
	if hasFile(snapshot, "package.json") {
		out = append(out, CommandCandidate{Name: "node test", Command: "npm test", Source: "detection:package.json"})
	}
	if hasFile(snapshot, "pom.xml") {
		out = append(out, CommandCandidate{Name: "maven test", Command: "mvn test", Source: "detection:pom.xml"})
	}
	if hasFile(snapshot, "build.gradle") || hasFile(snapshot, "build.gradle.kts") {
		out = append(out, CommandCandidate{Name: "gradle test", Command: "./gradlew test", Source: "detection:build.gradle"})
	}
	if hasFile(snapshot, "Makefile") {
		out = append(out, CommandCandidate{Name: "make test", Command: "make test", Source: "detection:Makefile"})
	}
	return out
}

func packageCandidates(snapshot RepositorySnapshot) []CommandCandidate {
	var out []CommandCandidate
	if hasFile(snapshot, "Dockerfile") {
		out = append(out, CommandCandidate{Name: "docker build", Command: "docker build -t <image> .", Source: "detection:Dockerfile"})
	}
	if hasFile(snapshot, "package.json") {
		out = append(out, CommandCandidate{Name: "node package", Command: "npm pack", Source: "detection:package.json"})
	}
	if hasFile(snapshot, "go.mod") {
		out = append(out, CommandCandidate{Name: "go binary", Command: "go build -o dist/app ./...", Source: "detection:go.mod"})
	}
	return out
}

func deploymentCandidates(snapshot RepositorySnapshot) []string {
	values := map[string]struct{}{}
	for _, path := range snapshot.DetectedDeploymentFiles {
		switch {
		case strings.Contains(path, "chart") || strings.HasSuffix(path, "Chart.yaml"):
			values["helm-chart"] = struct{}{}
		case strings.Contains(strings.ToLower(path), "argocd"), strings.Contains(strings.ToLower(path), "gitops"):
			values["gitops"] = struct{}{}
		case strings.Contains(strings.ToLower(path), "docker-compose"):
			values["docker-compose"] = struct{}{}
		default:
			values["kubernetes-yaml"] = struct{}{}
		}
	}
	if hasFile(snapshot, "Dockerfile") {
		values["container-image"] = struct{}{}
	}
	return sortedKeys(values)
}

func securityCandidates(snapshot RepositorySnapshot) []string {
	values := map[string]struct{}{"secret-scan": {}, "dependency-scan": {}}
	if len(snapshot.DetectedDeploymentFiles) > 0 {
		values["manifest-misconfiguration-scan"] = struct{}{}
	}
	if hasFile(snapshot, "Dockerfile") {
		values["container-image-scan"] = struct{}{}
	}
	if hasFile(snapshot, "go.mod") || hasFile(snapshot, "package.json") || hasFile(snapshot, "pom.xml") {
		values["sbom-foundation"] = struct{}{}
	}
	return sortedKeys(values)
}

func workflowDraft(snapshot RepositorySnapshot) string {
	lines := []string{
		"apiVersion: nivora.io/v1alpha1",
		"kind: Workflow",
		"metadata:",
		"  name: detected-ci",
		"on:",
		"  - manual",
		"jobs:",
	}
	testCommands := testCandidates(snapshot)
	buildCommands := buildCandidates(snapshot)
	if len(testCommands) > 0 {
		lines = append(lines, "  test:", "    runsOn: [self-hosted, shell]", "    steps:")
		for _, candidate := range testCommands {
			lines = append(lines, "      - name: "+quoteYAML(candidate.Name), "        run: "+quoteYAML(candidate.Command))
		}
	}
	if len(buildCommands) > 0 {
		needs := ""
		if len(testCommands) > 0 {
			needs = "    needs: [test]"
		}
		lines = append(lines, "  build:")
		if needs != "" {
			lines = append(lines, needs)
		}
		lines = append(lines, "    runsOn: [self-hosted, shell]", "    steps:")
		for _, candidate := range buildCommands {
			lines = append(lines, "      - name: "+quoteYAML(candidate.Name), "        run: "+quoteYAML(candidate.Command))
		}
	}
	if len(testCommands) == 0 && len(buildCommands) == 0 {
		lines = append(lines, "  inspect:", "    runsOn: [self-hosted, shell]", "    steps:", "      - name: inspect", "        run: \"echo plan-only repository inspection\"")
	}
	return strings.Join(lines, "\n") + "\n"
}

func hasSibling(snapshot RepositorySnapshot, path string, sibling string) bool {
	dir := filepath.ToSlash(filepath.Dir(path))
	target := filepath.ToSlash(filepath.Join(dir, sibling))
	if sibling == "templates" {
		return hasPrefix(snapshot, target+"/")
	}
	return hasFile(snapshot, target)
}

func looksLikeKubernetesManifest(path string) bool {
	lower := strings.ToLower(path)
	if !(strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml")) {
		return false
	}
	keywords := []string{
		"deployment", "service", "configmap", "secret", "ingress", "statefulset", "daemonset", "job", "cronjob", "namespace",
	}
	for _, keyword := range keywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}

func dedupeSorted(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			seen[value] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for value := range seen {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func quoteYAML(value string) string {
	escaped := strings.ReplaceAll(value, `"`, `\"`)
	return `"` + escaped + `"`
}
