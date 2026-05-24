package cache

import (
	"context"
	"fmt"

	"github.com/valkey-io/valkey-go"
)

func (c *ValkeyCache) Get(ctx context.Context, key string) ([]byte, error) {
	return c.withClient(ctx, func(client valkey.Client) ([]byte, error) {
		val, err := client.Do(ctx, client.B().Get().Key(key).Build()).ToString()
		if err != nil {
			if err == valkey.Nil {
				return nil, nil
			}
			return nil, err
		}
		return []byte(val), nil
	})
}

func (c *ValkeyCache) Set(ctx context.Context, key string, data []byte) error {
	_, err := c.withClient(ctx, func(client valkey.Client) ([]byte, error) {
		return nil, client.Do(ctx, client.B().Set().Key(key).Value(string(data)).Build()).Error()
	})
	return err
}

func (c *ValkeyCache) Delete(ctx context.Context, key string) error {
	_, err := c.withClient(ctx, func(client valkey.Client) ([]byte, error) {
		return nil, client.Do(ctx, client.B().Del().Key(key).Build()).Error()
	})
	return err
}

func (c *ValkeyCache) Flush(ctx context.Context) error {
	_, err := c.withClient(ctx, func(client valkey.Client) ([]byte, error) {
		return nil, client.Do(ctx, client.B().Flushall().Build()).Error()
	})
	return err
}

func (c *ValkeyCache) Warm(ctx context.Context, entries map[string][]byte) error {
	for key, data := range entries {
		if err := c.Set(ctx, key, data); err != nil {
			return err
		}
	}
	return nil
}

func (c *ValkeyCache) withClient(ctx context.Context, fn func(valkey.Client) ([]byte, error)) ([]byte, error) {
	client, err := valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{c.addr},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Valkey client: %w", err)
	}
	defer client.Close()
	return fn(client)
}

// NoopCache operations
func (c *NoopCache) Get(_ context.Context, _ string) ([]byte, error) {
	return nil, nil
}

func (c *NoopCache) Set(_ context.Context, _ string, _ []byte) error {
	return nil
}

func (c *NoopCache) Delete(_ context.Context, _ string) error {
	return nil
}

func (c *NoopCache) Flush(_ context.Context) error {
	return nil
}

func (c *NoopCache) Warm(_ context.Context, _ map[string][]byte) error {
	return nil
}
