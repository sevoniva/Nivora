package deployment

import (
	"context"
	"testing"
)

func doc(content string) ManifestDocument {
	return ManifestDocument{Content: content}
}

func TestK8sSafetyDenyPrivilegedPod(t *testing.T) {
	p := DefaultK8sSafetyPolicy()
	docs := []ManifestDocument{doc(`
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
spec:
  containers:
  - name: app
    image: nginx:1.25
    securityContext:
      privileged: true
`)}
	result := p.ValidateManifests(context.Background(), docs, "default")
	if result.Allowed {
		t.Fatal("expected privileged pod to be denied")
	}
}

func TestK8sSafetyAllowNonPrivilegedPod(t *testing.T) {
	p := DefaultK8sSafetyPolicy()
	docs := []ManifestDocument{doc(`
apiVersion: v1
kind: Pod
metadata:
  name: safe-pod
spec:
  containers:
  - name: app
    image: nginx:1.25
`)}
	result := p.ValidateManifests(context.Background(), docs, "default")
	if !result.Allowed {
		t.Fatalf("expected safe pod to be allowed, got checks: %v", result.Checks)
	}
}

func TestK8sSafetyDenyHostPath(t *testing.T) {
	p := DefaultK8sSafetyPolicy()
	docs := []ManifestDocument{doc(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hostpath-dep
spec:
  template:
    spec:
      containers:
      - name: app
        image: nginx:1.25
      volumes:
      - name: data
        hostPath:
          path: /etc/hostdata
`)}
	result := p.ValidateManifests(context.Background(), docs, "default")
	if result.Allowed {
		t.Fatal("expected hostPath volume to be denied")
	}
}

func TestK8sSafetyDenyHostNetwork(t *testing.T) {
	p := DefaultK8sSafetyPolicy()
	docs := []ManifestDocument{doc(`
apiVersion: v1
kind: Pod
metadata:
  name: hostnet-pod
spec:
  hostNetwork: true
  containers:
  - name: app
    image: nginx:1.25
`)}
	result := p.ValidateManifests(context.Background(), docs, "default")
	if result.Allowed {
		t.Fatal("expected hostNetwork pod to be denied")
	}
}

func TestK8sSafetyDenyLatestImageTag(t *testing.T) {
	p := DefaultK8sSafetyPolicy()
	docs := []ManifestDocument{doc(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: latest-dep
spec:
  template:
    spec:
      containers:
      - name: app
        image: nginx:latest
`)}
	result := p.ValidateManifests(context.Background(), docs, "default")
	if result.Allowed {
		t.Fatal("expected latest tag to be denied")
	}
}

func TestK8sSafetyDigestWarning(t *testing.T) {
	p := DefaultK8sSafetyPolicy()
	p.RequireDigest = true
	docs := []ManifestDocument{doc(`
apiVersion: v1
kind: Pod
metadata:
  name: no-digest-pod
spec:
  containers:
  - name: app
    image: nginx:1.25
`)}
	result := p.ValidateManifests(context.Background(), docs, "default")
	if len(result.Warnings) == 0 {
		t.Fatal("expected digest warning for image without digest")
	}
}

func TestK8sSafetyRequireNamespace(t *testing.T) {
	p := DefaultK8sSafetyPolicy()
	docs := []ManifestDocument{doc(`
apiVersion: v1
kind: Pod
metadata:
  name: no-ns-pod
spec:
  containers:
  - name: app
    image: nginx:1.25
`)}
	result := p.ValidateManifests(context.Background(), docs, "")
	if result.Allowed {
		t.Fatal("expected missing namespace to be denied")
	}
}

func TestK8sSafetyDenyKubeSystem(t *testing.T) {
	p := DefaultK8sSafetyPolicy()
	docs := []ManifestDocument{doc(`
apiVersion: v1
kind: Pod
metadata:
  name: kube-pod
spec:
  containers:
  - name: app
    image: nginx:1.25
`)}
	result := p.ValidateManifests(context.Background(), docs, "kube-system")
	if result.Allowed {
		t.Fatal("expected kube-system namespace to be denied")
	}
}

func TestK8sSafetyAllowedNamespaces(t *testing.T) {
	p := DefaultK8sSafetyPolicy()
	p.AllowedNamespaces = []string{"staging", "prod-apps"}
	docs := []ManifestDocument{doc(`
apiVersion: v1
kind: Service
metadata:
  name: safe-service
spec:
  selector:
    app: demo
  ports:
  - port: 80
`)}
	if result := p.ValidateManifests(context.Background(), docs, "staging"); !result.Allowed {
		t.Fatalf("expected staging namespace to be allowed, checks: %#v", result.Checks)
	}
	if result := p.ValidateManifests(context.Background(), docs, "default"); result.Allowed {
		t.Fatalf("expected default namespace to be denied by allowedNamespaces, checks: %#v", result.Checks)
	}
}

func TestK8sSafetyRejectsManifestNamespaceMismatch(t *testing.T) {
	p := DefaultK8sSafetyPolicy()
	p.AllowedNamespaces = []string{"staging"}
	docs := []ManifestDocument{doc(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: wrong-namespace
  namespace: prod
data:
  key: value
`)}
	result := p.ValidateManifests(context.Background(), docs, "staging")
	if result.Allowed {
		t.Fatalf("expected manifest namespace mismatch to be denied, checks: %#v", result.Checks)
	}
}

func TestK8sSafetyRejectsManifestDeniedNamespace(t *testing.T) {
	p := DefaultK8sSafetyPolicy()
	p.RequireNamespace = false
	docs := []ManifestDocument{doc(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: kube-system-config
  namespace: kube-system
data:
  key: value
`)}
	result := p.ValidateManifests(context.Background(), docs, "")
	if result.Allowed {
		t.Fatalf("expected manifest denied namespace to be rejected, checks: %#v", result.Checks)
	}
}

func TestK8sSafetyDenyClusterScoped(t *testing.T) {
	p := DefaultK8sSafetyPolicy()
	docs := []ManifestDocument{doc(`
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: admin-role
rules:
- apiGroups: ["*"]
  resources: ["*"]
  verbs: ["*"]
`)}
	result := p.ValidateManifests(context.Background(), docs, "default")
	if result.Allowed {
		t.Fatal("expected cluster-scoped resource to be denied")
	}
}

func TestK8sSafetyEmptyDocs(t *testing.T) {
	p := DefaultK8sSafetyPolicy()
	result := p.ValidateManifests(context.Background(), nil, "default")
	if !result.Allowed {
		t.Fatal("expected empty docs to be allowed")
	}
}

func TestK8sSafetyMaxSizeExceeded(t *testing.T) {
	p := DefaultK8sSafetyPolicy()
	p.MaxManifestBytes = 100
	large := make([]byte, 200)
	for i := range large {
		large[i] = 'x'
	}
	docs := []ManifestDocument{doc(string(large))}
	result := p.ValidateManifests(context.Background(), docs, "default")
	if result.Allowed {
		t.Fatal("expected max size check to fail")
	}
}

func TestK8sSafetyDenyHostPID(t *testing.T) {
	p := DefaultK8sSafetyPolicy()
	docs := []ManifestDocument{doc(`
apiVersion: v1
kind: Pod
metadata:
  name: hostpid-pod
spec:
  hostPID: true
  containers:
  - name: app
    image: nginx:1.25
`)}
	result := p.ValidateManifests(context.Background(), docs, "default")
	if result.Allowed {
		t.Fatal("expected hostPID pod to be denied")
	}
}

func TestK8sSafetyDenyHostIPC(t *testing.T) {
	p := DefaultK8sSafetyPolicy()
	docs := []ManifestDocument{doc(`
apiVersion: v1
kind: Pod
metadata:
  name: hostipc-pod
spec:
  hostIPC: true
  containers:
  - name: app
    image: nginx:1.25
`)}
	result := p.ValidateManifests(context.Background(), docs, "default")
	if result.Allowed {
		t.Fatal("expected hostIPC pod to be denied")
	}
}

func TestK8sSafetyDigestImagePasses(t *testing.T) {
	p := DefaultK8sSafetyPolicy()
	docs := []ManifestDocument{doc(`
apiVersion: v1
kind: Pod
metadata:
  name: digest-pod
spec:
  containers:
  - name: app
    image: nginx@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
`)}
	result := p.ValidateManifests(context.Background(), docs, "default")
	if !result.Allowed {
		t.Fatal("expected digest-tagged image to be allowed")
	}
}

func TestDefaultPolicyIsSafe(t *testing.T) {
	p := DefaultK8sSafetyPolicy()
	if !p.DenyPrivileged {
		t.Fatal("DenyPrivileged must be true")
	}
	if !p.DenyHostPath {
		t.Fatal("DenyHostPath must be true")
	}
	if !p.DenyHostNetwork {
		t.Fatal("DenyHostNetwork must be true")
	}
	if !p.RequireNamespace {
		t.Fatal("RequireNamespace must be true")
	}
	if !p.DenyClusterScoped {
		t.Fatal("DenyClusterScoped must be true")
	}
}
