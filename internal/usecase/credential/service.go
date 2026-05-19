package credential

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sevoniva/nivora/internal/domain/audit"
	domaincredential "github.com/sevoniva/nivora/internal/domain/credential"
	"github.com/sevoniva/nivora/internal/domain/event"
	"github.com/sevoniva/nivora/internal/infra/crypto"
	"github.com/sevoniva/nivora/internal/ports/eventbus"
	portsecret "github.com/sevoniva/nivora/internal/ports/secret"
)

type Service struct {
	store    Store
	secrets  portsecret.Provider
	eventBus eventbus.EventBus
	now      func() time.Time
}

func NewService(store Store, secrets portsecret.Provider, bus eventbus.EventBus) *Service {
	return &Service{store: store, secrets: secrets, eventBus: bus, now: time.Now}
}

func (s *Service) PutSecret(ctx context.Context, input SecretCreateInput) (domaincredential.SecretRef, error) {
	if strings.TrimSpace(input.Name) == "" {
		return domaincredential.SecretRef{}, errors.New("secret name is required")
	}
	if input.Value == "" {
		return domaincredential.SecretRef{}, errors.New("secret value is required")
	}
	if s.secrets == nil {
		return domaincredential.SecretRef{}, errors.New("secret provider is not configured")
	}
	now := s.now()
	ref := domaincredential.SecretRef{
		ID:        newID("secret"),
		Name:      input.Name,
		ScopeType: defaultScope(input.ScopeType),
		ScopeID:   input.ScopeID,
		Provider:  defaultProvider(input.Provider),
		Key:       defaultSecretKey(input.Key, input.Name),
		Policy:    input.Policy,
		Metadata:  crypto.RedactMap(input.Metadata),
		CreatedAt: now,
		UpdatedAt: now,
	}
	stored, err := s.secrets.PutSecret(ctx, portsecret.PutRequest{Ref: ref, Value: []byte(input.Value)})
	if err != nil {
		return domaincredential.SecretRef{}, err
	}
	_ = s.record(ctx, EventSecretCreated, "secret created", input.ActorID, stored.ID, map[string]any{
		"name":      stored.Name,
		"scopeType": stored.ScopeType,
		"provider":  stored.Provider,
	})
	return stored, nil
}

func (s *Service) RotateSecret(ctx context.Context, input SecretRotateInput) (domaincredential.SecretRef, error) {
	if input.ID == "" {
		return domaincredential.SecretRef{}, errors.New("secret id is required")
	}
	if input.Value == "" {
		return domaincredential.SecretRef{}, errors.New("secret value is required")
	}
	if s.secrets == nil {
		return domaincredential.SecretRef{}, errors.New("secret provider is not configured")
	}
	ref, err := s.findSecretRef(ctx, input.ID)
	if err != nil {
		return domaincredential.SecretRef{}, err
	}
	rotated, err := s.secrets.RotateSecret(ctx, ref, []byte(input.Value))
	if err != nil {
		return domaincredential.SecretRef{}, err
	}
	_ = s.record(ctx, EventSecretRotated, "secret rotated", input.ActorID, rotated.ID, map[string]any{
		"name":     rotated.Name,
		"provider": rotated.Provider,
		"version":  rotated.Version,
	})
	return rotated, nil
}

func (s *Service) ValidateSecretProvider(ctx context.Context, actorID string) (portsecret.ProviderStatus, error) {
	if s.secrets == nil {
		return portsecret.ProviderStatus{}, errors.New("secret provider is not configured")
	}
	status, err := s.secrets.ValidateProvider(ctx)
	if err != nil {
		return status, err
	}
	_ = s.record(ctx, EventSecretProviderValidated, "secret provider validated", actorID, status.Provider, map[string]any{"configured": status.Configured, "reachable": status.Reachable})
	return status, nil
}

func (s *Service) ListSecretRefs(ctx context.Context, scope portsecret.Scope) ([]domaincredential.SecretRef, error) {
	if s.secrets == nil {
		return nil, errors.New("secret provider is not configured")
	}
	return s.secrets.ListSecretRefs(ctx, scope)
}

func (s *Service) DeleteSecret(ctx context.Context, id string, actorID string) error {
	if id == "" {
		return errors.New("secret id is required")
	}
	ref, err := s.findSecretRef(ctx, id)
	if err != nil {
		return err
	}
	if err := s.secrets.DeleteSecret(ctx, ref); err != nil {
		return err
	}
	return s.record(ctx, EventSecretDeleted, "secret deleted", actorID, ref.ID, map[string]any{"name": ref.Name})
}

func (s *Service) CreateCredential(ctx context.Context, input CredentialCreateInput) (domaincredential.Credential, error) {
	if strings.TrimSpace(input.Name) == "" {
		return domaincredential.Credential{}, errors.New("credential name is required")
	}
	if strings.TrimSpace(input.Type) == "" {
		input.Type = domaincredential.TypeGeneric
	}
	if input.SecretRef.ID == "" && input.SecretRef.Key == "" {
		return domaincredential.Credential{}, errors.New("credential secretRef id or key is required")
	}
	now := s.now()
	cred := domaincredential.Credential{
		ID:        newID("cred"),
		Name:      input.Name,
		Type:      input.Type,
		ScopeType: defaultScope(input.ScopeType),
		ScopeID:   input.ScopeID,
		SecretRef: sanitizeRef(input.SecretRef),
		Metadata:  crypto.RedactMap(input.Metadata),
		Status:    domaincredential.StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if cred.SecretRef.Provider == "" {
		cred.SecretRef.Provider = "builtin"
	}
	if err := s.store.SaveCredential(ctx, cred); err != nil {
		return domaincredential.Credential{}, err
	}
	_ = s.record(ctx, EventCredentialCreated, "credential created", input.ActorID, cred.ID, map[string]any{
		"name": cred.Name,
		"type": cred.Type,
	})
	return cred, nil
}

func (s *Service) CreateCredentialFromDefinition(ctx context.Context, definition Definition) (domaincredential.Credential, error) {
	return s.CreateCredential(ctx, definition.CreateInput())
}

func (s *Service) ListCredentials(ctx context.Context) ([]domaincredential.Credential, error) {
	return s.store.ListCredentials(ctx)
}

func (s *Service) GetCredential(ctx context.Context, id string) (domaincredential.Credential, error) {
	return s.store.GetCredential(ctx, id)
}

func (s *Service) DeleteCredential(ctx context.Context, id string) error {
	return s.store.DeleteCredential(ctx, id)
}

func (s *Service) ValidateCredential(ctx context.Context, id string, actorID string) (CredentialValidationResult, error) {
	cred, err := s.store.GetCredential(ctx, id)
	if err != nil {
		return CredentialValidationResult{}, err
	}
	if s.secrets == nil {
		return CredentialValidationResult{}, errors.New("secret provider is not configured")
	}
	value, err := s.secrets.GetSecret(ctx, cred.SecretRef)
	if err != nil {
		result := CredentialValidationResult{CredentialID: id, Valid: false, Message: "secret reference could not be resolved", ValidatedAt: s.now()}
		_ = s.record(ctx, EventCredentialValidated, "credential validation failed", actorID, id, map[string]any{"valid": false})
		return result, err
	}
	if len(value) == 0 {
		result := CredentialValidationResult{CredentialID: id, Valid: false, Message: "secret value is empty", ValidatedAt: s.now()}
		_ = s.record(ctx, EventCredentialValidated, "credential validation failed", actorID, id, map[string]any{"valid": false})
		return result, nil
	}
	usage := domaincredential.SecretUsage{
		ID:          newID("usage"),
		SecretRef:   cred.SecretRef,
		UsedBy:      "credential.validate",
		Purpose:     "validate credential secret reference",
		SubjectType: "credential",
		SubjectID:   cred.ID,
		CreatedAt:   s.now(),
	}
	if err := validateUsagePolicy(cred.SecretRef, usage); err != nil {
		result := CredentialValidationResult{CredentialID: id, Valid: false, Message: err.Error(), ValidatedAt: s.now()}
		_ = s.record(ctx, EventCredentialValidated, "credential validation failed", actorID, id, map[string]any{"valid": false})
		return result, nil
	}
	if err := s.secrets.RecordUsage(ctx, usage); err != nil {
		return CredentialValidationResult{}, err
	}
	_ = s.record(ctx, EventSecretUsed, "secret used", actorID, cred.SecretRef.ID, map[string]any{"purpose": usage.Purpose, "subjectType": usage.SubjectType})
	_ = s.record(ctx, EventCredentialValidated, "credential validated", actorID, id, map[string]any{"valid": true})
	return CredentialValidationResult{CredentialID: id, Valid: true, Message: "credential secret reference resolved", ValidatedAt: s.now()}, nil
}

func (s *Service) findSecretRef(ctx context.Context, id string) (domaincredential.SecretRef, error) {
	refs, err := s.ListSecretRefs(ctx, portsecret.Scope{})
	if err != nil {
		return domaincredential.SecretRef{}, err
	}
	for _, ref := range refs {
		if ref.ID == id {
			return ref, nil
		}
	}
	return domaincredential.SecretRef{}, errors.New("secret ref not found")
}

func (s *Service) Events(ctx context.Context) ([]event.Event, error) {
	return s.store.Events(ctx)
}

func (s *Service) Audits(ctx context.Context) ([]audit.AuditLog, error) {
	return s.store.Audits(ctx)
}

func (s *Service) record(ctx context.Context, eventType string, action string, actorID string, subject string, data map[string]any) error {
	evt := event.Event{
		ID:              newID("evt"),
		SpecVersion:     "1.0",
		Type:            eventType,
		Source:          "nivora.credentials",
		Subject:         subject,
		Time:            s.now(),
		DataContentType: "application/json",
		Data:            data,
	}
	if err := s.store.AppendEvent(ctx, evt); err != nil {
		return err
	}
	if err := s.store.AppendAudit(ctx, audit.AuditLog{ID: newID("audit"), ActorID: actorID, Action: action, Subject: subject, CreatedAt: s.now()}); err != nil {
		return err
	}
	if s.eventBus != nil {
		_ = s.eventBus.Publish(ctx, evt)
	}
	return nil
}

func sanitizeRef(ref domaincredential.SecretRef) domaincredential.SecretRef {
	ref.Metadata = crypto.RedactMap(ref.Metadata)
	if ref.ScopeType == "" {
		ref.ScopeType = domaincredential.ScopeGlobal
	}
	if ref.Provider == "" {
		ref.Provider = "builtin"
	}
	return ref
}

func validateUsagePolicy(ref domaincredential.SecretRef, usage domaincredential.SecretUsage) error {
	if len(ref.Policy.AllowedUses) > 0 && !contains(ref.Policy.AllowedUses, usage.UsedBy) && !contains(ref.Policy.AllowedUses, usage.Purpose) {
		return fmt.Errorf("secret policy does not allow use %q", usage.UsedBy)
	}
	if len(ref.Policy.Environments) > 0 && usage.Environment != "" && !contains(ref.Policy.Environments, usage.Environment) {
		return fmt.Errorf("secret policy does not allow environment %q", usage.Environment)
	}
	return nil
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func defaultScope(scope string) string {
	if scope == "" {
		return domaincredential.ScopeGlobal
	}
	return scope
}

func defaultProvider(provider string) string {
	if provider == "" {
		return "builtin"
	}
	return provider
}

func defaultSecretKey(key string, name string) string {
	if key != "" {
		return key
	}
	return "secrets/" + strings.TrimSpace(name)
}

func newID(prefix string) string {
	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(b[:])
}
