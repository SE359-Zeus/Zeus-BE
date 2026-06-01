package sqlite

import (
	"context"
	"strings"
	"time"

	"zeus-sales-service/internal/models"
	rootrepo "zeus-sales-service/internal/repository"

	"github.com/google/uuid"
)

type clientRecord struct {
	ID                        string    `gorm:"primaryKey;column:id"`
	Name                      string    `gorm:"column:name;uniqueIndex"`
	Tier                      string    `gorm:"column:tier"`
	DefaultDestinationAddress string    `gorm:"column:default_destination_address"`
	TotalLifetimeOrders       int       `gorm:"column:total_lifetime_orders"`
	ApiKeyPrefix              string    `gorm:"column:api_key_prefix"`
	ApiKeyHash                string    `gorm:"column:api_key_hash"`
	CreatedAt                 time.Time `gorm:"column:created_at"`
	UpdatedAt                 time.Time `gorm:"column:updated_at"`
}

func (clientRecord) TableName() string { return "clients" }

func clientRecordFromModel(client *models.Client) *clientRecord {
	return &clientRecord{
		ID:                        client.ID.String(),
		Name:                      client.Name,
		Tier:                      string(client.Tier),
		DefaultDestinationAddress: client.DefaultDestinationAddress,
		TotalLifetimeOrders:       client.TotalLifetimeOrders,
		ApiKeyPrefix:              client.ApiKeyPrefix,
		ApiKeyHash:                client.ApiKeyHash,
		CreatedAt:                 client.CreatedAt,
		UpdatedAt:                 client.UpdatedAt,
	}
}

func (record clientRecord) toModel() models.Client {
	parsedID, _ := uuid.Parse(record.ID)
	return models.Client{
		ID:                        parsedID,
		Name:                      record.Name,
		Tier:                      models.ClientTier(record.Tier),
		DefaultDestinationAddress: record.DefaultDestinationAddress,
		TotalLifetimeOrders:       record.TotalLifetimeOrders,
		ApiKeyPrefix:              record.ApiKeyPrefix,
		ApiKeyHash:                record.ApiKeyHash,
		CreatedAt:                 record.CreatedAt,
		UpdatedAt:                 record.UpdatedAt,
	}
}

func (repo *Repository) CreateClient(ctx context.Context, client *models.Client) error {
	if client.ID == uuid.Nil {
		client.ID = uuid.New()
	}
	now := time.Now().UTC()
	if client.CreatedAt.IsZero() {
		client.CreatedAt = now
	}
	client.UpdatedAt = now
	return repo.db.WithContext(ctx).Create(clientRecordFromModel(client)).Error
}

func (repo *Repository) GetClient(ctx context.Context, id uuid.UUID) (*models.Client, error) {
	var record clientRecord
	if err := repo.db.WithContext(ctx).First(&record, "id = ?", id.String()).Error; err != nil {
		return nil, mapRecordError(err)
	}
	model := record.toModel()
	return &model, nil
}

func (repo *Repository) GetClientByName(ctx context.Context, name string) (*models.Client, error) {
	var record clientRecord
	if err := repo.db.WithContext(ctx).Where("lower(name) = lower(?)", strings.TrimSpace(name)).First(&record).Error; err != nil {
		return nil, mapRecordError(err)
	}
	model := record.toModel()
	return &model, nil
}

func (repo *Repository) ExistsClientByName(ctx context.Context, name string) (bool, error) {
	var count int64
	if err := repo.db.WithContext(ctx).Model(&clientRecord{}).Where("lower(name) = lower(?)", strings.TrimSpace(name)).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (repo *Repository) ListClients(ctx context.Context) ([]models.Client, error) {
	var records []clientRecord
	if err := repo.db.WithContext(ctx).Order("name ASC").Find(&records).Error; err != nil {
		return nil, err
	}
	clients := make([]models.Client, 0, len(records))
	for _, record := range records {
		clients = append(clients, record.toModel())
	}
	return clients, nil
}

func (repo *Repository) UpdateClient(ctx context.Context, client *models.Client) error {
	client.UpdatedAt = time.Now().UTC()
	updates := map[string]any{
		"name":                        client.Name,
		"tier":                        string(client.Tier),
		"default_destination_address": client.DefaultDestinationAddress,
		"total_lifetime_orders":       client.TotalLifetimeOrders,
		"updated_at":                  client.UpdatedAt,
	}
	// Only update API key fields when explicitly provided.
	// Cached clients (json:"-") lose these fields, so empty values
	// would silently overwrite valid DB data.
	if client.ApiKeyPrefix != "" {
		updates["api_key_prefix"] = client.ApiKeyPrefix
	}
	if client.ApiKeyHash != "" {
		updates["api_key_hash"] = client.ApiKeyHash
	}
	result := repo.db.WithContext(ctx).Model(&clientRecord{}).Where("id = ?", client.ID.String()).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return rootrepo.ErrNotFound
	}
	return nil
}

func (repo *Repository) DeleteClient(ctx context.Context, id uuid.UUID) error {
	result := repo.db.WithContext(ctx).Delete(&clientRecord{}, "id = ?", id.String())
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return rootrepo.ErrNotFound
	}
	return nil
}

func (repo *Repository) GetClientByAPIKeyPrefix(ctx context.Context, prefix string) (*models.Client, error) {
	var record clientRecord
	if err := repo.db.WithContext(ctx).Where("api_key_prefix = ?", prefix).First(&record).Error; err != nil {
		return nil, mapRecordError(err)
	}
	model := record.toModel()
	return &model, nil
}

