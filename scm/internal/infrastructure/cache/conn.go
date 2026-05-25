package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/valkey-io/valkey-go"
)

type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, data []byte) error
	Delete(ctx context.Context, key string) error
	Flush(ctx context.Context) error
	Warm(ctx context.Context, entries map[string][]byte) error
}

type ValkeyCache struct {
	addr string
}

func NewValkey(addr string) (*ValkeyCache, error) {
	client, err := valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{addr},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Valkey client: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Do(ctx, client.B().Ping().Build()).Error(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to connect to Valkey: %w", err)
	}
	client.Close()
	return &ValkeyCache{addr: addr}, nil
}

type NoopCache struct{}

func NewNoop() *NoopCache {
	return &NoopCache{}
}
