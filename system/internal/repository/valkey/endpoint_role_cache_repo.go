package valkey

import (
	"context"
	"fmt"

	"zeus-system-service/internal/cache"
)

const (
	endpointLevelPrefix = "rbac:endpoint:%s:%s"
	roleLevelPrefix     = "rbac:role:%s:level"
)

type endpointRBACCacheRepository struct {
	vk *cache.Valkey
}

func NewEndpointRBACCacheRepository(vk *cache.Valkey) *endpointRBACCacheRepository {
	return &endpointRBACCacheRepository{vk: vk}
}

func (r *endpointRBACCacheRepository) Warm(ctx context.Context, endpointLevels map[string]string, roleLevels map[string]string) error {
	for key, level := range endpointLevels {
		if err := r.vk.Set(ctx, key, level, 0); err != nil {
			return fmt.Errorf("failed to cache endpoint level %s: %w", key, err)
		}
	}
	for role, level := range roleLevels {
		key := fmt.Sprintf(roleLevelPrefix, role)
		if err := r.vk.Set(ctx, key, level, 0); err != nil {
			return fmt.Errorf("failed to cache role level %s: %w", role, err)
		}
	}
	return nil
}

func (r *endpointRBACCacheRepository) GetRequiredLevel(ctx context.Context, method, path string) (string, error) {
	key := fmt.Sprintf(endpointLevelPrefix, method, path)
	return r.vk.Get(ctx, key)
}

func (r *endpointRBACCacheRepository) GetRoleLevel(ctx context.Context, roleName string) (string, error) {
	key := fmt.Sprintf(roleLevelPrefix, roleName)
	return r.vk.Get(ctx, key)
}
