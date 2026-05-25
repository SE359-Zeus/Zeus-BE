package valkey

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"zeus-system-service/internal/infrastructure/cache"
	"zeus-system-service/internal/models"

	"github.com/google/uuid"
)

type cachedUser struct {
	ID           uuid.UUID            `json:"id"`
	Email        string               `json:"email"`
	PasswordHash string               `json:"password_hash"`
	FullName     string               `json:"full_name"`
	Role         string               `json:"role"`
	Status       models.AccountStatus `json:"status"`
	LastLoginAt  *time.Time           `json:"last_login_at,omitempty"`
	CreatedAt    time.Time            `json:"created_at"`
	UpdatedAt    time.Time            `json:"updated_at"`
}

type userCacheRepository struct {
	dialer func() (cache.ValkeyConn, error)
	ttl    time.Duration
}

func NewUserCacheRepository(dialer func() (cache.ValkeyConn, error)) *userCacheRepository {
	return &userCacheRepository{dialer: dialer, ttl: 24 * time.Hour}
}

func (r *userCacheRepository) Set(ctx context.Context, user *models.User) error {
	if user == nil {
		return nil
	}
	conn, err := r.dialer()
	if err != nil {
		return fmt.Errorf("failed to dial Valkey: %w", err)
	}
	defer conn.Close()

	cu := cachedUser{
		ID:           user.ID,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		FullName:     user.FullName,
		Role:         user.Role,
		Status:       user.Status,
		LastLoginAt:  user.LastLoginAt,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}

	data, err := json.Marshal(cu)
	if err != nil {
		return err
	}

	idKey := "user:id:" + user.ID.String()
	emailKey := "user:email:" + user.Email

	if err := conn.Set(ctx, idKey, string(data), r.ttl); err != nil {
		return err
	}
	return conn.Set(ctx, emailKey, string(data), r.ttl)
}

func (r *userCacheRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	conn, err := r.dialer()
	if err != nil {
		return nil, fmt.Errorf("failed to dial Valkey: %w", err)
	}
	defer conn.Close()

	key := "user:id:" + id.String()
	val, err := conn.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if val == "" {
		return nil, nil
	}

	var cu cachedUser
	if err := json.Unmarshal([]byte(val), &cu); err != nil {
		return nil, err
	}

	return &models.User{
		ID:           cu.ID,
		Email:        cu.Email,
		PasswordHash: cu.PasswordHash,
		FullName:     cu.FullName,
		Role:         cu.Role,
		Status:       cu.Status,
		LastLoginAt:  cu.LastLoginAt,
		CreatedAt:    cu.CreatedAt,
		UpdatedAt:    cu.UpdatedAt,
	}, nil
}

func (r *userCacheRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	conn, err := r.dialer()
	if err != nil {
		return nil, fmt.Errorf("failed to dial Valkey: %w", err)
	}
	defer conn.Close()

	key := "user:email:" + email
	val, err := conn.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if val == "" {
		return nil, nil
	}

	var cu cachedUser
	if err := json.Unmarshal([]byte(val), &cu); err != nil {
		return nil, err
	}

	return &models.User{
		ID:           cu.ID,
		Email:        cu.Email,
		PasswordHash: cu.PasswordHash,
		FullName:     cu.FullName,
		Role:         cu.Role,
		Status:       cu.Status,
		LastLoginAt:  cu.LastLoginAt,
		CreatedAt:    cu.CreatedAt,
		UpdatedAt:    cu.UpdatedAt,
	}, nil
}

func (r *userCacheRepository) Delete(ctx context.Context, id uuid.UUID, email string) error {
	conn, err := r.dialer()
	if err != nil {
		return fmt.Errorf("failed to dial Valkey: %w", err)
	}
	defer conn.Close()

	idKey := "user:id:" + id.String()
	emailKey := "user:email:" + email

	return conn.Del(ctx, idKey, emailKey)
}
