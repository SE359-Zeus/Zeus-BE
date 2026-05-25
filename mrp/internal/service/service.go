package service

import (
	"context"
	"zeus-mrp-service/internal/models"
	"zeus-mrp-service/internal/repository"
)

type SCMClient interface {
	GetPartCatalogBySKU(ctx context.Context, sku string) (*models.Part, error)
	CreateCatalogPart(ctx context.Context, sku, description string, price float64) (*models.Part, error)
	UpdateCatalogPart(ctx context.Context, sku, description string, price float64) (*models.Part, error)
	DeleteCatalogPart(ctx context.Context, sku string) error
}

type ProductionService struct {
	repo      repository.MRPRepository
	cache     repository.CacheRepository
	scmClient SCMClient
}

func NewProductionService(repo repository.MRPRepository, deps ...any) *ProductionService {
	svc := &ProductionService{repo: repo}
	for _, dep := range deps {
		switch d := dep.(type) {
		case SCMClient:
			svc.scmClient = d
		case repository.CacheRepository:
			svc.cache = d
		}
	}
	return svc
}
