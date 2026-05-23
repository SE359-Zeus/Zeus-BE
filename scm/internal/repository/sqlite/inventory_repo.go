package sqlite

import (
	"context"

	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/pagination"
	"zeus-scm-service/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type inventoryRepository struct {
	db *gorm.DB
}

func NewInventoryRepository(db *gorm.DB) repository.IInventoryRepository {
	return &inventoryRepository{db: db}
}

func (r *inventoryRepository) GetProductByID(ctx context.Context, id uuid.UUID) (*models.Product, error) {
	var p models.Product
	if err := r.db.WithContext(ctx).First(&p, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *inventoryRepository) ListProducts(ctx context.Context, params pagination.Params, q string) ([]models.Product, *pagination.Meta, error) {
	query := r.db.WithContext(ctx).Model(&models.Product{})
	if q != "" {
		like := "%" + q + "%"
		query = query.Where(
			"product_name LIKE ? OR serial_number LIKE ?",
			like, like,
		)
	}
	var products []models.Product
	meta, err := pagination.Paginate(query, params, &products, "created_at", "updated_at", "product_name", "serial_number")
	if err != nil {
		return nil, nil, err
	}
	return products, meta, nil
}

func (r *inventoryRepository) CreateProduct(ctx context.Context, p *models.Product) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *inventoryRepository) UpdateProduct(ctx context.Context, id uuid.UUID, fields map[string]any) (int64, error) {
	result := r.db.WithContext(ctx).Model(&models.Product{}).Where("id = ?", id).Updates(fields)
	return result.RowsAffected, result.Error
}

func (r *inventoryRepository) GetProductModelByCode(ctx context.Context, code string) (*models.ProductModel, error) {
	var m models.ProductModel
	if err := r.db.WithContext(ctx).First(&m, "model_code = ?", code).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *inventoryRepository) CreateProductModel(ctx context.Context, m *models.ProductModel) error {
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *inventoryRepository) GetPartByID(ctx context.Context, id uuid.UUID) (*models.Part, error) {
	var p models.Part
	if err := r.db.WithContext(ctx).First(&p, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *inventoryRepository) ListParts(ctx context.Context, catalogID *uuid.UUID, productID *uuid.UUID, conditionID *int32, params pagination.Params, q string) ([]models.Part, *pagination.Meta, error) {
	query := r.db.WithContext(ctx).Model(&models.Part{})
	if catalogID != nil {
		query = query.Where("part_catalog_id = ?", *catalogID)
	}
	if productID != nil {
		query = query.Where("product_id = ?", *productID)
	}
	if conditionID != nil {
		query = query.Where("part_condition_id = ?", *conditionID)
	}
	if q != "" {
		like := "%" + q + "%"
		query = query.Where("serial_number LIKE ?", like)
	}
	var parts []models.Part
	meta, err := pagination.Paginate(query, params, &parts, "created_at", "updated_at", "serial_number", "part_condition_id")
	if err != nil {
		return nil, nil, err
	}
	return parts, meta, nil
}

func (r *inventoryRepository) CreatePart(ctx context.Context, p *models.Part) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *inventoryRepository) UpdatePart(ctx context.Context, id uuid.UUID, fields map[string]any) (int64, error) {
	result := r.db.WithContext(ctx).Model(&models.Part{}).Where("id = ?", id).Updates(fields)
	return result.RowsAffected, result.Error
}

func (r *inventoryRepository) UpdatePartFields(ctx context.Context, id uuid.UUID, updates map[string]interface{}) (int64, error) {
	result := r.db.WithContext(ctx).Model(&models.Part{}).Where("id = ?", id).Updates(updates)
	return result.RowsAffected, result.Error
}

func (r *inventoryRepository) GetPartCatalogByID(ctx context.Context, id uuid.UUID) (*models.PartCatalog, error) {
	var c models.PartCatalog
	if err := r.db.WithContext(ctx).First(&c, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *inventoryRepository) ListPartCatalog(ctx context.Context, typeID *int32, params pagination.Params, q string) ([]models.PartCatalog, *pagination.Meta, error) {
	query := r.db.WithContext(ctx).Model(&models.PartCatalog{})
	if typeID != nil {
		query = query.Where("part_types_id = ?", *typeID)
	}
	if q != "" {
		like := "%" + q + "%"
		query = query.Where(
			"part_number LIKE ? OR mfg_number LIKE ? OR description LIKE ?",
			like, like, like,
		)
	}
	var catalogs []models.PartCatalog
	meta, err := pagination.Paginate(query, params, &catalogs, "created_at", "updated_at", "part_number", "mfg_number")
	if err != nil {
		return nil, nil, err
	}
	return catalogs, meta, nil
}
