package service

import (
	"context"
	"encoding/json"
	"log"

	"zeus-scm-service/internal/infrastructure/cache"
	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/pagination"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type cachedInventoryService struct {
	inner IInventoryService
	cache cache.Cache
}

func NewCachedInventoryService(inner IInventoryService, c cache.Cache) IInventoryService {
	return &cachedInventoryService{inner: inner, cache: c}
}

func cacheKey(prefix, id string) string {
	return "scm:" + prefix + ":" + id
}

func (s *cachedInventoryService) GetProduct(ctx context.Context, id uuid.UUID) (*models.Product, error) {
	key := cacheKey("product", id.String())
	if data, err := s.cache.Get(ctx, key); err == nil && data != nil {
		var p models.Product
		if err := json.Unmarshal(data, &p); err == nil {
			return &p, nil
		}
		log.Printf("cache decode error for %s: %v", key, err)
	}
	p, err := s.inner.GetProduct(ctx, id)
	if err != nil {
		return nil, err
	}
	if data, err := json.Marshal(p); err == nil {
		s.cache.Set(ctx, key, data)
	}
	return p, nil
}

func (s *cachedInventoryService) ListProducts(ctx context.Context, params pagination.Params, q string) ([]models.Product, *pagination.Meta, error) {
	return s.inner.ListProducts(ctx, params, q)
}

func (s *cachedInventoryService) CreateProduct(ctx context.Context, p *models.Product) error {
	if err := s.inner.CreateProduct(ctx, p); err != nil {
		return err
	}
	key := cacheKey("product", p.ID.String())
	if data, err := json.Marshal(p); err == nil {
		s.cache.Set(ctx, key, data)
	}
	return nil
}

func (s *cachedInventoryService) UpdateProduct(ctx context.Context, id uuid.UUID, fields map[string]any) (*models.Product, error) {
	p, err := s.inner.UpdateProduct(ctx, id, fields)
	if err != nil {
		return nil, err
	}
	key := cacheKey("product", id.String())
	if data, err := json.Marshal(p); err == nil {
		s.cache.Set(ctx, key, data)
	}
	return p, nil
}

func (s *cachedInventoryService) GetProductModel(ctx context.Context, code string) (*models.ProductModel, error) {
	key := cacheKey("product_model", code)
	if data, err := s.cache.Get(ctx, key); err == nil && data != nil {
		var m models.ProductModel
		if err := json.Unmarshal(data, &m); err == nil {
			return &m, nil
		}
		log.Printf("cache decode error for %s: %v", key, err)
	}
	m, err := s.inner.GetProductModel(ctx, code)
	if err != nil {
		return nil, err
	}
	if data, err := json.Marshal(m); err == nil {
		s.cache.Set(ctx, key, data)
	}
	return m, nil
}

func (s *cachedInventoryService) CreateProductModel(ctx context.Context, m *models.ProductModel) error {
	if err := s.inner.CreateProductModel(ctx, m); err != nil {
		return err
	}
	key := cacheKey("product_model", m.ModelCode)
	if data, err := json.Marshal(m); err == nil {
		s.cache.Set(ctx, key, data)
	}
	return nil
}

func (s *cachedInventoryService) GetPart(ctx context.Context, id uuid.UUID) (*models.Part, error) {
	key := cacheKey("part", id.String())
	if data, err := s.cache.Get(ctx, key); err == nil && data != nil {
		var p models.Part
		if err := json.Unmarshal(data, &p); err == nil {
			return &p, nil
		}
		log.Printf("cache decode error for %s: %v", key, err)
	}
	p, err := s.inner.GetPart(ctx, id)
	if err != nil {
		return nil, err
	}
	if data, err := json.Marshal(p); err == nil {
		s.cache.Set(ctx, key, data)
	}
	return p, nil
}

func (s *cachedInventoryService) ListParts(ctx context.Context, catalogID *uuid.UUID, productID *uuid.UUID, conditionID *int32, params pagination.Params, q string) ([]models.Part, *pagination.Meta, error) {
	return s.inner.ListParts(ctx, catalogID, productID, conditionID, params, q)
}

func (s *cachedInventoryService) CreatePart(ctx context.Context, p *models.Part) error {
	if err := s.inner.CreatePart(ctx, p); err != nil {
		return err
	}
	key := cacheKey("part", p.ID.String())
	if data, err := json.Marshal(p); err == nil {
		s.cache.Set(ctx, key, data)
	}
	return nil
}

func (s *cachedInventoryService) UpdatePart(ctx context.Context, id uuid.UUID, fields map[string]any) (*models.Part, error) {
	p, err := s.inner.UpdatePart(ctx, id, fields)
	if err != nil {
		return nil, err
	}
	key := cacheKey("part", id.String())
	if data, err := json.Marshal(p); err == nil {
		s.cache.Set(ctx, key, data)
	}
	return p, nil
}

func (s *cachedInventoryService) UpdatePartCondition(ctx context.Context, partID uuid.UUID, conditionID int32) error {
	if err := s.inner.UpdatePartCondition(ctx, partID, conditionID); err != nil {
		return err
	}
	s.cache.Delete(ctx, cacheKey("part", partID.String()))
	return nil
}

func (s *cachedInventoryService) MarkPartScrapped(ctx context.Context, partID uuid.UUID) error {
	if err := s.inner.MarkPartScrapped(ctx, partID); err != nil {
		return err
	}
	s.cache.Delete(ctx, cacheKey("part", partID.String()))
	return nil
}

func (s *cachedInventoryService) InstallPart(ctx context.Context, partID uuid.UUID, productID uuid.UUID) error {
	if err := s.inner.InstallPart(ctx, partID, productID); err != nil {
		return err
	}
	s.cache.Delete(ctx, cacheKey("part", partID.String()))
	return nil
}

func (s *cachedInventoryService) RemovePart(ctx context.Context, partID uuid.UUID) error {
	if err := s.inner.RemovePart(ctx, partID); err != nil {
		return err
	}
	s.cache.Delete(ctx, cacheKey("part", partID.String()))
	return nil
}

func (s *cachedInventoryService) GetPartCatalog(ctx context.Context, id uuid.UUID) (*models.PartCatalog, error) {
	key := cacheKey("part_catalog", id.String())
	if data, err := s.cache.Get(ctx, key); err == nil && data != nil {
		var c models.PartCatalog
		if err := json.Unmarshal(data, &c); err == nil {
			return &c, nil
		}
		log.Printf("cache decode error for %s: %v", key, err)
	}
	c, err := s.inner.GetPartCatalog(ctx, id)
	if err != nil {
		return nil, err
	}
	if data, err := json.Marshal(c); err == nil {
		s.cache.Set(ctx, key, data)
	}
	return c, nil
}

func (s *cachedInventoryService) ListPartCatalog(ctx context.Context, typeID *int32, params pagination.Params, q string) ([]models.PartCatalog, *pagination.Meta, error) {
	return s.inner.ListPartCatalog(ctx, typeID, params, q)
}

func WarmupCache(ctx context.Context, db *gorm.DB, c cache.Cache) {
	log.Println("cache warmup started")

	var products []models.Product
	db.WithContext(ctx).Model(&models.Product{}).Find(&products)
	prodEntries := make(map[string][]byte, len(products))
	for i := range products {
		key := cacheKey("product", products[i].ID.String())
		data, _ := json.Marshal(products[i])
		prodEntries[key] = data
	}
	if len(prodEntries) > 0 {
		c.Warm(ctx, prodEntries)
		log.Printf("cache warmup: %d products", len(products))
	}

	var parts []models.Part
	db.WithContext(ctx).Model(&models.Part{}).Find(&parts)
	partEntries := make(map[string][]byte, len(parts))
	for i := range parts {
		key := cacheKey("part", parts[i].ID.String())
		data, _ := json.Marshal(parts[i])
		partEntries[key] = data
	}
	if len(partEntries) > 0 {
		c.Warm(ctx, partEntries)
		log.Printf("cache warmup: %d parts", len(parts))
	}

	var catalogs []models.PartCatalog
	db.WithContext(ctx).Model(&models.PartCatalog{}).Find(&catalogs)
	catEntries := make(map[string][]byte, len(catalogs))
	for i := range catalogs {
		key := cacheKey("part_catalog", catalogs[i].ID.String())
		data, _ := json.Marshal(catalogs[i])
		catEntries[key] = data
	}
	if len(catEntries) > 0 {
		c.Warm(ctx, catEntries)
		log.Printf("cache warmup: %d part catalogs", len(catalogs))
	}

	var modelsList []models.ProductModel
	db.WithContext(ctx).Model(&models.ProductModel{}).Find(&modelsList)
	modelEntries := make(map[string][]byte, len(modelsList))
	for i := range modelsList {
		key := cacheKey("product_model", modelsList[i].ModelCode)
		data, _ := json.Marshal(modelsList[i])
		modelEntries[key] = data
	}
	if len(modelEntries) > 0 {
		c.Warm(ctx, modelEntries)
		log.Printf("cache warmup: %d product models", len(modelsList))
	}

	log.Println("cache warmup complete")
}
