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
	result := repo.db.WithContext(ctx).Model(&clientRecord{}).Where("id = ?", client.ID.String()).Updates(map[string]any{
		"name":                        client.Name,
		"tier":                        string(client.Tier),
		"default_destination_address": client.DefaultDestinationAddress,
		"total_lifetime_orders":       client.TotalLifetimeOrders,
		"updated_at":                  client.UpdatedAt,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return rootrepo.ErrNotFound
	}
	return nil
}
