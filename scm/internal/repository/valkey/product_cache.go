package valkey

import (
	"context"
	"encoding/json"
	"fmt"

	"zeus-scm-service/internal/infrastructure/cache"
	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/repository"

	"github.com/google/uuid"
)

type ProductCache struct {
	backend cache.Cache
}

func NewProductCache(backend cache.Cache) repository.IProductCache {
	return &ProductCache{backend: backend}
}

func (c *ProductCache) GetProductByID(ctx context.Context, id uuid.UUID) (*models.Product, error) {
	if c == nil || c.backend == nil {
		return nil, nil
	}
	data, err := c.backend.Get(ctx, c.key(id))
	if err != nil || data == nil {
		return nil, err
	}
	var product models.Product
	if err := json.Unmarshal(data, &product); err != nil {
		return nil, err
	}
	return &product, nil
}

func (c *ProductCache) SetProduct(ctx context.Context, p *models.Product) error {
	if c == nil || c.backend == nil || p == nil {
		return nil
	}
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	return c.backend.Set(ctx, c.key(p.ID), data)
}

func (c *ProductCache) DeleteProduct(ctx context.Context, id uuid.UUID) error {
	if c == nil || c.backend == nil {
		return nil
	}
	return c.backend.Delete(ctx, c.key(id))
}

func (c *ProductCache) WarmProducts(ctx context.Context, products []models.Product) error {
	for i := range products {
		if err := c.SetProduct(ctx, &products[i]); err != nil {
			return fmt.Errorf("warm product %s: %w", products[i].ID.String(), err)
		}
	}
	return nil
}

func (c *ProductCache) key(id uuid.UUID) string {
	return "product:" + id.String()
}
