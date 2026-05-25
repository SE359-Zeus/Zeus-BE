package repository

import (
	"context"
	"time"

	"zeus-system-service/internal/models"

	"github.com/google/uuid"
)

type RefreshTokenRepository interface {
	SaveRefreshToken(ctx context.Context, jti, userID string) error
	ValidateRefreshToken(ctx context.Context, jti string) (string, error)
	DeleteUserTokens(ctx context.Context, userID string) error
	DeleteExpired(ctx context.Context) error
	BlacklistAccessToken(ctx context.Context, jti string, ttl time.Duration) error
	IsAccessTokenBlacklisted(ctx context.Context, jti string) (bool, error)
}

type ActionTypeCacheRepository interface {
	Warm(ctx context.Context, names []string) error
	IsValid(ctx context.Context, name string) (bool, error)
}

type UserCacheRepository interface {
	Set(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	Delete(ctx context.Context, id uuid.UUID, email string) error
}
