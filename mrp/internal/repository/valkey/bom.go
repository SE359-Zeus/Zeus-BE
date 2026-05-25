package valkey

import (
	"context"
	"errors"
	"fmt"
	"zeus-mrp-service/internal/models"

	"github.com/google/uuid"
)

const (
	bomAllCacheKey = "mrp:bom:all:v1"
	bomByModelKey  = "mrp:bom:model:v1:%s"
	bomWhereKey    = "mrp:bom:where-used:v1:%s"
)

func (repo *Repository) GetBOMByModelCode(ctx context.Context, modelCode string, loader func(context.Context, string) ([]models.BomEntry, error)) ([]models.BomEntry, error) {
	if loader == nil {
		return nil, errors.New("bom loader is required")
	}
	if repo == nil || repo.client == nil || modelCode == "" {
		return loader(ctx, modelCode)
	}

	var cached []models.BomEntry
	if err := repo.Get(ctx, fmt.Sprintf(bomByModelKey, modelCode), &cached); err == nil {
		if cached == nil {
			return []models.BomEntry{}, nil
		}
		return cached, nil
	}

	entries, err := loader(ctx, modelCode)
	if err != nil {
		return nil, err
	}
	_ = repo.Set(ctx, fmt.Sprintf(bomByModelKey, modelCode), entries)
	return entries, nil
}

func (repo *Repository) GetAllBOMs(ctx context.Context, loader func(context.Context) ([]models.BomEntry, error)) ([]models.BomEntry, error) {
	if loader == nil {
		return nil, errors.New("bom loader is required")
	}
	if repo == nil || repo.client == nil {
		return loader(ctx)
	}

	var cached []models.BomEntry
	if err := repo.Get(ctx, bomAllCacheKey, &cached); err == nil {
		if cached == nil {
			return []models.BomEntry{}, nil
		}
		return cached, nil
	}

	entries, err := loader(ctx)
	if err != nil {
		return nil, err
	}
	_ = repo.Set(ctx, bomAllCacheKey, entries)
	return entries, nil
}

func (repo *Repository) GetWhereUsedByPartID(ctx context.Context, partID uuid.UUID, loader func(context.Context, uuid.UUID) ([]models.BomEntry, error)) ([]models.BomEntry, error) {
	if loader == nil {
		return nil, errors.New("bom loader is required")
	}
	if repo == nil || repo.client == nil || partID == uuid.Nil {
		return loader(ctx, partID)
	}

	var cached []models.BomEntry
	if err := repo.Get(ctx, fmt.Sprintf(bomWhereKey, partID.String()), &cached); err == nil {
		if cached == nil {
			return []models.BomEntry{}, nil
		}
		return cached, nil
	}

	entries, err := loader(ctx, partID)
	if err != nil {
		return nil, err
	}
	_ = repo.Set(ctx, fmt.Sprintf(bomWhereKey, partID.String()), entries)
	return entries, nil
}

func (repo *Repository) InvalidateBOM(ctx context.Context, modelCode string, partIDs ...uuid.UUID) error {
	if repo == nil || repo.client == nil {
		return fmt.Errorf("valkey repository or client is nil")
	}
	keys := []string{bomAllCacheKey}
	if modelCode != "" {
		keys = append(keys, fmt.Sprintf(bomByModelKey, modelCode))
	}
	for _, partID := range partIDs {
		if partID == uuid.Nil {
			continue
		}
		keys = append(keys, fmt.Sprintf(bomWhereKey, partID.String()))
	}
	return repo.Del(ctx, keys...)
}
