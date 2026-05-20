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
	}
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

	for _, denied := range p.DeniedNamespaces {
		if namespace == denied {
			result.Allowed = false
			result.Checks = append(result.Checks, K8sSafetyCheck{
				Passed: false, Rule: "denied-namespace", Message: fmt.Sprintf("namespace %s is denied", denied),
			})
		}
	}

	// Manifest size check.
	totalBytes := 0
	for _, doc := range documents {
		totalBytes += len(doc.Content)
	}
	if int64(totalBytes) > p.MaxManifestBytes {
		result.Allowed = false
		result.Checks = append(result.Checks, K8sSafetyCheck{
			Passed: false, Rule: "max-manifest-size",
			Message: fmt.Sprintf("manifest size %d exceeds max %d", totalBytes, p.MaxManifestBytes),
		})
	} else {
		result.Checks = append(result.Checks, K8sSafetyCheck{
			Passed: true, Rule: "max-manifest-size",
			Message: fmt.Sprintf("manifest size %d within limit", totalBytes),
		})
	}

	// Per-document checks.
	for _, doc := range documents {
		var obj map[string]interface{}
		if err := yaml.Unmarshal([]byte(doc.Content), &obj); err != nil {
			continue
		}
		kind, _ := obj["kind"].(string)
		name, _ := obj["metadata"].(map[string]interface{})["name"].(string)

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
