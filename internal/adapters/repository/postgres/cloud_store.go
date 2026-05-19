package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sevoniva/nivora/internal/domain/audit"
	domaincloud "github.com/sevoniva/nivora/internal/domain/cloud"
	"github.com/sevoniva/nivora/internal/domain/event"
	cloudusecase "github.com/sevoniva/nivora/internal/usecase/cloud"
)

type CloudStore struct {
	pool *pgxpool.Pool
}

var _ cloudusecase.Store = (*CloudStore)(nil)

func NewCloudStore(pool *pgxpool.Pool) *CloudStore {
	return &CloudStore{pool: pool}
}

// --- Accounts ---

func (s *CloudStore) SaveAccount(ctx context.Context, account domaincloud.CloudAccount) error {
	if account.ID == "" {
		return errors.New("cloud account id is required")
	}
	configJSON, _ := json.Marshal(account.Config)
	metadataJSON, _ := json.Marshal(account.Metadata)
	_, err := s.pool.Exec(ctx, `INSERT INTO cloud_accounts (id, provider, name, credential_ref, config, metadata, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (id) DO UPDATE SET provider=EXCLUDED.provider, name=EXCLUDED.name, credential_ref=EXCLUDED.credential_ref, config=EXCLUDED.config, metadata=EXCLUDED.metadata, updated_at=EXCLUDED.updated_at`,
		account.ID, account.Provider, account.Name, account.CredentialRef, configJSON, metadataJSON, account.CreatedAt, account.UpdatedAt)
	return err
}

func (s *CloudStore) GetAccount(ctx context.Context, id string) (domaincloud.CloudAccount, error) {
	var a domaincloud.CloudAccount
	var configJSON, metadataJSON []byte
	err := s.pool.QueryRow(ctx, `SELECT id, provider, name, credential_ref, config, metadata, created_at, updated_at FROM cloud_accounts WHERE id=$1`, id).
		Scan(&a.ID, &a.Provider, &a.Name, &a.CredentialRef, &configJSON, &metadataJSON, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domaincloud.CloudAccount{}, cloudusecase.ErrAccountNotFound
	}
	if err != nil {
		return domaincloud.CloudAccount{}, err
	}
	json.Unmarshal(configJSON, &a.Config)
	json.Unmarshal(metadataJSON, &a.Metadata)
	return a, nil
}

func (s *CloudStore) ListAccounts(ctx context.Context) ([]domaincloud.CloudAccount, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, provider, name, credential_ref, config, metadata, created_at, updated_at FROM cloud_accounts ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domaincloud.CloudAccount
	for rows.Next() {
		var a domaincloud.CloudAccount
		var configJSON, metadataJSON []byte
		if err := rows.Scan(&a.ID, &a.Provider, &a.Name, &a.CredentialRef, &configJSON, &metadataJSON, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal(configJSON, &a.Config)
		json.Unmarshal(metadataJSON, &a.Metadata)
		out = append(out, a)
	}
	if out == nil {
		out = []domaincloud.CloudAccount{}
	}
	return out, rows.Err()
}

// --- Snapshots ---

func (s *CloudStore) SaveSnapshot(ctx context.Context, snapshot domaincloud.CloudInventorySnapshot) error {
	if snapshot.ID == "" {
		return errors.New("snapshot id is required")
	}
	regionsJSON, _ := json.Marshal(snapshot.Regions)
	clustersJSON, _ := json.Marshal(snapshot.Clusters)
	hostsJSON, _ := json.Marshal(snapshot.Hosts)
	registriesJSON, _ := json.Marshal(snapshot.Registries)
	_, err := s.pool.Exec(ctx, `INSERT INTO cloud_inventory_snapshots (id, account_id, regions, clusters, hosts, registries, scanned_at, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (id) DO UPDATE SET regions=EXCLUDED.regions, clusters=EXCLUDED.clusters, hosts=EXCLUDED.hosts, registries=EXCLUDED.registries, scanned_at=EXCLUDED.scanned_at`,
		snapshot.ID, snapshot.AccountID, regionsJSON, clustersJSON, hostsJSON, registriesJSON, snapshot.ScannedAt, snapshot.ScannedAt)
	return err
}

func (s *CloudStore) GetSnapshot(ctx context.Context, accountID string) (domaincloud.CloudInventorySnapshot, error) {
	var snap domaincloud.CloudInventorySnapshot
	var regionsJSON, clustersJSON, hostsJSON, registriesJSON []byte
	err := s.pool.QueryRow(ctx, `SELECT id, account_id, regions, clusters, hosts, registries, scanned_at, created_at FROM cloud_inventory_snapshots WHERE account_id=$1 ORDER BY scanned_at DESC LIMIT 1`, accountID).
		Scan(&snap.ID, &snap.AccountID, &regionsJSON, &clustersJSON, &hostsJSON, &registriesJSON, &snap.ScannedAt, &snap.ScannedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domaincloud.CloudInventorySnapshot{}, nil
	}
	if err != nil {
		return domaincloud.CloudInventorySnapshot{}, err
	}
	json.Unmarshal(regionsJSON, &snap.Regions)
	json.Unmarshal(clustersJSON, &snap.Clusters)
	json.Unmarshal(hostsJSON, &snap.Hosts)
	json.Unmarshal(registriesJSON, &snap.Registries)
	return snap, nil
}

// --- Events & Audit ---

func (s *CloudStore) AppendEvent(ctx context.Context, evt event.Event) error {
	payload, _ := json.Marshal(evt)
	_, err := s.pool.Exec(ctx, `INSERT INTO governance_event_logs (id, source, event_type, subject, payload, created_at) VALUES ($1,$2,$3,$4,$5,$6)`,
		evt.ID, "cloud", evt.Type, evt.Subject, payload, evt.Time)
	return err
}

func (s *CloudStore) AppendAudit(ctx context.Context, entry audit.AuditLog) error {
	return AppendHashChainedAudit(ctx, s.pool, "cloud", entry)
}
