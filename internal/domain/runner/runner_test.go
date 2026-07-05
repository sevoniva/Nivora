package runner

import "testing"

func TestExecutorCapabilityNormalizationAndSupport(t *testing.T) {
	tests := []struct {
		input      string
		normalized string
		supported  bool
	}{
		{input: " shell ", normalized: ExecutorShell, supported: true},
		{input: "CONTAINER", normalized: ExecutorContainer, supported: true},
		{input: "kubernetes_job", normalized: ExecutorKubernetesJob, supported: true},
		{input: "kubernetes-job", normalized: ExecutorKubernetesJob, supported: true},
		{input: "webhook", normalized: ExecutorWebhook, supported: true},
		{input: "noop", normalized: ExecutorNoop, supported: true},
		{input: "external", normalized: ExecutorExternal, supported: true},
		{input: "privileged-shell", normalized: "privileged-shell", supported: false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := NormalizeExecutorCapability(tt.input); got != tt.normalized {
				t.Fatalf("NormalizeExecutorCapability(%q) = %q, want %q", tt.input, got, tt.normalized)
			}
			if got := IsSupportedExecutorCapability(tt.input); got != tt.supported {
				t.Fatalf("IsSupportedExecutorCapability(%q) = %v, want %v", tt.input, got, tt.supported)
			}
		})
	}
}
