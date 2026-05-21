package repository

import (
	"context"
	"time"

	"zeus-system-service/internal/models"

	"github.com/google/uuid"
)

type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	List(ctx context.Context, page, limit int, q string) ([]models.User, int64, error)
	Update(ctx context.Context, user *models.User) error
	SetStatus(ctx context.Context, id uuid.UUID, status models.AccountStatus) error
}

type RefreshTokenRepository interface {
	SaveRefreshToken(ctx context.Context, jti, userID string) error
	ValidateRefreshToken(ctx context.Context, jti string) (string, error)
	DeleteUserTokens(ctx context.Context, userID string) error
	DeleteExpired(ctx context.Context) error
	BlacklistAccessToken(ctx context.Context, jti string, ttl time.Duration) error
	IsAccessTokenBlacklisted(ctx context.Context, jti string) (bool, error)
}

type AuditRepository interface {
	Insert(ctx context.Context, log *models.AuditLog) error
	Query(ctx context.Context, filter models.AuditFilter, page, limit int) ([]models.AuditLog, int64, error)
	CountByAction(ctx context.Context, actionType models.ActionType, start, end time.Time) (int64, error)
}

type RoleRepository interface {
	GetAll(ctx context.Context) ([]models.Role, error)
	GetByName(ctx context.Context, name string) (*models.Role, error)
	Exists(ctx context.Context, name string) (bool, error)
}

type ActionTypeRepository interface {
	GetAll(ctx context.Context) ([]models.ActionTypeEntry, error)
	Exists(ctx context.Context, name string) (bool, error)
}

type EndpointRoleRepository interface {
	GetRequiredLevel(ctx context.Context, method, path string) (string, error)
	GetAll(ctx context.Context) ([]models.EndpointRole, error)
}

type ActionTypeCacheRepository interface {
	Warm(ctx context.Context, names []string) error
	IsValid(ctx context.Context, name string) (bool, error)
}

type EndpointRBACCacheRepository interface {
	Warm(ctx context.Context, endpointLevels map[string]string, roleLevels map[string]string) error
	GetRequiredLevel(ctx context.Context, method, path string) (string, error)
	GetRoleLevel(ctx context.Context, roleName string) (string, error)
}
