package cloud

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/sevoniva/nivora/internal/domain/audit"
	domaincloud "github.com/sevoniva/nivora/internal/domain/cloud"
	"github.com/sevoniva/nivora/internal/domain/event"
	"github.com/sevoniva/nivora/internal/ports/cloud"
	"github.com/sevoniva/nivora/internal/ports/eventbus"
)

type Service struct {
	store     Store
	providers map[string]cloud.CloudProvider
	eventBus  eventbus.EventBus
	now       func() time.Time
}

func NewService(store Store, providers map[string]cloud.CloudProvider, bus eventbus.EventBus) *Service {
	return &Service{store: store, providers: providers, eventBus: bus, now: time.Now}
}

func (s *Service) Providers(ctx context.Context) ([]domaincloud.CloudProviderInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	names := []string{domaincloud.ProviderAWS, domaincloud.ProviderAliyun, domaincloud.ProviderTencent, domaincloud.ProviderGeneric}
	seen := map[string]bool{}
	items := make([]domaincloud.CloudProviderInfo, 0, len(names))
	for _, name := range names {
		provider, ok := s.providers[name]
		if !ok {
			continue
		}
		info, err := provider.Info(ctx)
		if err != nil {
			return nil, err
		}
		items = append(items, info)
		seen[name] = true
	}
	for name, provider := range s.providers {
		if seen[name] {
			continue
		}
		info, err := provider.Info(ctx)
		if err != nil {
			return nil, err
		}
		items = append(items, info)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items, nil
}

func (s *Service) CreateAccount(ctx context.Context, input CreateAccountInput) (domaincloud.CloudAccount, error) {
	if input.Name == "" {
		return domaincloud.CloudAccount{}, errors.New("cloud account name is required")
	}
	if input.Provider == "" {
		return domaincloud.CloudAccount{}, errors.New("cloud provider is required")
	}
	provider, ok := s.providers[input.Provider]
	if !ok {
		provider = s.providers[domaincloud.ProviderGeneric]
	}
	if provider == nil {
		return domaincloud.CloudAccount{}, fmt.Errorf("cloud provider %q is not configured", input.Provider)
	}
	if err := provider.ValidateConfig(ctx, input.Config); err != nil {
		return domaincloud.CloudAccount{}, redactCloudError(err)
	}
	now := s.now()
	account := domaincloud.CloudAccount{
		ID:            newID("cloudacct"),
		Name:          input.Name,
		Provider:      input.Provider,
		Config:        input.Config,
		CredentialRef: input.CredentialRef,
		Metadata:      input.Metadata,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if account.Config.Provider == "" {
		account.Config.Provider = input.Provider
	}
	if account.Config.CredentialRef == "" {
		account.Config.CredentialRef = input.CredentialRef
	}
	if err := s.store.SaveAccount(ctx, account); err != nil {
		return domaincloud.CloudAccount{}, err
	}
	_ = s.record(ctx, EventCloudAccountCreated, "Cloud account created", account.ID, map[string]any{"provider": account.Provider})
	return account, nil
}

func (s *Service) ListAccounts(ctx context.Context) ([]domaincloud.CloudAccount, error) {
	return s.store.ListAccounts(ctx)
}

func (s *Service) GetAccount(ctx context.Context, id string) (domaincloud.CloudAccount, error) {
	return s.store.GetAccount(ctx, id)
}

func (s *Service) ValidateAccount(ctx context.Context, id string) (ValidationResult, error) {
	account, provider, err := s.accountProvider(ctx, id)
	if err != nil {
		return ValidationResult{}, err
	}
	info, infoErr := provider.Info(ctx)
	if err := provider.ValidateCredential(ctx, account); err != nil {
		return ValidationResult{AccountID: account.ID, Provider: account.Provider, Valid: false, Message: redactCloudError(err).Error()}, redactCloudError(err)
	}
	_ = s.record(ctx, EventCloudCredentialValidated, "Cloud credential validated", account.ID, map[string]any{"provider": account.Provider})
	result := ValidationResult{AccountID: account.ID, Provider: account.Provider, Valid: true, Message: "credential validation succeeded without exposing secret values"}
	if infoErr == nil {
		result.Warnings = append(result.Warnings, info.Warnings...)
	}
	return result, nil
}

func (s *Service) Regions(ctx context.Context, id string) ([]domaincloud.CloudRegion, error) {
	account, provider, err := s.accountProvider(ctx, id)
	if err != nil {
		return nil, err
	}
	return provider.ListRegions(ctx, account)
}

func (s *Service) Clusters(ctx context.Context, id string, region string) ([]domaincloud.CloudCluster, error) {
	account, provider, err := s.accountProvider(ctx, id)
	if err != nil {
		return nil, err
	}
	return provider.ListClusters(ctx, account, region)
}

func (s *Service) Hosts(ctx context.Context, id string, region string) ([]domaincloud.CloudHost, error) {
	account, provider, err := s.accountProvider(ctx, id)
	if err != nil {
		return nil, err
	}
	return provider.ListHosts(ctx, account, region)
}

func (s *Service) Registries(ctx context.Context, id string, region string) ([]domaincloud.CloudRegistry, error) {
	account, provider, err := s.accountProvider(ctx, id)
	if err != nil {
		return nil, err
	}
	return provider.ListRegistries(ctx, account, region)
}

func (s *Service) Inventory(ctx context.Context, id string) (domaincloud.CloudInventorySnapshot, error) {
	account, provider, err := s.accountProvider(ctx, id)
	if err != nil {
		return domaincloud.CloudInventorySnapshot{}, err
	}
	snapshot, err := provider.GetInventorySnapshot(ctx, account)
	if err != nil {
		_ = s.record(ctx, EventCloudInventoryFailed, "Cloud inventory failed", account.ID, map[string]any{"provider": account.Provider})
		return domaincloud.CloudInventorySnapshot{}, err
	}
	if err := s.store.SaveSnapshot(ctx, snapshot); err != nil {
		return domaincloud.CloudInventorySnapshot{}, err
	}
	_ = s.record(ctx, EventCloudInventoryScanned, "Cloud inventory scanned", account.ID, map[string]any{"provider": account.Provider, "regions": len(snapshot.Regions), "clusters": len(snapshot.Clusters), "hosts": len(snapshot.Hosts), "registries": len(snapshot.Registries)})
	return snapshot, nil
}

func (s *Service) accountProvider(ctx context.Context, id string) (domaincloud.CloudAccount, cloud.CloudProvider, error) {
	account, err := s.store.GetAccount(ctx, id)
	if err != nil {
		return domaincloud.CloudAccount{}, nil, err
	}
	provider, ok := s.providers[account.Provider]
	if !ok {
		provider, ok = s.providers[domaincloud.ProviderGeneric]
	}
	if !ok {
		return domaincloud.CloudAccount{}, nil, fmt.Errorf("cloud provider %q is not configured", account.Provider)
	}
	return account, provider, nil
}

func (s *Service) record(ctx context.Context, eventType string, action string, subject string, data map[string]any) error {
	evt := event.Event{ID: newID("evt"), SpecVersion: "1.0", Type: eventType, Source: "nivora.cloud", Subject: subject, Time: s.now(), DataContentType: "application/json", Data: data}
	if err := s.store.AppendEvent(ctx, evt); err != nil {
		return err
	}
	if err := s.store.AppendAudit(ctx, audit.AuditLog{ID: newID("audit"), Action: action, Subject: subject, CreatedAt: s.now()}); err != nil {
		return err
	}
	if s.eventBus != nil {
		_ = s.eventBus.Publish(ctx, evt)
	}
	return nil
}

func redactCloudError(err error) error {
	if err == nil {
		return nil
	}
	message := err.Error()
	for _, marker := range []string{"token", "password", "secret", "access_key", "accesskey", "authorization", "credential"} {
		if strings.Contains(strings.ToLower(message), marker) {
			message = strings.ReplaceAll(message, marker, "[REDACTED]")
			message = strings.ReplaceAll(message, strings.ToUpper(marker), "[REDACTED]")
		}
	}
	return errors.New(message)
}

func newID(prefix string) string {
	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(b[:])
}
