package sqlite

import (
	"context"
	"fmt"

	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type poRepository struct {
	db *gorm.DB
}

func NewPORepository(db *gorm.DB) repository.IPORepository {
	return &poRepository{db: db}
}

func (r *poRepository) GetPOByID(ctx context.Context, id string) (*models.PurchaseOrder, error) {
	var po models.PurchaseOrder
	if err := r.db.WithContext(ctx).First(&po, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &po, nil
}

func (r *poRepository) CreatePO(ctx context.Context, po *models.PurchaseOrder) error {
	return r.db.WithContext(ctx).Create(po).Error
}

func (r *poRepository) SavePO(ctx context.Context, po *models.PurchaseOrder) error {
	return r.db.WithContext(ctx).Save(po).Error
}

func (r *poRepository) UpdatePOStatus(ctx context.Context, id string, status models.POStatus) error {
	return r.db.WithContext(ctx).Model(&models.PurchaseOrder{}).Where("id = ?", id).Update("status", status).Error
}

func (r *poRepository) CreatePOLineItem(ctx context.Context, item *models.POLineItem) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *poRepository) GetPOLineItemsByPOID(ctx context.Context, poID string) ([]models.POLineItem, error) {
	var items []models.POLineItem
	if err := r.db.WithContext(ctx).Where("po_id = ?", poID).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *poRepository) CountPOsByYearPattern(ctx context.Context, year int, pattern string) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.PurchaseOrder{}).
		Where("id LIKE ?", fmt.Sprintf(pattern, year)).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *poRepository) FindPOByVendorAndStatuses(ctx context.Context, vendorID uuid.UUID, statuses []models.POStatus) (*models.PurchaseOrder, error) {
	var po models.PurchaseOrder
	if err := r.db.WithContext(ctx).
		Where("vendor_id = ? AND status IN ?", vendorID, statuses).
		First(&po).Error; err != nil {
		return nil, err
	}
	return &po, nil
}
