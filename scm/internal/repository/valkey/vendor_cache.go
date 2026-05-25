package valkey

import (
	"context"
	"encoding/json"

	"zeus-scm-service/internal/infrastructure/cache"
	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/repository"
)

type VendorCache struct {
	backend cache.Cache
}

func NewVendorCache(backend cache.Cache) repository.IVendorCache {
	return &VendorCache{backend: backend}
}

type cachedOptimalSupplier struct {
	Supplier *models.Supplier   `json:"supplier"`
	Mapping  *models.SkuMapping `json:"mapping"`
}

func (c *VendorCache) GetOptimalSupplier(ctx context.Context, sku string) (*models.Supplier, *models.SkuMapping, error) {
	if c == nil || c.backend == nil {
		return nil, nil, nil
	}
	data, err := c.backend.Get(ctx, c.key(sku))
	if err != nil || data == nil {
		return nil, nil, err
	}
	var cached cachedOptimalSupplier
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, nil, err
	}
	return cached.Supplier, cached.Mapping, nil
}

func (c *VendorCache) SetOptimalSupplier(ctx context.Context, sku string, supplier *models.Supplier, mapping *models.SkuMapping) error {
	if c == nil || c.backend == nil {
		return nil
	}
	cached := cachedOptimalSupplier{
		Supplier: supplier,
		Mapping:  mapping,
	}
	data, err := json.Marshal(cached)
	if err != nil {
		return err
	}
	return c.backend.Set(ctx, c.key(sku), data)
}

func (c *VendorCache) DeleteOptimalSupplier(ctx context.Context, sku string) error {
	if c == nil || c.backend == nil {
		return nil
	}
	return c.backend.Delete(ctx, c.key(sku))
}

func (c *VendorCache) key(sku string) string {
	return "optimal_supplier:" + sku
}
