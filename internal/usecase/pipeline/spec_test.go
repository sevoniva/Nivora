package pipeline

import "testing"

func TestParseDefinition(t *testing.T) {
	def, err := ParseDefinition([]byte(`
apiVersion: nivora.io/v1alpha1
kind: Pipeline
metadata:
  name: hello-shell
spec:
  stages:
    - name: build
      jobs:
        - name: echo
          executor: shell
          steps:
            - name: say-hello
              run: echo "hello from nivora"
`))
	if err != nil {
		t.Fatalf("parse definition: %v", err)
	}
	if def.Metadata.Name != "hello-shell" {
		t.Fatalf("name = %q", def.Metadata.Name)
	}
	if def.Spec.Stages[0].Jobs[0].Steps[0].Name != "say-hello" {
		t.Fatalf("step name = %q", def.Spec.Stages[0].Jobs[0].Steps[0].Name)
	}
}

func TestParseDefinitionRejectsUnsupportedExecutor(t *testing.T) {
	_, err := ParseDefinition([]byte(`
apiVersion: nivora.io/v1alpha1
kind: Pipeline
metadata:
  name: bad
spec:
  stages:
    - name: build
      jobs:
        - name: job
          executor: kubernetes_job
          steps:
            - run: echo nope
`))
	if err == nil {
		t.Fatal("expected unsupported executor error")
	}
}
