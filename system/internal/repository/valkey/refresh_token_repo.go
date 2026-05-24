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
	dialer func() (cache.ValkeyConn, error)
}

func NewRefreshTokenRepository(dialer func() (cache.ValkeyConn, error)) repository.RefreshTokenRepository {
	return &refreshTokenRepo{dialer: dialer}
}

func (r *refreshTokenRepo) SaveRefreshToken(ctx context.Context, jti, userID string) error {
	conn, err := r.dialer()
	if err != nil {
		return fmt.Errorf("failed to dial Valkey: %w", err)
	}
	defer conn.Close()

	if err := conn.Set(ctx, "refresh_token:"+jti, userID, 7*24*time.Hour); err != nil {
		return fmt.Errorf("failed to save refresh token: %w", err)
	}
	if err := conn.SAdd(ctx, "user:"+userID+":refresh_tokens", jti); err != nil {
		return fmt.Errorf("failed to index refresh token: %w", err)
	}
	return nil
}

func (r *refreshTokenRepo) ValidateRefreshToken(ctx context.Context, jti string) (string, error) {
	conn, err := r.dialer()
	if err != nil {
		return "", fmt.Errorf("failed to dial Valkey: %w", err)
	}
	defer conn.Close()

	userID, err := conn.Get(ctx, "refresh_token:"+jti)
	if err != nil {
		return "", fmt.Errorf("failed to validate refresh token: %w", err)
	}
	if userID == "" {
		return "", valkey.Nil
	}
	return userID, nil
}

func (r *refreshTokenRepo) DeleteUserTokens(ctx context.Context, userID string) error {
	conn, err := r.dialer()
	if err != nil {
		return fmt.Errorf("failed to dial Valkey: %w", err)
	}
	defer conn.Close()

	jtis, err := conn.SMembers(ctx, "user:"+userID+":refresh_tokens")
	if err != nil {
		return fmt.Errorf("failed to list user tokens: %w", err)
	}
	for _, jti := range jtis {
		if err := conn.Del(ctx, "refresh_token:"+jti); err != nil {
			return fmt.Errorf("failed to delete token %s: %w", jti, err)
		}
	}
	if err := conn.Del(ctx, "user:"+userID+":refresh_tokens"); err != nil {
		return fmt.Errorf("failed to delete user token index: %w", err)
	}
	return nil
}

func (r *refreshTokenRepo) DeleteExpired(ctx context.Context) error {
	return nil
}

func (r *refreshTokenRepo) BlacklistAccessToken(ctx context.Context, jti string, ttl time.Duration) error {
	conn, err := r.dialer()
	if err != nil {
		return fmt.Errorf("failed to dial Valkey: %w", err)
	}
	defer conn.Close()

	return conn.Set(ctx, "access_token_blacklist:"+jti, "1", ttl)
}

func (r *refreshTokenRepo) IsAccessTokenBlacklisted(ctx context.Context, jti string) (bool, error) {
	conn, err := r.dialer()
	if err != nil {
		return false, fmt.Errorf("failed to dial Valkey: %w", err)
	}
	defer conn.Close()

	return conn.Exists(ctx, "access_token_blacklist:"+jti)
}
