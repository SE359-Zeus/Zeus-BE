package sqlite

import (
	"context"

	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/repository"

	"gorm.io/gorm"
)

type lutRepository struct {
	db *gorm.DB
}

func NewLUTRepository(db *gorm.DB) repository.ILUTRepository {
	return &lutRepository{db: db}
}

func (r *lutRepository) ListPartTypes(ctx context.Context) ([]models.PartType, error) {
	var items []models.PartType
	if err := r.db.WithContext(ctx).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *lutRepository) ListPartConditions(ctx context.Context) ([]models.PartCondition, error) {
	var items []models.PartCondition
	if err := r.db.WithContext(ctx).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *lutRepository) ListPartMfgStatuses(ctx context.Context) ([]models.PartMfgStatus, error) {
	var items []models.PartMfgStatus
	if err := r.db.WithContext(ctx).Where("deleted_at IS NULL").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *lutRepository) ListComponentStockStates(ctx context.Context) ([]models.ComponentStockState, error) {
	var items []models.ComponentStockState
	if err := r.db.WithContext(ctx).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *lutRepository) ListPurchaseOrderStates(ctx context.Context) ([]models.PurchaseOrderState, error) {
	var items []models.PurchaseOrderState
	if err := r.db.WithContext(ctx).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *lutRepository) ListGoodsReceiptStates(ctx context.Context) ([]models.GoodsReceiptState, error) {
	var items []models.GoodsReceiptState
	if err := r.db.WithContext(ctx).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *lutRepository) ListShipmentStates(ctx context.Context) ([]models.ShipmentState, error) {
	var items []models.ShipmentState
	if err := r.db.WithContext(ctx).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}
