package sqlite

import (
	"context"

	"zeus-system-service/internal/models"

	"gorm.io/gorm"
)

type actionTypeRepository struct {
	db *gorm.DB
}

func NewActionTypeRepository(db *gorm.DB) *actionTypeRepository {
	return &actionTypeRepository{db: db}
}

func (r *actionTypeRepository) GetAll(ctx context.Context) ([]models.ActionTypeEntry, error) {
	var entries []models.ActionTypeEntry
	if err := r.db.WithContext(ctx).Find(&entries).Error; err != nil {
		return nil, err
	}
	return entries, nil
}

func (r *actionTypeRepository) Exists(ctx context.Context, name string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.ActionTypeEntry{}).Where("name = ?", name).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
