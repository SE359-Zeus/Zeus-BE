package service

import "context"

type SCMClient interface {
	CheckSKU(ctx context.Context, sku string) (bool, error)
	GetProductModelPrice(ctx context.Context, sku string) (float64, error)
}
