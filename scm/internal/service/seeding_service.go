package service

import (
	"context"

	"zeus-scm-service/internal/models"
)

// ISeedingService handles product assembly: creating products (with automatic
// part generation from the BOM) and standalone parts.
type ISeedingService interface {
	CreateProduct(ctx context.Context, p *models.Product) error
	CreatePart(ctx context.Context, p *models.Part) error
}

type seedingService struct {
	inventorySvc IInventoryService
}

func NewSeedingService(inventorySvc IInventoryService) ISeedingService {
	return &seedingService{inventorySvc: inventorySvc}
}

func (s *seedingService) CreateProduct(ctx context.Context, p *models.Product) error {
	return s.inventorySvc.CreateProduct(ctx, p)
}

func (s *seedingService) CreatePart(ctx context.Context, p *models.Part) error {
	return s.inventorySvc.CreatePart(ctx, p)
}
