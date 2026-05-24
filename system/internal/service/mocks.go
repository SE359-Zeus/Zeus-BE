package service

import (
	"context"
	"time"

	"zeus-system-service/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) != nil {
		return args.Get(0).(*models.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) != nil {
		return args.Get(0).(*models.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserRepository) List(ctx context.Context, page, limit int, q string) ([]models.User, int64, error) {
	args := m.Called(ctx, page, limit, q)
	var users []models.User
	if v := args.Get(0); v != nil {
		users = v.([]models.User)
	}
	return users, args.Get(1).(int64), args.Error(2)
}

func (m *MockUserRepository) Update(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) SetStatus(ctx context.Context, id uuid.UUID, status models.AccountStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

type MockRefreshTokenRepository struct {
	mock.Mock
}

func (m *MockRefreshTokenRepository) SaveRefreshToken(ctx context.Context, jti, userID string) error {
	args := m.Called(ctx, jti, userID)
	return args.Error(0)
}

func (m *MockRefreshTokenRepository) ValidateRefreshToken(ctx context.Context, jti string) (string, error) {
	args := m.Called(ctx, jti)
	return args.String(0), args.Error(1)
}

func (m *MockRefreshTokenRepository) DeleteUserTokens(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockRefreshTokenRepository) DeleteExpired(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockRefreshTokenRepository) BlacklistAccessToken(ctx context.Context, jti string, ttl time.Duration) error {
	args := m.Called(ctx, jti, ttl)
	return args.Error(0)
}

func (m *MockRefreshTokenRepository) IsAccessTokenBlacklisted(ctx context.Context, jti string) (bool, error) {
	args := m.Called(ctx, jti)
	return args.Bool(0), args.Error(1)
}

type MockAuditRepository struct {
	mock.Mock
}

func (m *MockAuditRepository) Insert(ctx context.Context, log *models.AuditLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func (m *MockAuditRepository) Query(ctx context.Context, filter models.AuditFilter, page, limit int) ([]models.AuditLog, int64, error) {
	args := m.Called(ctx, filter, page, limit)
	var logs []models.AuditLog
	if v := args.Get(0); v != nil {
		logs = v.([]models.AuditLog)
	}
	return logs, args.Get(1).(int64), args.Error(2)
}

func (m *MockAuditRepository) CountByAction(ctx context.Context, actionType models.ActionType, start, end time.Time) (int64, error) {
	args := m.Called(ctx, actionType, start, end)
	return args.Get(0).(int64), args.Error(1)
}

type MockEndpointRBACService struct {
	mock.Mock
}

func (m *MockEndpointRBACService) ValidateRole(ctx context.Context, role string) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *MockEndpointRBACService) WarmCache(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

type MockSessionRepository struct {
	mock.Mock
}

func (m *MockSessionRepository) Create(ctx context.Context, session *models.Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockSessionRepository) GetByJTI(ctx context.Context, jti string) (*models.Session, error) {
	args := m.Called(ctx, jti)
	if args.Get(0) != nil {
		return args.Get(0).(*models.Session), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockSessionRepository) DeleteByJTI(ctx context.Context, jti string) error {
	args := m.Called(ctx, jti)
	return args.Error(0)
}

func (m *MockSessionRepository) DeleteByUserID(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockSessionRepository) DeleteExpired(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockSessionRepository) ListActive(ctx context.Context) ([]models.Session, error) {
	args := m.Called(ctx)
	if args.Get(0) != nil {
		return args.Get(0).([]models.Session), args.Error(1)
	}
	return nil, args.Error(1)
}

type MockActionTypeService struct {
	mock.Mock
}

func (m *MockActionTypeService) IsValid(ctx context.Context, name models.ActionType) (bool, error) {
	args := m.Called(ctx, name)
	return args.Bool(0), args.Error(1)
}

func (m *MockActionTypeService) WarmCache(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

type MockEmailService struct {
	mock.Mock
}

func (m *MockEmailService) SendTemplate(ctx context.Context, req EmailTemplateRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

type MockUserCacheRepository struct {
	mock.Mock
}

func (m *MockUserCacheRepository) Set(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserCacheRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) != nil {
		return args.Get(0).(*models.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserCacheRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) != nil {
		return args.Get(0).(*models.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserCacheRepository) Delete(ctx context.Context, id uuid.UUID, email string) error {
	args := m.Called(ctx, id, email)
	return args.Error(0)
}

