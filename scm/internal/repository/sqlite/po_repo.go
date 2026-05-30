package sqlite

import (
	"context"
	"fmt"

	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/pagination"
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
	if err := r.db.WithContext(ctx).Preload("LineItems").Preload("Vendor").First(&po, "id = ?", id).Error; err != nil {
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

func (r *poRepository) SavePOLineItem(ctx context.Context, item *models.POLineItem) error {
	return r.db.WithContext(ctx).Save(item).Error
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

func (r *poRepository) ListPOs(ctx context.Context, params pagination.Params, q string) ([]models.PurchaseOrder, *pagination.Meta, error) {
	query := r.db.WithContext(ctx).Model(&models.PurchaseOrder{}).Preload("LineItems").Preload("Vendor")
	if q != "" {
		like := "%" + q + "%"
		query = query.Where("id LIKE ? OR target_build LIKE ? OR status LIKE ?", like, like, like)
	}
	var pos []models.PurchaseOrder
	meta, err := pagination.Paginate(query, params, &pos, "created_at", "updated_at", "id", "status")
	if err != nil {
		return nil, nil, err
	}
	return pos, meta, nil
}

func (r *poRepository) FindAllPOs(ctx context.Context) ([]models.PurchaseOrder, error) {
	var pos []models.PurchaseOrder
	if err := r.db.WithContext(ctx).Preload("LineItems").Preload("Vendor").Order("created_at DESC").Find(&pos).Error; err != nil {
		return nil, err
	}
	return pos, nil
}

func (r *poRepository) FindSkuMapping(ctx context.Context, vendorID uuid.UUID, sku string) (*models.SkuMapping, error) {
	var mapping models.SkuMapping
	if err := r.db.WithContext(ctx).Where("supplier_id = ? AND sku = ?", vendorID, sku).First(&mapping).Error; err != nil {
		return nil, err
	}
	return &mapping, nil
}

func (r *poRepository) GetPOMetrics(ctx context.Context) (total int64, draft int64, approved int64, inTransit int64, received int64, partial int64, void int64, err error) {
	db := r.db.WithContext(ctx).Model(&models.PurchaseOrder{})
	err = db.Count(&total).Error
	if err != nil {
		return
	}
	err = db.Where("status = ?", models.POStatusDraft).Count(&draft).Error
	if err != nil {
		return
	}
	err = db.Where("status = ?", models.POStatusApproved).Count(&approved).Error
	if err != nil {
		return
	}
	err = db.Where("status = ?", models.POStatusInTransit).Count(&inTransit).Error
	if err != nil {
		return
	}
	err = db.Where("status = ?", models.POStatusReceived).Count(&received).Error
	if err != nil {
		return
	}
	err = db.Where("status = ?", models.POStatusPartial).Count(&partial).Error
	if err != nil {
		return
	}
	err = db.Where("status = ?", models.POStatusVoid).Count(&void).Error
	return
}
