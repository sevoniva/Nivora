package quality_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	domainplugin "github.com/sevoniva/nivora/internal/domain/plugin"
	artifactusecase "github.com/sevoniva/nivora/internal/usecase/artifact"
	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
	pluginusecase "github.com/sevoniva/nivora/internal/usecase/plugin"
	releaseusecase "github.com/sevoniva/nivora/internal/usecase/releaseorchestration"
	"gopkg.in/yaml.v3"
)

func TestExampleYAMLFilesParse(t *testing.T) {
	files := exampleYAMLFiles(t)
	for _, path := range files {
		path := path
		t.Run(path, func(t *testing.T) {
			body, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read example: %v", err)
			}
			decoder := yaml.NewDecoder(bytes.NewReader(body))
			for {
				var node yaml.Node
				if err := decoder.Decode(&node); err != nil {
					if err == io.EOF {
						break
					}
					t.Fatalf("parse YAML: %v", err)
				}
			}
		})
	}
}

func TestSupportedExamplesValidateWithRuntimeParsers(t *testing.T) {
	for _, path := range globExamples(t, "examples/pipelines/*.yaml") {
		if strings.HasSuffix(path, "kubernetes-job.yaml") {
			continue
		}
		if _, err := pipelineusecase.LoadDefinitionFile(path); err != nil {
			t.Fatalf("%s should be a valid local Pipeline example: %v", path, err)
		}
	}

	for _, path := range globExamples(t, "examples/deployments/*.yaml") {
		if intentionallyInvalidDeploymentExample(path) {
			continue
		}
		if _, err := deploymentusecase.LoadDefinitionFile(path); err != nil {
			t.Fatalf("%s should be a valid Deployment example: %v", path, err)
		}
	}

	for _, path := range globExamples(t, "examples/releases/*.yaml") {
		kind := readKind(t, path)
		switch kind {
		case "Release":
			if _, err := artifactusecase.LoadReleaseDefinitionFile(path); err != nil {
				t.Fatalf("%s should be a valid Release example: %v", path, err)
			}
		case "ReleaseOrchestration":
			if _, err := releaseusecase.LoadDefinitionFile(path); err != nil {
				t.Fatalf("%s should be a valid ReleaseOrchestration example: %v", path, err)
			}
		default:
			t.Fatalf("%s has unsupported kind %q", path, kind)
		}
	}
}

func TestPluginTemplatesValidate(t *testing.T) {
	for _, path := range globExamples(t, "examples/plugins/templates/*.yaml") {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		var manifest domainplugin.Manifest
		if err := yaml.Unmarshal(body, &manifest); err != nil {
			t.Fatalf("parse plugin template %s: %v", path, err)
		}
		result := pluginusecase.ValidateManifest(manifest)
		if !result.Valid {
			t.Fatalf("plugin template %s is invalid: %#v", path, result)
		}
	}
}

func TestExamplesDoNotContainHighRiskSecretLiterals(t *testing.T) {
	for _, path := range exampleYAMLFiles(t) {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read example: %v", err)
		}
		content := strings.ToLower(string(body))
		blocked := []string{
			"admin123",
			"harbor12345",
			"begin private key",
			"aws_secret_access_key",
			"password: admin",
			"token: ghp_",
			"authorization: bearer ",
		}
		for _, marker := range blocked {
			if strings.Contains(content, marker) {
				t.Fatalf("%s contains blocked secret-like literal %q", path, marker)
			}
		}
	}
}

func TestDeploymentExamplesReferenceExistingManifestFiles(t *testing.T) {
	for _, path := range globExamples(t, "examples/deployments/*.yaml") {
		if intentionallyInvalidDeploymentExample(path) {
			continue
		}
		def, err := deploymentusecase.LoadDefinitionFile(path)
		if err != nil {
			t.Fatalf("load deployment %s: %v", path, err)
		}
		for _, manifest := range def.Spec.Manifests {
			if _, err := os.Stat(filepath.Join(repoRoot(t), manifest)); err != nil {
				t.Fatalf("%s references missing manifest %s: %v", path, manifest, err)
			}
		}
	}
}

func TestIntentionallyInvalidDeploymentExamplesStayUnsafeToExecute(t *testing.T) {
	path := filepath.Join(repoRoot(t), "examples/deployments/yaml-invalid.yaml")
	def, err := deploymentusecase.LoadDefinitionFile(path)
	if err != nil {
		t.Fatalf("invalid deployment example should still parse as a spec: %v", err)
	}
	for _, manifest := range def.Spec.Manifests {
		if _, err := os.Stat(filepath.Join(repoRoot(t), manifest)); err != nil {
			return
		}
	}
	t.Fatalf("%s should continue to reference a missing manifest for negative-path examples", path)
}

func intentionallyInvalidDeploymentExample(path string) bool {
	return strings.HasSuffix(path, "yaml-invalid.yaml") ||
		strings.HasSuffix(path, "argocd-release.yaml")
}

func exampleYAMLFiles(t *testing.T) []string {
	t.Helper()
	patterns := []string{
		"examples/pipelines/*.yaml",
		"examples/deployments/*.yaml",
		"examples/releases/*.yaml",
		"examples/security/*.yaml",
		"examples/cloud/*.yaml",
		"examples/credentials/*.yaml",
		"examples/approvals/*.yaml",
		"examples/change-windows/*.yaml",
		"examples/notifications/*.yaml",
		"examples/release-targets/*.yaml",
		"examples/artifacts/*.yaml",
		"examples/artifact-registries/*.yaml",
		"examples/hosts/*.yaml",
		"examples/yaml/*.yaml",
		"examples/argocd/*.yaml",
		"examples/gitops/apps/*/*/*.yaml",
		"examples/plugins/templates/*.yaml",
	}
	var files []string
	for _, pattern := range patterns {
		files = append(files, globExamples(t, pattern)...)
	}
	if len(files) == 0 {
		t.Fatal("no example YAML files found")
	}
	return files
}

func globExamples(t *testing.T, pattern string) []string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(repoRoot(t), pattern))
	if err != nil {
		t.Fatalf("glob %s: %v", pattern, err)
	}
	return matches
}

func readKind(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var header struct {
		Kind string `yaml:"kind"`
	}
	if err := yaml.Unmarshal(body, &header); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return header.Kind
}

func repoRoot(t *testing.T) string {
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
