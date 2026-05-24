package cache

import "context"

type NoopCache struct{}

func NewNoop() *NoopCache {
	return &NoopCache{}
}

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
