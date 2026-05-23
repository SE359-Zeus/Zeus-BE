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
	addr string
	ttl  time.Duration
}

func New(addr string) *Repository {
	return &Repository{
		addr: addr,
		ttl:  5 * time.Minute,
	}
}

func (repo *Repository) newClient() (*redis.Client, error) {
	if repo == nil {
		return nil, errors.New("valkey repository is nil")
	}
	if repo.addr == "" {
		return nil, errors.New("valkey address is required")
	}

	client := redis.NewClient(&redis.Options{Addr: repo.addr})
	return client, nil
}

func (repo *Repository) Set(ctx context.Context, key string, value interface{}) error {
	if key == "" {
		return fmt.Errorf("cache key is required")
	}

	payload, err := encodeCacheValue(value)
	if err != nil {
		return err
	}

	client, err := repo.newClient()
	if err != nil {
		return err
	}
	defer client.Close()

	if err := client.Ping(ctx).Err(); err != nil {
		return err
	}

	return client.Set(ctx, key, payload, repo.ttl).Err()
}

func (repo *Repository) Get(ctx context.Context, key string, dest interface{}) error {
	if key == "" {
		return fmt.Errorf("cache key is required")
	}
	if dest == nil {
		return fmt.Errorf("cache destination is required")
	}

	client, err := repo.newClient()
	if err != nil {
		return err
	}
	defer client.Close()

	if err := client.Ping(ctx).Err(); err != nil {
		return err
	}

	payload, err := client.Get(ctx, key).Bytes()
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
