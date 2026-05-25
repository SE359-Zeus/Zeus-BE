package repository

import (
	"context"

	"zeus-scm-service/internal/models"

	"github.com/google/uuid"
)

type IProductCache interface {
	GetProductByID(ctx context.Context, id uuid.UUID) (*models.Product, error)
	SetProduct(ctx context.Context, p *models.Product) error
	DeleteProduct(ctx context.Context, id uuid.UUID) error
	WarmProducts(ctx context.Context, products []models.Product) error
}
