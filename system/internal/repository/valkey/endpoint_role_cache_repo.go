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
	dialer func() (cache.ValkeyConn, error)
}

func NewEndpointRBACCacheRepository(dialer func() (cache.ValkeyConn, error)) *endpointRBACCacheRepository {
	return &endpointRBACCacheRepository{dialer: dialer}
}

func (r *endpointRBACCacheRepository) Warm(ctx context.Context, endpointLevels map[string]string, roleLevels map[string]string) error {
	conn, err := r.dialer()
	if err != nil {
		return fmt.Errorf("failed to dial Valkey: %w", err)
	}
	defer conn.Close()

	for key, level := range endpointLevels {
		if err := conn.Set(ctx, key, level, 0); err != nil {
			return fmt.Errorf("failed to cache endpoint level %s: %w", key, err)
		}
	}
	for role, level := range roleLevels {
		key := fmt.Sprintf(roleLevelPrefix, role)
		if err := conn.Set(ctx, key, level, 0); err != nil {
			return fmt.Errorf("failed to cache role level %s: %w", role, err)
		}
	}
	return nil
}

func (r *endpointRBACCacheRepository) GetRequiredLevel(ctx context.Context, method, path string) (string, error) {
	conn, err := r.dialer()
	if err != nil {
		return "", fmt.Errorf("failed to dial Valkey: %w", err)
	}
	defer conn.Close()

	key := fmt.Sprintf(endpointLevelPrefix, method, path)
	return conn.Get(ctx, key)
}

func (r *endpointRBACCacheRepository) GetRoleLevel(ctx context.Context, roleName string) (string, error) {
	conn, err := r.dialer()
	if err != nil {
		return "", fmt.Errorf("failed to dial Valkey: %w", err)
	}
	defer conn.Close()

	key := fmt.Sprintf(roleLevelPrefix, roleName)
	return conn.Get(ctx, key)
}
