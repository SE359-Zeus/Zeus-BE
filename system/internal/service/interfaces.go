package service

import (
	"context"

	"zeus-system-service/internal/models"

	"github.com/google/uuid"
)

type UserService interface {
	Create(ctx context.Context, req models.CreateUserRequest) (*models.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	List(ctx context.Context, page, limit int, q string) ([]models.User, *models.PaginationMeta, error)
	Update(ctx context.Context, id uuid.UUID, req models.UpdateUserRequest) (*models.User, error)
	SetStatus(ctx context.Context, id uuid.UUID, status models.AccountStatus) error
	Authenticate(ctx context.Context, email, password string) (*models.User, error)
}

type AuthService interface {
	Login(ctx context.Context, req models.LoginRequest) (*models.TokenPair, error)
	Refresh(ctx context.Context, req models.RefreshRequest) (*models.TokenPair, error)
	VerifyAccessToken(tokenString string) (*JWTClaims, error)
	Logout(ctx context.Context, accessToken string) error
}

type JWTClaims struct {
	UserID   uuid.UUID `json:"user_id"`
	Role     string    `json:"role"`
	Email    string    `json:"email"`
	FullName string    `json:"full_name"`
	Status   string    `json:"status"`
}

type AuditService interface {
	Ingest(ctx context.Context, req models.IngestAuditRequest) error
	Query(ctx context.Context, filter models.AuditFilter, page, limit int) ([]models.AuditLog, *models.PaginationMeta, error)
	GetMetrics(ctx context.Context) (*models.AuditMetrics, error)
}

type EndpointRBACService interface {
	ValidateRole(ctx context.Context, role string) error
	WarmCache(ctx context.Context) error
}

type ActionTypeService interface {
	IsValid(ctx context.Context, name models.ActionType) (bool, error)
	WarmCache(ctx context.Context) error
}
