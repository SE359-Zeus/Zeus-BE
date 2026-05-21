package valkey

import (
	"context"
	"fmt"

	"zeus-system-service/internal/cache"
)

const (
	actionTypeSetKey = "action_types:valid"
)

type actionTypeCacheRepository struct {
	vk *cache.Valkey
}

func NewActionTypeCacheRepository(vk *cache.Valkey) *actionTypeCacheRepository {
	return &actionTypeCacheRepository{vk: vk}
}

func (r *actionTypeCacheRepository) Warm(ctx context.Context, names []string) error {
	for _, name := range names {
		if err := r.vk.SAdd(ctx, actionTypeSetKey, name); err != nil {
			return fmt.Errorf("failed to cache action type %s: %w", name, err)
		}
	}
	return nil
}

func (r *actionTypeCacheRepository) IsValid(ctx context.Context, name string) (bool, error) {
	return r.vk.SIsMember(ctx, actionTypeSetKey, name)
}
