package sqlite

import (
	"context"

	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/repository"

	"gorm.io/gorm"
)

type carrierRepository struct {
	db *gorm.DB
}

func NewCarrierRepository(db *gorm.DB) repository.ICarrierRepository {
	return &carrierRepository{db: db}
}

func (r *carrierRepository) ListCarriers(ctx context.Context) ([]models.Carrier, error) {
	var carriers []models.Carrier
	if err := r.db.WithContext(ctx).Order("name ASC").Find(&carriers).Error; err != nil {
		return nil, err
	}
	return carriers, nil
}
