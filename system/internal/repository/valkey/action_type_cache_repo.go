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
	dialer func() (cache.ValkeyConn, error)
}

func NewActionTypeCacheRepository(dialer func() (cache.ValkeyConn, error)) *actionTypeCacheRepository {
	return &actionTypeCacheRepository{dialer: dialer}
}

func (r *actionTypeCacheRepository) Warm(ctx context.Context, names []string) error {
	conn, err := r.dialer()
	if err != nil {
		return fmt.Errorf("failed to dial Valkey: %w", err)
	}
	defer conn.Close()

	for _, name := range names {
		if err := conn.SAdd(ctx, actionTypeSetKey, name); err != nil {
			return fmt.Errorf("failed to cache action type %s: %w", name, err)
		}
	}
	return nil
}

func (r *actionTypeCacheRepository) IsValid(ctx context.Context, name string) (bool, error) {
	conn, err := r.dialer()
	if err != nil {
		return false, fmt.Errorf("failed to dial Valkey: %w", err)
	}
	defer conn.Close()

	return conn.SIsMember(ctx, actionTypeSetKey, name)
}
