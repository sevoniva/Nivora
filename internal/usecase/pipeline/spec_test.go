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
              timeoutSeconds: 3
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
	if def.Spec.Stages[0].Jobs[0].Steps[0].TimeoutSeconds != 3 {
		t.Fatalf("timeoutSeconds = %d", def.Spec.Stages[0].Jobs[0].Steps[0].TimeoutSeconds)
	}
}

func TestParseDefinitionPreservesJobLabels(t *testing.T) {
	def, err := ParseDefinition([]byte(`
apiVersion: nivora.io/v1alpha1
kind: Pipeline
metadata:
  name: labeled
spec:
  stages:
    - name: build
      jobs:
        - name: secure
          executor: shell
          labels:
            runtime: workflow
            tier: secure
          steps:
            - run: echo ok
`))
	if err != nil {
		t.Fatalf("parse definition: %v", err)
	}
	if got := def.Spec.Stages[0].Jobs[0].Labels["tier"]; got != "secure" {
		t.Fatalf("job labels = %#v", def.Spec.Stages[0].Jobs[0].Labels)
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

func TestParseDefinitionValidationFailures(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "missing metadata name",
			body: `
apiVersion: nivora.io/v1alpha1
kind: Pipeline
metadata: {}
spec:
  stages:
    - name: build
      jobs:
        - name: job
          executor: shell
          steps:
            - run: echo nope
`,
		},
		{
			name: "negative retries",
			body: `
apiVersion: nivora.io/v1alpha1
kind: Pipeline
metadata:
  name: bad
spec:
  stages:
    - name: build
      jobs:
        - name: job
          executor: shell
          retries: -1
          steps:
            - run: echo nope
`,
		},
		{
			name: "duplicate job name",
			body: `
apiVersion: nivora.io/v1alpha1
kind: Pipeline
metadata:
  name: bad
spec:
  stages:
    - name: build
      jobs:
        - name: job
          executor: shell
          steps:
            - run: echo one
        - name: job
          executor: shell
          steps:
            - run: echo two
`,
		},
		{
			name: "negative step timeout",
			body: `
apiVersion: nivora.io/v1alpha1
kind: Pipeline
metadata:
  name: bad
spec:
  stages:
    - name: build
      jobs:
        - name: job
          executor: shell
          steps:
            - run: echo nope
              timeoutSeconds: -1
`,
		},
		{
			name: "empty job label",
			body: `
apiVersion: nivora.io/v1alpha1
kind: Pipeline
metadata:
  name: bad
spec:
  stages:
    - name: build
      jobs:
        - name: job
          executor: shell
          labels:
            runtime: ""
          steps:
            - run: echo nope
`,
		},
		{
			name: "secret-like job label",
			body: `
apiVersion: nivora.io/v1alpha1
kind: Pipeline
metadata:
  name: bad
spec:
  stages:
    - name: build
      jobs:
        - name: job
          executor: shell
          labels:
            token: runner-a
          steps:
            - run: echo nope
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParseDefinition([]byte(tt.body)); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
