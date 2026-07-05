package deployment

import (
	"context"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// K8sSafetyPolicy defines safety rules for Kubernetes manifest validation.
type K8sSafetyPolicy struct {
	RequireNamespace  bool     `json:"requireNamespace" yaml:"requireNamespace"`
	RequireDigest     bool     `json:"requireDigest" yaml:"requireDigest"`
	DenyLatestTag     bool     `json:"denyLatestTag" yaml:"denyLatestTag"`
	DenyPrivileged    bool     `json:"denyPrivileged" yaml:"denyPrivileged"`
	DenyHostPath      bool     `json:"denyHostPath" yaml:"denyHostPath"`
	DenyHostNetwork   bool     `json:"denyHostNetwork" yaml:"denyHostNetwork"`
	DenyHostPID       bool     `json:"denyHostPID" yaml:"denyHostPID"`
	DenyHostIPC       bool     `json:"denyHostIPC" yaml:"denyHostIPC"`
	AllowedNamespaces []string `json:"allowedNamespaces,omitempty" yaml:"allowedNamespaces,omitempty"`
	DeniedNamespaces  []string `json:"deniedNamespaces,omitempty" yaml:"deniedNamespaces,omitempty"`
	AllowedKinds      []string `json:"allowedKinds,omitempty" yaml:"allowedKinds,omitempty"`
	DeniedKinds       []string `json:"deniedKinds,omitempty" yaml:"deniedKinds,omitempty"`
	MaxManifestBytes  int64    `json:"maxManifestBytes" yaml:"maxManifestBytes"`
	MaxResourceCount  int      `json:"maxResourceCount" yaml:"maxResourceCount"`
	DenyClusterScoped bool     `json:"denyClusterScoped" yaml:"denyClusterScoped"`
}

// DefaultK8sSafetyPolicy returns a safe-by-default policy.
func DefaultK8sSafetyPolicy() K8sSafetyPolicy {
	return K8sSafetyPolicy{
		RequireNamespace:  true,
		DenyPrivileged:    true,
		DenyHostPath:      true,
		DenyHostNetwork:   true,
		DenyHostPID:       true,
		DenyHostIPC:       true,
		DenyLatestTag:     true,
		DeniedNamespaces:  []string{"kube-system", "kube-public", "kube-node-lease"},
		DeniedKinds:       []string{},
		DenyClusterScoped: true,
		MaxManifestBytes:  1 << 20, // 1MB
		MaxResourceCount:  100,
	}
}

func KubernetesSafetyPolicyFromSpec(spec Spec) K8sSafetyPolicy {
	return DefaultK8sSafetyPolicy().WithOptions(spec.KubernetesSafety)
}

// WithOptions applies Deployment spec safety options as stricter overlays.
// It never disables the built-in deny rules and never raises default size/count
// limits, so user-provided options can only narrow the accepted manifest set.
func (p K8sSafetyPolicy) WithOptions(options KubernetesSafetyOptions) K8sSafetyPolicy {
	if options.RequireDigest {
		p.RequireDigest = true
	}
	if len(options.AllowedNamespaces) > 0 {
		p.AllowedNamespaces = compactK8sNames(options.AllowedNamespaces)
	}
	if len(options.DeniedNamespaces) > 0 {
		p.DeniedNamespaces = appendUniqueK8sNames(p.DeniedNamespaces, options.DeniedNamespaces...)
	}
	if len(options.AllowedKinds) > 0 {
		p.AllowedKinds = compactK8sNames(options.AllowedKinds)
	}
	if len(options.DeniedKinds) > 0 {
		p.DeniedKinds = appendUniqueK8sNames(p.DeniedKinds, options.DeniedKinds...)
	}
	if options.MaxManifestBytes > 0 && (p.MaxManifestBytes == 0 || options.MaxManifestBytes < p.MaxManifestBytes) {
		p.MaxManifestBytes = options.MaxManifestBytes
	}
	if options.MaxResourceCount > 0 && (p.MaxResourceCount == 0 || options.MaxResourceCount < p.MaxResourceCount) {
		p.MaxResourceCount = options.MaxResourceCount
	}
	return p
}

// K8sSafetyCheck holds a single policy check result.
type K8sSafetyCheck struct {
	Passed  bool   `json:"passed"`
	Rule    string `json:"rule"`
	Kind    string `json:"kind,omitempty"`
	Name    string `json:"name,omitempty"`
	Message string `json:"message"`
}

// K8sSafetyResult is the aggregate result of all safety checks.
type K8sSafetyResult struct {
	Allowed  bool             `json:"allowed"`
	Checks   []K8sSafetyCheck `json:"checks"`
	Warnings []string         `json:"warnings,omitempty"`
}

// ValidateManifests runs the safety policy against rendered manifests.
func (p K8sSafetyPolicy) ValidateManifests(ctx context.Context, documents []ManifestDocument, namespace string) K8sSafetyResult {
	if err := ctx.Err(); err != nil {
		return K8sSafetyResult{Allowed: false, Checks: []K8sSafetyCheck{{Passed: false, Rule: "context", Message: err.Error()}}}
	}
	namespace = strings.TrimSpace(namespace)

	result := K8sSafetyResult{Allowed: true}

	// Namespace validation.
	if p.RequireNamespace && namespace == "" {
		result.Allowed = false
		result.Checks = append(result.Checks, K8sSafetyCheck{
			Passed: false, Rule: "require-namespace", Message: "namespace is required but not specified",
		})
	} else {
		result.Checks = append(result.Checks, K8sSafetyCheck{
			Passed: true, Rule: "require-namespace", Message: "namespace specified: " + namespace,
		})
	}

	if namespace != "" && containsK8sName(p.DeniedNamespaces, namespace) {
		result.Allowed = false
		result.Checks = append(result.Checks, K8sSafetyCheck{
			Passed: false, Rule: "denied-namespace", Message: fmt.Sprintf("namespace %s is denied", namespace),
		})
	}
	if namespace != "" && len(p.AllowedNamespaces) > 0 && !containsK8sName(p.AllowedNamespaces, namespace) {
		result.Allowed = false
		result.Checks = append(result.Checks, K8sSafetyCheck{
			Passed: false, Rule: "allowed-namespace", Message: fmt.Sprintf("namespace %s is not in the allowed namespace list", namespace),
		})
	}

	// Manifest size check.
	totalBytes := 0
	for _, doc := range documents {
		totalBytes += len(doc.Content)
	}
	if p.MaxManifestBytes > 0 && int64(totalBytes) > p.MaxManifestBytes {
		result.Allowed = false
		result.Checks = append(result.Checks, K8sSafetyCheck{
			Passed: false, Rule: "max-manifest-size",
			Message: fmt.Sprintf("manifest size %d exceeds max %d", totalBytes, p.MaxManifestBytes),
		})
	} else if p.MaxManifestBytes > 0 {
		result.Checks = append(result.Checks, K8sSafetyCheck{
			Passed: true, Rule: "max-manifest-size",
			Message: fmt.Sprintf("manifest size %d within limit", totalBytes),
		})
	}
	if p.MaxResourceCount > 0 && len(documents) > p.MaxResourceCount {
		result.Allowed = false
		result.Checks = append(result.Checks, K8sSafetyCheck{
			Passed: false, Rule: "max-resource-count",
			Message: fmt.Sprintf("resource count %d exceeds max %d", len(documents), p.MaxResourceCount),
		})
	} else if p.MaxResourceCount > 0 {
		result.Checks = append(result.Checks, K8sSafetyCheck{
			Passed: true, Rule: "max-resource-count",
			Message: fmt.Sprintf("resource count %d within limit", len(documents)),
		})
	}

	// Per-document checks.
	for _, doc := range documents {
		var obj map[string]interface{}
		if err := yaml.Unmarshal([]byte(doc.Content), &obj); err != nil {
			continue
		}
		kind, _ := obj["kind"].(string)
		metadata := nestedMap(obj, "metadata")
		name, _ := metadata["name"].(string)
		documentNamespace, _ := metadata["namespace"].(string)
		documentNamespace = strings.TrimSpace(documentNamespace)
		if documentNamespace != "" {
			if namespace != "" && documentNamespace != namespace {
				result.Allowed = false
				result.Checks = append(result.Checks, K8sSafetyCheck{
					Passed: false, Rule: "namespace-mismatch", Kind: kind, Name: name,
					Message: fmt.Sprintf("%s/%s declares namespace %s but target namespace is %s", kind, name, documentNamespace, namespace),
				})
			}
			if containsK8sName(p.DeniedNamespaces, documentNamespace) {
				result.Allowed = false
				result.Checks = append(result.Checks, K8sSafetyCheck{
					Passed: false, Rule: "denied-namespace", Kind: kind, Name: name,
					Message: fmt.Sprintf("%s/%s declares denied namespace %s", kind, name, documentNamespace),
				})
			}
			if len(p.AllowedNamespaces) > 0 && !containsK8sName(p.AllowedNamespaces, documentNamespace) {
				result.Allowed = false
				result.Checks = append(result.Checks, K8sSafetyCheck{
					Passed: false, Rule: "allowed-namespace", Kind: kind, Name: name,
					Message: fmt.Sprintf("%s/%s declares namespace %s outside the allowed namespace list", kind, name, documentNamespace),
				})
			}
		}

		// Cluster-scoped check.
		if p.DenyClusterScoped {
			clusterScopedKinds := map[string]bool{
				"ClusterRole": true, "ClusterRoleBinding": true,
				"Namespace": true, "PersistentVolume": true,
				"StorageClass": true, "CustomResourceDefinition": true,
				"PriorityClass": true, "Node": true,
			}
			if clusterScopedKinds[kind] {
				result.Allowed = false
				result.Checks = append(result.Checks, K8sSafetyCheck{
					Passed: false, Rule: "deny-cluster-scoped", Kind: kind, Name: name,
					Message: fmt.Sprintf("%s/%s is cluster-scoped and denied by policy", kind, name),
				})
			}
		}

		// Denied kinds.
		if len(p.AllowedKinds) > 0 && !containsK8sName(p.AllowedKinds, kind) {
			result.Allowed = false
			result.Checks = append(result.Checks, K8sSafetyCheck{
				Passed: false, Rule: "allowed-kind", Kind: kind, Name: name,
				Message: fmt.Sprintf("%s is not in the allowed kind list", kind),
			})
		}
		for _, denied := range p.DeniedKinds {
			if strings.EqualFold(kind, denied) {
				result.Allowed = false
				result.Checks = append(result.Checks, K8sSafetyCheck{
					Passed: false, Rule: "denied-kind", Kind: kind, Name: name,
					Message: fmt.Sprintf("%s is a denied kind", kind),
				})
			}
		}

		// Pod-level checks.
		spec := nestedMap(obj, "spec")
		if kind == "Pod" || kind == "Deployment" || kind == "StatefulSet" || kind == "DaemonSet" || kind == "Job" || kind == "CronJob" || kind == "ReplicaSet" {
			if kind != "Pod" {
				spec = nestedMap(nestedMap(obj, "spec"), "template", "spec")
			}

			// Privileged containers.
			if p.DenyPrivileged {
				containers := k8sSlice(spec, "containers")
				for _, c := range containers {
					container, _ := c.(map[string]interface{})
					sec, _ := container["securityContext"].(map[string]interface{})
					if priv, _ := sec["privileged"].(bool); priv {
						result.Allowed = false
						cName, _ := container["name"].(string)
						result.Checks = append(result.Checks, K8sSafetyCheck{
							Passed: false, Rule: "deny-privileged", Kind: kind, Name: name,
							Message: fmt.Sprintf("container %s in %s/%s is privileged", cName, kind, name),
						})
					}
				}
			}

			// Host path volumes.
			if p.DenyHostPath {
				volumes := k8sSlice(spec, "volumes")
				for _, v := range volumes {
					vol, _ := v.(map[string]interface{})
					if _, hasHostPath := vol["hostPath"]; hasHostPath {
						result.Allowed = false
						vName, _ := vol["name"].(string)
						result.Checks = append(result.Checks, K8sSafetyCheck{
							Passed: false, Rule: "deny-hostpath", Kind: kind, Name: name,
							Message: fmt.Sprintf("volume %s in %s/%s uses hostPath", vName, kind, name),
						})
					}
				}
			}

			// Host network/PID/IPC.
			if p.DenyHostNetwork && k8sBool(spec, "hostNetwork") {
				result.Allowed = false
				result.Checks = append(result.Checks, K8sSafetyCheck{
					Passed: false, Rule: "deny-host-network", Kind: kind, Name: name,
					Message: fmt.Sprintf("%s/%s uses hostNetwork", kind, name),
				})
			}
			if p.DenyHostPID && k8sBool(spec, "hostPID") {
				result.Allowed = false
				result.Checks = append(result.Checks, K8sSafetyCheck{
					Passed: false, Rule: "deny-host-pid", Kind: kind, Name: name,
					Message: fmt.Sprintf("%s/%s uses hostPID", kind, name),
				})
			}
			if p.DenyHostIPC && k8sBool(spec, "hostIPC") {
				result.Allowed = false
				result.Checks = append(result.Checks, K8sSafetyCheck{
					Passed: false, Rule: "deny-host-ipc", Kind: kind, Name: name,
					Message: fmt.Sprintf("%s/%s uses hostIPC", kind, name),
				})
			}

			// Image tag checks.
			if p.DenyLatestTag || p.RequireDigest {
				containers := k8sSlice(spec, "containers")
				for _, c := range containers {
					container, _ := c.(map[string]interface{})
					image, _ := container["image"].(string)
					if image == "" {
						continue
					}
					cName, _ := container["name"].(string)
					if p.DenyLatestTag && (strings.HasSuffix(image, ":latest") || !strings.Contains(image, ":")) {
						result.Allowed = false
						result.Checks = append(result.Checks, K8sSafetyCheck{
							Passed: false, Rule: "deny-latest-tag", Kind: kind, Name: name,
							Message: fmt.Sprintf("container %s uses latest tag: %s", cName, image),
						})
					}
					if p.RequireDigest && !strings.Contains(image, "@sha256:") {
						result.Warnings = append(result.Warnings,
							fmt.Sprintf("%s/%s/%s: image %s does not use digest", kind, name, cName, image))
					}
				}
			}
		}
	}

	return result
}

func k8sSlice(obj map[string]interface{}, key string) []interface{} {
	if v, ok := obj[key]; ok {
		if s, ok := v.([]interface{}); ok {
			return s
		}
	}
	return nil
}

func k8sBool(obj map[string]interface{}, key string) bool {
	if v, ok := obj[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func containsK8sName(values []string, value string) bool {
	value = strings.TrimSpace(value)
	for _, candidate := range values {
		if strings.EqualFold(strings.TrimSpace(candidate), value) {
			return true
		}
	}
	return false
}

func appendUniqueK8sNames(base []string, values ...string) []string {
	out := compactK8sNames(base)
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || containsK8sName(out, value) {
			continue
		}
		out = append(out, value)
	}
	return out
}

func compactK8sNames(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || containsK8sName(out, value) {
			continue
		}
		out = append(out, value)
	}
	return out
}
