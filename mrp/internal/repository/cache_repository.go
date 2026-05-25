package repository

import (
	"context"

	"zeus-mrp-service/internal/models"

	"github.com/google/uuid"
)

type CacheRepository interface {
	Set(ctx context.Context, key string, value interface{}) error
	Get(ctx context.Context, key string, dest interface{}) error
	Del(ctx context.Context, keys ...string) error
	GetBOMByModelCode(ctx context.Context, modelCode string, loader func(context.Context, string) ([]models.BomEntry, error)) ([]models.BomEntry, error)
	GetAllBOMs(ctx context.Context, loader func(context.Context) ([]models.BomEntry, error)) ([]models.BomEntry, error)
	GetWhereUsedByPartID(ctx context.Context, partID uuid.UUID, loader func(context.Context, uuid.UUID) ([]models.BomEntry, error)) ([]models.BomEntry, error)
	InvalidateBOM(ctx context.Context, modelCode string, partIDs ...uuid.UUID) error
}

// In the service, you would inject this:
// type productionService struct {
//    repo  repository.MRPRepository
//    cache repository.CacheRepository
// }
