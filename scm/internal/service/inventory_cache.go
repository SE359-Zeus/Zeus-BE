package service

import (
	"context"

	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/pagination"
	"zeus-scm-service/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type cachedInventoryService struct {
	base         IInventoryService
	productCache repository.IProductCache
}

func NewCachedInventoryService(base IInventoryService, productCache repository.IProductCache) IInventoryService {
	return &cachedInventoryService{base: base, productCache: productCache}
}

func WarmupCache(ctx context.Context, db *gorm.DB, productCache repository.IProductCache) {
	if db == nil || productCache == nil {
		return
	}
	var products []models.Product
	if err := db.WithContext(ctx).Find(&products).Error; err != nil {
		return
	}
	_ = productCache.WarmProducts(ctx, products)
}

func (s *cachedInventoryService) GetProduct(ctx context.Context, id uuid.UUID) (*models.Product, error) {
	if s.productCache != nil {
		if product, err := s.productCache.GetProductByID(ctx, id); err == nil && product != nil {
			return product, nil
		}
	}
	product, err := s.base.GetProduct(ctx, id)
	if err != nil {
		return nil, err
	}
	if s.productCache != nil {
		_ = s.productCache.SetProduct(ctx, product)
	}
	return product, nil
}

func (s *cachedInventoryService) ListProducts(ctx context.Context, params pagination.Params, q string) ([]models.Product, *pagination.Meta, error) {
	return s.base.ListProducts(ctx, params, q)
}

func (s *cachedInventoryService) CreateProduct(ctx context.Context, p *models.Product) error {
	if err := s.base.CreateProduct(ctx, p); err != nil {
		return err
	}
	if s.productCache != nil {
		_ = s.productCache.SetProduct(ctx, p)
	}
	return nil
}

func (s *cachedInventoryService) UpdateProduct(ctx context.Context, id uuid.UUID, fields map[string]any) (*models.Product, error) {
	product, err := s.base.UpdateProduct(ctx, id, fields)
	if err != nil {
		return nil, err
	}
	if s.productCache != nil {
		_ = s.productCache.SetProduct(ctx, product)
	}
	return product, nil
}

func (s *cachedInventoryService) GetProductModel(ctx context.Context, code string) (*models.ProductModel, error) {
	return s.base.GetProductModel(ctx, code)
}

func (s *cachedInventoryService) CreateProductModel(ctx context.Context, m *models.ProductModel) error {
	return s.base.CreateProductModel(ctx, m)
}

func (s *cachedInventoryService) GetPart(ctx context.Context, id uuid.UUID) (*models.Part, error) {
	return s.base.GetPart(ctx, id)
}

func (s *cachedInventoryService) ListParts(ctx context.Context, catalogID *uuid.UUID, productID *uuid.UUID, conditionID *int32, params pagination.Params, q string) ([]models.Part, *pagination.Meta, error) {
	return s.base.ListParts(ctx, catalogID, productID, conditionID, params, q)
}

func (s *cachedInventoryService) CreatePart(ctx context.Context, p *models.Part) error {
	return s.base.CreatePart(ctx, p)
}

func (s *cachedInventoryService) UpdatePart(ctx context.Context, id uuid.UUID, fields map[string]any) (*models.Part, error) {
	return s.base.UpdatePart(ctx, id, fields)
}

func (s *cachedInventoryService) UpdatePartCondition(ctx context.Context, partID uuid.UUID, conditionID int32) error {
	return s.base.UpdatePartCondition(ctx, partID, conditionID)
}

func (s *cachedInventoryService) MarkPartScrapped(ctx context.Context, partID uuid.UUID) error {
	return s.base.MarkPartScrapped(ctx, partID)
}

func (s *cachedInventoryService) InstallPart(ctx context.Context, partID uuid.UUID, productID uuid.UUID) error {
	return s.base.InstallPart(ctx, partID, productID)
}

func (s *cachedInventoryService) RemovePart(ctx context.Context, partID uuid.UUID) error {
	return s.base.RemovePart(ctx, partID)
}

func (s *cachedInventoryService) GetPartCatalog(ctx context.Context, id uuid.UUID) (*models.PartCatalog, error) {
	return s.base.GetPartCatalog(ctx, id)
}

func (s *cachedInventoryService) ListPartCatalog(ctx context.Context, typeID *int32, params pagination.Params, q string) ([]models.PartCatalog, *pagination.Meta, error) {
	return s.base.ListPartCatalog(ctx, typeID, params, q)
}

func (s *cachedInventoryService) CreatePartCatalog(ctx context.Context, pc *models.PartCatalog, price float64) error {
	return s.base.CreatePartCatalog(ctx, pc, price)
}

func (s *cachedInventoryService) UpdatePartCatalogBySKU(ctx context.Context, sku string, fields map[string]any) (*models.PartCatalog, error) {
	return s.base.UpdatePartCatalogBySKU(ctx, sku, fields)
}

func (s *cachedInventoryService) DeletePartCatalogBySKU(ctx context.Context, sku string) error {
	return s.base.DeletePartCatalogBySKU(ctx, sku)
}

func (s *cachedInventoryService) GetPartCatalogBySKU(ctx context.Context, sku string) (*models.PartCatalog, float64, int, error) {
	return s.base.GetPartCatalogBySKU(ctx, sku)
}

func (s *cachedInventoryService) ListStocks(ctx context.Context, params pagination.Params, status, q string) ([]models.ComponentStock, *pagination.Meta, error) {
	return s.base.ListStocks(ctx, params, status, q)
}

func (s *cachedInventoryService) CreateComponentStock(ctx context.Context, stock *models.ComponentStock) error {
	return s.base.CreateComponentStock(ctx, stock)
}

func (s *cachedInventoryService) GetStockBySKU(ctx context.Context, sku string) (*models.ComponentStock, error) {
	return s.base.GetStockBySKU(ctx, sku)
}
