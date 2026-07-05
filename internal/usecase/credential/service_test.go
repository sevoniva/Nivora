package credential

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	domaincredential "github.com/sevoniva/nivora/internal/domain/credential"
	portsecret "github.com/sevoniva/nivora/internal/ports/secret"
)

func TestSecretLifecycleDoesNotExposeValue(t *testing.T) {
	service := NewService(NewMemoryStore(), newFakeSecretProvider(), nil)
	ref, err := service.PutSecret(context.Background(), SecretCreateInput{
		Name:      "registry token",
		Key:       "examples/registry/token",
		Value:     "sample-value-for-test-only",
		ScopeType: domaincredential.ScopeProject,
		ScopeID:   "project-a",
	})
	if err != nil {
		t.Fatalf("put secret: %v", err)
	}
	if ref.ID == "" || ref.Key == "" {
		t.Fatalf("expected secret ref metadata, got %#v", ref)
	}
	refs, err := service.ListSecretRefs(context.Background(), portsecret.Scope{ScopeType: domaincredential.ScopeProject})
	if err != nil {
		t.Fatalf("list refs: %v", err)
	}
	if len(refs) != 1 {
		t.Fatalf("refs len = %d", len(refs))
	}
}

func TestCredentialValidateRecordsUsageAndAudit(t *testing.T) {
	provider := newFakeSecretProvider()
	service := NewService(NewMemoryStore(), provider, nil)
	ref, err := service.PutSecret(context.Background(), SecretCreateInput{Name: "argocd token", Key: "examples/argocd/token", Value: "sample-value-for-test-only"})
	if err != nil {
		t.Fatalf("put secret: %v", err)
	}
	cred, err := service.CreateCredential(context.Background(), CredentialCreateInput{Name: "argocd", Type: domaincredential.TypeArgoCD, SecretRef: ref})
	if err != nil {
		t.Fatalf("create credential: %v", err)
	}
	result, err := service.ValidateCredential(context.Background(), cred.ID, "tester")
	if err != nil {
		t.Fatalf("validate credential: %v", err)
	}
	if !result.Valid {
		t.Fatalf("expected credential to validate")
	}
	if len(provider.usages) != 1 {
		t.Fatalf("usage len = %d", len(provider.usages))
	}
	audits, err := service.Audits(context.Background())
	if err != nil {
		t.Fatalf("audits: %v", err)
	}
	if len(audits) == 0 {
		t.Fatalf("expected audit records")
	}
	for _, audit := range audits {
		if audit.Subject == "sample-value-for-test-only" {
			t.Fatalf("audit leaked secret value")
		}
	}
}

func TestSecretRotateUpdatesVersionWithoutReturningValue(t *testing.T) {
	service := NewService(NewMemoryStore(), newFakeSecretProvider(), nil)
	ref, err := service.PutSecret(context.Background(), SecretCreateInput{Name: "registry token", Key: "examples/registry/token", Value: "old-value"})
	if err != nil {
		t.Fatalf("put secret: %v", err)
	}
	rotated, err := service.RotateSecret(context.Background(), SecretRotateInput{ID: ref.ID, Value: "new-value", ActorID: "tester"})
	if err != nil {
		t.Fatalf("rotate secret: %v", err)
	}
	if rotated.Version == "" || rotated.Version == ref.Version {
		t.Fatalf("expected rotated version, before=%q after=%q", ref.Version, rotated.Version)
	}
	if rotated.Metadata["value"] == "new-value" {
		t.Fatalf("secret value leaked through metadata")
	}
	audits, err := service.Audits(context.Background())
	if err != nil {
		t.Fatalf("audits: %v", err)
	}
	if len(audits) == 0 {
		t.Fatalf("expected rotation audit")
	}
	for _, audit := range audits {
		if audit.Subject == "new-value" {
			t.Fatalf("audit leaked rotated secret value")
		}
	}
}

func TestValidateSecretProvider(t *testing.T) {
	service := NewService(NewMemoryStore(), newFakeSecretProvider(), nil)
	status, err := service.ValidateSecretProvider(context.Background(), "tester")
	if err != nil {
		t.Fatalf("validate provider: %v", err)
	}
	if !status.Configured || !status.Reachable {
		t.Fatalf("unexpected provider status: %#v", status)
	}
}

func TestValidateSecretProviderRedactsProviderStatus(t *testing.T) {
	provider := newFakeSecretProvider()
	provider.status = portsecret.ProviderStatus{
		Provider:     "vault",
		Configured:   true,
		Reachable:    false,
		Capabilities: []string{"get", "client_secret_debug"},
		Message:      "client_secret=provider-secret-value",
		Metadata: map[string]string{
			"mount":         "secret",
			"token":         "provider-token-value",
			"diagnosticURL": "https://vault.example.invalid?password=provider-password-value",
		},
	}
	service := NewService(NewMemoryStore(), provider, nil)
	status, err := service.ValidateSecretProvider(context.Background(), "tester")
	if err != nil {
		t.Fatalf("validate provider: %v", err)
	}
	body, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("marshal status: %v", err)
	}
	for _, forbidden := range []string{"provider-secret-value", "provider-token-value", "provider-password-value", "client_secret_debug"} {
		if strings.Contains(string(body), forbidden) {
			t.Fatalf("provider status leaked sensitive value %q: %s", forbidden, body)
		}
	}
	if status.Message != "[REDACTED]" || status.Metadata["token"] != "[REDACTED]" || status.Metadata["diagnosticURL"] != "[REDACTED]" || status.Capabilities[1] != "[REDACTED]" {
		t.Fatalf("provider status was not redacted: %#v", status)
	}
	audits, err := service.Audits(context.Background())
	if err != nil {
		t.Fatalf("audits: %v", err)
	}
	auditBody, err := json.Marshal(audits)
	if err != nil {
		t.Fatalf("marshal audits: %v", err)
	}
	for _, forbidden := range []string{"provider-secret-value", "provider-token-value", "provider-password-value"} {
		if strings.Contains(string(auditBody), forbidden) {
			t.Fatalf("provider validation audit leaked sensitive value %q: %s", forbidden, auditBody)
		}
	}
}

func TestListCredentialsFiltersByScope(t *testing.T) {
	service := NewService(NewMemoryStore(), newFakeSecretProvider(), nil)
	refA, err := service.PutSecret(context.Background(), SecretCreateInput{Name: "a", ScopeType: domaincredential.ScopeProject, ScopeID: "project-a", Value: "a-value"})
	if err != nil {
		t.Fatalf("put secret a: %v", err)
	}
	refB, err := service.PutSecret(context.Background(), SecretCreateInput{Name: "b", ScopeType: domaincredential.ScopeProject, ScopeID: "project-b", Value: "b-value"})
	if err != nil {
		t.Fatalf("put secret b: %v", err)
	}
	if _, err := service.CreateCredential(context.Background(), CredentialCreateInput{Name: "a", ScopeType: domaincredential.ScopeProject, ScopeID: "project-a", SecretRef: refA}); err != nil {
		t.Fatalf("create credential a: %v", err)
	}
	if _, err := service.CreateCredential(context.Background(), CredentialCreateInput{Name: "b", ScopeType: domaincredential.ScopeProject, ScopeID: "project-b", SecretRef: refB}); err != nil {
		t.Fatalf("create credential b: %v", err)
	}
	credentials, err := service.ListCredentials(context.Background(), portsecret.Scope{ScopeType: domaincredential.ScopeProject, ScopeID: "project-a"})
	if err != nil {
		t.Fatalf("list credentials: %v", err)
	}
	if len(credentials) != 1 || credentials[0].ScopeID != "project-a" {
		t.Fatalf("expected only project-a credential, got %#v", credentials)
	}
}

func TestSecretUsagePolicyDeniesUnexpectedUse(t *testing.T) {
	provider := newFakeSecretProvider()
	service := NewService(NewMemoryStore(), provider, nil)
	ref, err := service.PutSecret(context.Background(), SecretCreateInput{
		Name:  "limited",
		Value: "secret-value",
		Policy: domaincredential.SecretPolicy{
			AllowedUses: []string{"deployment.apply"},
		},
	})
	if err != nil {
		t.Fatalf("put secret: %v", err)
	}
	cred, err := service.CreateCredential(context.Background(), CredentialCreateInput{Name: "limited", Type: domaincredential.TypeGeneric, SecretRef: ref})
	if err != nil {
		t.Fatalf("create credential: %v", err)
	}
	result, err := service.ValidateCredential(context.Background(), cred.ID, "tester")
	if err != nil {
		t.Fatalf("validate credential: %v", err)
	}
	if result.Valid {
		t.Fatalf("expected policy to deny validation use")
	}
	if len(provider.usages) != 0 {
		t.Fatalf("unexpected usage recorded when policy denied")
	}
}

type fakeSecretProvider struct {
	values map[string][]byte
	refs   map[string]domaincredential.SecretRef
	usages []domaincredential.SecretUsage
	status portsecret.ProviderStatus
}

func newFakeSecretProvider() *fakeSecretProvider {
	return &fakeSecretProvider{values: make(map[string][]byte), refs: make(map[string]domaincredential.SecretRef)}
}

func (f *fakeSecretProvider) PutSecret(ctx context.Context, request portsecret.PutRequest) (domaincredential.SecretRef, error) {
	ref := request.Ref
	if ref.Version == "" {
		ref.Version = "1"
	}
	f.refs[ref.ID] = ref
	f.values[ref.ID] = append([]byte(nil), request.Value...)
	return ref, nil
}

func (f *fakeSecretProvider) GetSecret(ctx context.Context, ref domaincredential.SecretRef) ([]byte, error) {
	return append([]byte(nil), f.values[ref.ID]...), nil
}

func (f *fakeSecretProvider) DeleteSecret(ctx context.Context, ref domaincredential.SecretRef) error {
	delete(f.values, ref.ID)
	delete(f.refs, ref.ID)
	return nil
}

func (f *fakeSecretProvider) RotateSecret(ctx context.Context, ref domaincredential.SecretRef, newValue []byte) (domaincredential.SecretRef, error) {
	f.values[ref.ID] = append([]byte(nil), newValue...)
	ref.Version = "2"
	f.refs[ref.ID] = ref
	return ref, nil
}

func (f *fakeSecretProvider) ListSecretRefs(ctx context.Context, scope portsecret.Scope) ([]domaincredential.SecretRef, error) {
	refs := make([]domaincredential.SecretRef, 0, len(f.refs))
	for _, ref := range f.refs {
		if scope.ScopeType != "" && ref.ScopeType != scope.ScopeType {
			continue
		}
		refs = append(refs, ref)
	}
	return refs, nil
}

func (f *fakeSecretProvider) RecordUsage(ctx context.Context, usage domaincredential.SecretUsage) error {
	f.usages = append(f.usages, usage)
	return nil
}

func (f *fakeSecretProvider) ValidateProvider(ctx context.Context) (portsecret.ProviderStatus, error) {
	if f.status.Provider != "" {
		return f.status, ctx.Err()
	}
	return portsecret.ProviderStatus{Provider: "fake", Configured: true, Reachable: true}, ctx.Err()
}
