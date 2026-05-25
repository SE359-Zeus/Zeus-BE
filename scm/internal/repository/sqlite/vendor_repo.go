package sqlite

import (
	"context"

	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type vendorRepository struct {
	db *gorm.DB
}

func NewVendorRepository(db *gorm.DB) repository.IVendorRepository {
	return &vendorRepository{db: db}
}

func (r *vendorRepository) GetSupplierByID(ctx context.Context, id uuid.UUID) (*models.Supplier, error) {
	var s models.Supplier
	if err := r.db.WithContext(ctx).First(&s, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *vendorRepository) FindSkuMappingsBySKU(ctx context.Context, sku string) ([]models.SkuMapping, error) {
	var mappings []models.SkuMapping
	if err := r.db.WithContext(ctx).
		Where("sku = ?", sku).
		Order("unit_price ASC").
		Find(&mappings).Error; err != nil {
		return nil, err
	}
	return mappings, nil
}

func (r *vendorRepository) UpdateSupplier(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.Supplier{}).Where("id = ?", id).Updates(updates).Error
}

func (r *vendorRepository) FindGoodsReceiptsByVendor(ctx context.Context, vendorID uuid.UUID) ([]models.GoodsReceipt, error) {
	var receipts []models.GoodsReceipt
	if err := r.db.WithContext(ctx).Where("vendor_id = ?", vendorID).Find(&receipts).Error; err != nil {
		return nil, err
	}
	return receipts, nil
}

func (r *vendorRepository) CountGoodsReceiptsByVendor(ctx context.Context, vendorID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.GoodsReceipt{}).
		Where("vendor_id = ?", vendorID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *vendorRepository) FindGRLineItemsByGRID(ctx context.Context, grID string) ([]models.GRLineItem, error) {
	var items []models.GRLineItem
	if err := r.db.WithContext(ctx).Where("gr_id = ?", grID).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}
