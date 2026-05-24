package valkey

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"zeus-mrp-service/internal/repository"

	"github.com/redis/go-redis/v9"
)

type Repository struct {
	client *redis.Client
	ttl    time.Duration
}

func New(client *redis.Client) *Repository {
	return &Repository{
		client: client,
		ttl:    5 * time.Minute,
	}
}

func (repo *Repository) Set(ctx context.Context, key string, value interface{}) error {
	if repo == nil || repo.client == nil {
		return errors.New("valkey client is nil")
	}
	if key == "" {
		return fmt.Errorf("cache key is required")
	}

	payload, err := encodeCacheValue(value)
	if err != nil {
		return err
	}

	return repo.client.Set(ctx, key, payload, repo.ttl).Err()
}

func (repo *Repository) Get(ctx context.Context, key string, dest interface{}) error {
	if repo == nil || repo.client == nil {
		return errors.New("valkey client is nil")
	}
	if key == "" {
		return fmt.Errorf("cache key is required")
	}
	if dest == nil {
		return fmt.Errorf("cache destination is required")
	}

	payload, err := repo.client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}

	return decodeCacheValue(payload, dest)
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
