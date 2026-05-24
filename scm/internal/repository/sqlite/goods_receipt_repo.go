package sqlite

import (
	"context"

	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/repository"

	"gorm.io/gorm"
)

type goodsReceiptRepository struct {
	db *gorm.DB
}

func NewGoodsReceiptRepository(db *gorm.DB) repository.IGoodsReceiptRepository {
	return &goodsReceiptRepository{db: db}
}

func (r *goodsReceiptRepository) GetGRByID(ctx context.Context, id string) (*models.GoodsReceipt, error) {
	var gr models.GoodsReceipt
	if err := r.db.WithContext(ctx).First(&gr, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &gr, nil
}

func (r *goodsReceiptRepository) UpdateGR(ctx context.Context, gr *models.GoodsReceipt) error {
	return r.db.WithContext(ctx).Save(gr).Error
}

func (r *goodsReceiptRepository) UpdateGRFields(ctx context.Context, id string, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.GoodsReceipt{}).Where("id = ?", id).Updates(fields).Error
}

func (r *goodsReceiptRepository) FindGRLineItemsByGRID(ctx context.Context, grID string) ([]models.GRLineItem, error) {
	var items []models.GRLineItem
	if err := r.db.WithContext(ctx).Where("gr_id = ?", grID).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *goodsReceiptRepository) SaveGRLineItem(ctx context.Context, item *models.GRLineItem) error {
	return r.db.WithContext(ctx).Save(item).Error
}
