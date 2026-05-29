package sqlite

import (
	"context"
	"strings"

	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/pagination"
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

func (r *vendorRepository) FindSkuMappingsBySupplierID(ctx context.Context, supplierID uuid.UUID) ([]models.SkuMapping, error) {
	var mappings []models.SkuMapping
	if err := r.db.WithContext(ctx).Where("supplier_id = ?", supplierID).Find(&mappings).Error; err != nil {
		return nil, err
	}
	return mappings, nil
}

func (r *vendorRepository) ListSuppliers(ctx context.Context, tier string, params pagination.Params, q string) ([]models.Supplier, *pagination.Meta, error) {
	query := r.db.WithContext(ctx).Model(&models.Supplier{}).Preload("SkuMappings")
	if tier != "" {
		switch strings.ToLower(tier) {
		case "tier 1", "tier1":
			query = query.Where("tier = ?", models.SupplierTier1)
		case "tier 2", "tier2":
			query = query.Where("tier = ?", models.SupplierTier2)
		case "tier 3", "tier3":
			query = query.Where("tier = ?", models.SupplierTier3)
		default:
			query = query.Where("tier = ?", tier)
		}
	}
	if q != "" {
		like := "%" + q + "%"
		query = query.Where("name LIKE ? OR contact LIKE ? OR id LIKE ?", like, like, like)
	}
	var suppliers []models.Supplier
	meta, err := pagination.Paginate(query, params, &suppliers, "created_at", "updated_at", "name", "contact")
	if err != nil {
		return nil, nil, err
	}
	return suppliers, meta, nil
}

func (r *vendorRepository) CreateSupplier(ctx context.Context, supplier *models.Supplier) error {
	return r.db.WithContext(ctx).Create(supplier).Error
}

func (r *vendorRepository) CreateSkuMapping(ctx context.Context, mapping *models.SkuMapping) error {
	return r.db.WithContext(ctx).Create(mapping).Error
}

func (r *vendorRepository) CountSuppliers(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.Supplier{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *vendorRepository) GetAverageOnTimeRate(ctx context.Context) (float64, error) {
	var avg float64
	err := r.db.WithContext(ctx).Model(&models.Supplier{}).Select("COALESCE(AVG(on_time_rate), 0)").Row().Scan(&avg)
	if err != nil {
		return 0, err
	}
	return avg, nil
}

func (r *vendorRepository) FindAllSuppliersWithMappings(ctx context.Context) ([]models.Supplier, error) {
	var suppliers []models.Supplier
	if err := r.db.WithContext(ctx).Preload("SkuMappings").Order("name ASC").Find(&suppliers).Error; err != nil {
		return nil, err
	}
	return suppliers, nil
}
