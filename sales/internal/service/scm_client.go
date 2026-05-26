package service

import "context"

type SCMClient interface {
	CheckSKU(ctx context.Context, sku string) (bool, error)
}
