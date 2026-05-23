package sqlite

import (
	"context"

	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/repository"

	"gorm.io/gorm"
)

type stockRepository struct {
	db *gorm.DB
}

func NewStockRepository(db *gorm.DB) repository.IStockRepository {
	return &stockRepository{db: db}
}

func (r *stockRepository) GetStockBySKU(ctx context.Context, sku string) (*models.ComponentStock, error) {
	var stock models.ComponentStock
	if err := r.db.WithContext(ctx).First(&stock, "sku = ?", sku).Error; err != nil {
		return nil, err
	}
	return &stock, nil
}

func (r *stockRepository) SaveStock(ctx context.Context, stock *models.ComponentStock) error {
	return r.db.WithContext(ctx).Save(stock).Error
}

func (r *stockRepository) UpsertStock(ctx context.Context, stock *models.ComponentStock) error {
	return r.db.WithContext(ctx).Save(stock).Error
}
