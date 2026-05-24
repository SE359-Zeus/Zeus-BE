package service

import (
	"context"
	"time"

	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/pagination"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IInventoryService interface {
	GetProduct(ctx context.Context, id uuid.UUID) (*models.Product, error)
	ListProducts(ctx context.Context, params pagination.Params, q string) ([]models.Product, *pagination.Meta, error)
	CreateProduct(ctx context.Context, p *models.Product) error
	UpdateProduct(ctx context.Context, id uuid.UUID, fields map[string]any) (*models.Product, error)

	GetProductModel(ctx context.Context, code string) (*models.ProductModel, error)
	CreateProductModel(ctx context.Context, m *models.ProductModel) error

	GetPart(ctx context.Context, id uuid.UUID) (*models.Part, error)
	ListParts(ctx context.Context, catalogID *uuid.UUID, productID *uuid.UUID, conditionID *int32, params pagination.Params, q string) ([]models.Part, *pagination.Meta, error)
	CreatePart(ctx context.Context, p *models.Part) error
	UpdatePart(ctx context.Context, id uuid.UUID, fields map[string]any) (*models.Part, error)
	UpdatePartCondition(ctx context.Context, partID uuid.UUID, conditionID int32) error
	MarkPartScrapped(ctx context.Context, partID uuid.UUID) error
	InstallPart(ctx context.Context, partID uuid.UUID, productID uuid.UUID) error
	RemovePart(ctx context.Context, partID uuid.UUID) error

	GetPartCatalog(ctx context.Context, id uuid.UUID) (*models.PartCatalog, error)
	ListPartCatalog(ctx context.Context, typeID *int32, params pagination.Params, q string) ([]models.PartCatalog, *pagination.Meta, error)
}

type inventoryService struct {
	db *gorm.DB
}

func NewInventoryService(db *gorm.DB) IInventoryService {
	return &inventoryService{db: db}
}

func (s *inventoryService) GetProduct(ctx context.Context, id uuid.UUID) (*models.Product, error) {
	var p models.Product
	if err := s.db.WithContext(ctx).First(&p, "id = ?", id).Error; err != nil {
		return nil, ErrNotFound
	}
	return &p, nil
}

func (s *inventoryService) ListProducts(ctx context.Context, params pagination.Params, q string) ([]models.Product, *pagination.Meta, error) {
	query := s.db.WithContext(ctx).Model(&models.Product{})
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

func (s *inventoryService) UpdateProduct(ctx context.Context, id uuid.UUID, fields map[string]any) (*models.Product, error) {
	result := s.db.WithContext(ctx).Model(&models.Product{}).Where("id = ?", id).Updates(fields)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, ErrNotFound
	}
	var p models.Product
	if err := s.db.WithContext(ctx).First(&p, "id = ?", id).Error; err != nil {
		return nil, ErrNotFound
	}
	return &p, nil
}

func (s *inventoryService) CreateProduct(ctx context.Context, p *models.Product) error {
	return s.db.WithContext(ctx).Create(p).Error
}

func (s *inventoryService) GetProductModel(ctx context.Context, code string) (*models.ProductModel, error) {
	var m models.ProductModel
	if err := s.db.WithContext(ctx).First(&m, "model_code = ?", code).Error; err != nil {
		return nil, ErrNotFound
	}
	return &m, nil
}

func (s *inventoryService) CreateProductModel(ctx context.Context, m *models.ProductModel) error {
	return s.db.WithContext(ctx).Create(m).Error
}

func (s *inventoryService) GetPart(ctx context.Context, id uuid.UUID) (*models.Part, error) {
	var p models.Part
	if err := s.db.WithContext(ctx).First(&p, "id = ?", id).Error; err != nil {
		return nil, ErrNotFound
	}
	return &p, nil
}

func (s *inventoryService) ListParts(ctx context.Context, catalogID *uuid.UUID, productID *uuid.UUID, conditionID *int32, params pagination.Params, q string) ([]models.Part, *pagination.Meta, error) {
	query := s.db.WithContext(ctx).Model(&models.Part{})
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

func (s *inventoryService) CreatePart(ctx context.Context, p *models.Part) error {
	return s.db.WithContext(ctx).Create(p).Error
}

func (s *inventoryService) UpdatePart(ctx context.Context, id uuid.UUID, fields map[string]any) (*models.Part, error) {
	result := s.db.WithContext(ctx).Model(&models.Part{}).Where("id = ?", id).Updates(fields)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, ErrNotFound
	}
	var p models.Part
	if err := s.db.WithContext(ctx).First(&p, "id = ?", id).Error; err != nil {
		return nil, ErrNotFound
	}
	return &p, nil
}

func (s *inventoryService) UpdatePartCondition(ctx context.Context, partID uuid.UUID, conditionID int32) error {
	result := s.db.WithContext(ctx).Model(&models.Part{}).
		Where("id = ?", partID).
		Update("part_condition_id", conditionID)
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return result.Error
}

func (s *inventoryService) MarkPartScrapped(ctx context.Context, partID uuid.UUID) error {
	now := time.Now()
	result := s.db.WithContext(ctx).Model(&models.Part{}).
		Where("id = ?", partID).
		Update("scrapped_date", now)
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return result.Error
}

func (s *inventoryService) InstallPart(ctx context.Context, partID uuid.UUID, productID uuid.UUID) error {
	now := time.Now()
	result := s.db.WithContext(ctx).Model(&models.Part{}).
		Where("id = ?", partID).
		Updates(map[string]interface{}{
			"product_id":        productID,
			"installation_date": now,
		})
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return result.Error
}

func (s *inventoryService) RemovePart(ctx context.Context, partID uuid.UUID) error {
	now := time.Now()
	result := s.db.WithContext(ctx).Model(&models.Part{}).
		Where("id = ?", partID).
		Updates(map[string]interface{}{
			"product_id":   nil,
			"removal_date": now,
		})
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return result.Error
}

func (s *inventoryService) GetPartCatalog(ctx context.Context, id uuid.UUID) (*models.PartCatalog, error) {
	var c models.PartCatalog
	if err := s.db.WithContext(ctx).First(&c, "id = ?", id).Error; err != nil {
		return nil, ErrNotFound
	}
	return &c, nil
}

func (s *inventoryService) ListPartCatalog(ctx context.Context, typeID *int32, params pagination.Params, q string) ([]models.PartCatalog, *pagination.Meta, error) {
	query := s.db.WithContext(ctx).Model(&models.PartCatalog{})
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
