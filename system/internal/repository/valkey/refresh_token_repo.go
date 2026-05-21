package valkey

import (
	"context"
	"fmt"
	"time"

	"zeus-system-service/internal/cache"
	"zeus-system-service/internal/repository"

	"github.com/valkey-io/valkey-go"
)

type refreshTokenRepo struct {
	vk *cache.Valkey
}

func NewRefreshTokenRepository(vk *cache.Valkey) repository.RefreshTokenRepository {
	return &refreshTokenRepo{vk: vk}
}

func (r *refreshTokenRepo) SaveRefreshToken(ctx context.Context, jti, userID string) error {
	if err := r.vk.Set(ctx, "refresh_token:"+jti, userID, 7*24*time.Hour); err != nil {
		return fmt.Errorf("failed to save refresh token: %w", err)
	}
	if err := r.vk.SAdd(ctx, "user:"+userID+":refresh_tokens", jti); err != nil {
		return fmt.Errorf("failed to index refresh token: %w", err)
	}
	return nil
}

func (r *refreshTokenRepo) ValidateRefreshToken(ctx context.Context, jti string) (string, error) {
	userID, err := r.vk.Get(ctx, "refresh_token:"+jti)
	if err != nil {
		return "", fmt.Errorf("failed to validate refresh token: %w", err)
	}
	if userID == "" {
		return "", valkey.Nil
	}
	return userID, nil
}

func (r *refreshTokenRepo) DeleteUserTokens(ctx context.Context, userID string) error {
	jtis, err := r.vk.SMembers(ctx, "user:"+userID+":refresh_tokens")
	if err != nil {
		return fmt.Errorf("failed to list user tokens: %w", err)
	}
	for _, jti := range jtis {
		if err := r.vk.Del(ctx, "refresh_token:"+jti); err != nil {
			return fmt.Errorf("failed to delete token %s: %w", jti, err)
		}
	}
	if err := r.vk.Del(ctx, "user:"+userID+":refresh_tokens"); err != nil {
		return fmt.Errorf("failed to delete user token index: %w", err)
	}
	return nil
}

func (r *refreshTokenRepo) DeleteExpired(ctx context.Context) error {
	return nil
}

func (r *refreshTokenRepo) BlacklistAccessToken(ctx context.Context, jti string, ttl time.Duration) error {
	return r.vk.Set(ctx, "access_token_blacklist:"+jti, "1", ttl)
}

func (r *refreshTokenRepo) IsAccessTokenBlacklisted(ctx context.Context, jti string) (bool, error) {
	return r.vk.Exists(ctx, "access_token_blacklist:"+jti)
}
