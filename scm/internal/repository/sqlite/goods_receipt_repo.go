package sqlite

import (
	"context"
	"time"

	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/pagination"
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
	if err := r.db.WithContext(ctx).Preload("LineItems").Preload("Vendor").First(&gr, "id = ?", id).Error; err != nil {
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

func (r *goodsReceiptRepository) ListGRs(ctx context.Context, status string, params pagination.Params) ([]models.GoodsReceipt, *pagination.Meta, error) {
	query := r.db.WithContext(ctx).Model(&models.GoodsReceipt{}).Preload("LineItems").Preload("Vendor")
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var grs []models.GoodsReceipt
	meta, err := pagination.Paginate(query, params, &grs, "created_at", "updated_at", "id", "status", "arrival_date")
	if err != nil {
		return nil, nil, err
	}
	return grs, meta, nil
}

func (r *goodsReceiptRepository) GetMetrics(ctx context.Context) (pending int64, completedToday int64, discrepancies int64, queue int64, err error) {
	err = r.db.WithContext(ctx).Model(&models.GoodsReceipt{}).Where("status = ?", models.GRStatusPending).Count(&pending).Error
	if err != nil {
		return
	}

	todayStart := time.Now().Truncate(24 * time.Hour)
	err = r.db.WithContext(ctx).Model(&models.GoodsReceipt{}).
		Where("status = ? AND updated_at >= ?", models.GRStatusComplete, todayStart).
		Count(&completedToday).Error
	if err != nil {
		return
	}

	err = r.db.WithContext(ctx).Model(&models.GoodsReceipt{}).Where("status = ?", models.GRStatusDiscrepancy).Count(&discrepancies).Error
	if err != nil {
		return
	}

	err = r.db.WithContext(ctx).Model(&models.GoodsReceipt{}).Where("status = ?", models.GRStatusInspected).Count(&queue).Error
	return
}
