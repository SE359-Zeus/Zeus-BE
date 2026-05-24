package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/valkey-io/valkey-go"
)

type ValkeyConn interface {
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Del(ctx context.Context, keys ...string) error
	SAdd(ctx context.Context, key, member string) error
	SMembers(ctx context.Context, key string) ([]string, error)
	Exists(ctx context.Context, key string) (bool, error)
	HSet(ctx context.Context, key, field, value string) error
	HGetAll(ctx context.Context, key string) (map[string]string, error)
	SIsMember(ctx context.Context, key, member string) (bool, error)
	Close()
}

func DialValkey(addr string) (ValkeyConn, error) {
	client, err := valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{addr},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Valkey client: %w", err)
	}
	return &valkeyConn{client: client}, nil
}

type valkeyConn struct {
	client valkey.Client
}
