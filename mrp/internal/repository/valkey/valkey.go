package valkey

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	cacheinfra "zeus-mrp-service/internal/infrastructure/cache"
	"zeus-mrp-service/internal/models"
	"zeus-mrp-service/internal/repository"

	"github.com/google/uuid"
)

const (
	bomAllCacheKey = "mrp:bom:all:v1"
	bomByModelKey  = "mrp:bom:model:v1:%s"
	bomWhereKey    = "mrp:bom:where-used:v1:%s"
)

type Repository struct {
	client cacheinfra.ValkeyConn
	ttl    time.Duration
}

func New(addr string) *Repository {
	return &Repository{
		client: cacheinfra.DialValkey(addr),
		ttl:    5 * time.Minute,
	}
}

func NewWithClient(client cacheinfra.ValkeyConn) *Repository {
	return &Repository{
		client: client,
		ttl:    5 * time.Minute,
	}
}

func (repo *Repository) Set(ctx context.Context, key string, value interface{}) error {
	if repo == nil {
		return errors.New("valkey repository is nil")
	}
	if key == "" {
		return fmt.Errorf("cache key is required")
	}

	payload, err := encodeCacheValue(value)
	if err != nil {
		return err
	}
	if repo.client == nil {
		return cacheinfra.ErrUnavailable
	}

	return repo.client.Set(ctx, key, string(payload), repo.ttl)
}

func (repo *Repository) Get(ctx context.Context, key string, dest interface{}) error {
	if repo == nil {
		return errors.New("valkey repository is nil")
	}
	if key == "" {
		return fmt.Errorf("cache key is required")
	}
	if dest == nil {
		return fmt.Errorf("cache destination is required")
	}
	if repo.client == nil {
		return cacheinfra.ErrUnavailable
	}

	payload, err := repo.client.Get(ctx, key)
	if err != nil {
		return err
	}

	return decodeCacheValue([]byte(payload), dest)
}

func (repo *Repository) Del(ctx context.Context, keys ...string) error {
	if repo == nil {
		return errors.New("valkey repository is nil")
	}
	if repo.client == nil {
		return cacheinfra.ErrUnavailable
	}
	if len(keys) == 0 {
		return nil
	}
	return repo.client.Del(ctx, keys...)
}

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
		return cacheinfra.ErrUnavailable
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

func encodeCacheValue(value interface{}) ([]byte, error) {
	switch typed := value.(type) {
	case nil:
		return []byte("null"), nil
	case []byte:
		return typed, nil
	case string:
		return []byte(typed), nil
	default:
		return json.Marshal(typed)
	}
}

func decodeCacheValue(payload []byte, dest interface{}) error {
	switch typed := dest.(type) {
	case *[]byte:
		*typed = append((*typed)[:0], payload...)
		return nil
	case *string:
		*typed = string(payload)
		return nil
	default:
		return json.Unmarshal(payload, dest)
	}
}

var _ repository.CacheRepository = (*Repository)(nil)
