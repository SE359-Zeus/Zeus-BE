package cache

import (
	"context"
	"log"
	"time"

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
	if err != nil {
		log.Printf("Warning: Valkey Set failed (degraded cache mode): %v", err)
	}
	return nil
}

func (c *ValkeyCache) Delete(ctx context.Context, key string) error {
	_, err := c.withClient(ctx, func(client valkey.Client) ([]byte, error) {
		return nil, client.Do(ctx, client.B().Del().Key(key).Build()).Error()
	})
	if err != nil {
		log.Printf("Warning: Valkey Delete failed (degraded cache mode): %v", err)
	}
	return nil
}

func (c *ValkeyCache) Flush(ctx context.Context) error {
	_, err := c.withClient(ctx, func(client valkey.Client) ([]byte, error) {
		return nil, client.Do(ctx, client.B().Flushall().Build()).Error()
	})
	if err != nil {
		log.Printf("Warning: Valkey Flush failed (degraded cache mode): %v", err)
	}
	return nil
}

func (c *ValkeyCache) Warm(ctx context.Context, entries map[string][]byte) error {
	for key, data := range entries {
		_ = c.Set(ctx, key, data)
	}
	return nil
}

func (c *ValkeyCache) withClient(ctx context.Context, fn func(valkey.Client) ([]byte, error)) ([]byte, error) {
	client, err := valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{c.addr},
	})
	if err != nil {
		log.Printf("Warning: Valkey client creation failed (degraded cache mode): %v", err)
		return nil, nil
	}
	defer client.Close()

	pingCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	if err := client.Do(pingCtx, client.B().Ping().Build()).Error(); err != nil {
		log.Printf("Warning: Valkey ping failed (degraded cache mode): %v", err)
		return nil, nil
	}

	res, err := fn(client)
	if err != nil {
		log.Printf("Warning: Valkey operation failed (degraded cache mode): %v", err)
		return nil, nil
	}
	return res, nil
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
