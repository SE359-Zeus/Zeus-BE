package cache

import "context"

type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, data []byte) error
	Delete(ctx context.Context, key string) error
	Flush(ctx context.Context) error
	Warm(ctx context.Context, entries map[string][]byte) error
}
