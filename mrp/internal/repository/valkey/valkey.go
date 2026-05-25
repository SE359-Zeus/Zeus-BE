package valkey

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	cacheinfra "zeus-mrp-service/internal/infrastructure/cache"
	"zeus-mrp-service/internal/repository"
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
