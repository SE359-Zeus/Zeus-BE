package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/valkey-io/valkey-go"
)

type ValkeyCache struct {
	client valkey.Client
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
		return nil, fmt.Errorf("failed to connect to Valkey: %w", err)
	}
	return &ValkeyCache{client: client}, nil
}

func (c *ValkeyCache) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := c.client.Do(ctx, c.client.B().Get().Key(key).Build()).ToString()
	if err != nil {
		if err == valkey.Nil {
			return nil, nil
		}
		return nil, err
	}
	return []byte(val), nil
}

func (c *ValkeyCache) Set(ctx context.Context, key string, data []byte) error {
	return c.client.Do(ctx, c.client.B().Set().Key(key).Value(string(data)).Build()).Error()
}

func (c *ValkeyCache) Delete(ctx context.Context, key string) error {
	return c.client.Do(ctx, c.client.B().Del().Key(key).Build()).Error()
}

func (c *ValkeyCache) Flush(ctx context.Context) error {
	return c.client.Do(ctx, c.client.B().Flushall().Build()).Error()
}

func (c *ValkeyCache) Warm(ctx context.Context, entries map[string][]byte) error {
	for key, data := range entries {
		if err := c.Set(ctx, key, data); err != nil {
			return err
		}
	}
	return nil
}
