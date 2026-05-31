package handler

import (
	"context"

	"zeus-system-service/internal/models"
	"zeus-system-service/internal/service"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) Create(ctx context.Context, req models.CreateUserRequest) (*models.User, error) {
	args := m.Called(ctx, req)
	if args.Get(0) != nil {
		return args.Get(0).(*models.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserService) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) != nil {
		return args.Get(0).(*models.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserService) List(ctx context.Context, page, limit int, q string) ([]models.User, *models.PaginationMeta, error) {
	args := m.Called(ctx, page, limit, q)
	var users []models.User
	if v := args.Get(0); v != nil {
		users = v.([]models.User)
	}
	meta, _ := args.Get(1).(*models.PaginationMeta)
	return users, meta, args.Error(2)
}

func (m *MockUserService) Update(ctx context.Context, id uuid.UUID, req models.UpdateUserRequest, currentRole string, currentID uuid.UUID) (*models.User, error) {
	args := m.Called(ctx, id, req, currentRole, currentID)
	if args.Get(0) != nil {
		return args.Get(0).(*models.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserService) SetStatus(ctx context.Context, id uuid.UUID, status models.AccountStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockUserService) Authenticate(ctx context.Context, email, password string) (*models.User, error) {
	args := m.Called(ctx, email, password)
	if args.Get(0) != nil {
		return args.Get(0).(*models.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserService) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) (*models.User, error) {
	args := m.Called(ctx, userID, oldPassword, newPassword)
	if args.Get(0) != nil {
		return args.Get(0).(*models.User), args.Error(1)
	}
	return nil, args.Error(1)
}

type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) Login(ctx context.Context, req models.LoginRequest) (*models.AuthLoginResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) != nil {
		return args.Get(0).(*models.AuthLoginResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAuthService) Refresh(ctx context.Context, req models.RefreshRequest) (*models.AuthLoginResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) != nil {
		return args.Get(0).(*models.AuthLoginResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAuthService) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) (*models.User, error) {
	args := m.Called(ctx, userID, oldPassword, newPassword)
	if args.Get(0) != nil {
		return args.Get(0).(*models.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAuthService) VerifyAccessToken(tokenString string) (*service.JWTClaims, error) {
	args := m.Called(tokenString)
	if args.Get(0) != nil {
		return args.Get(0).(*service.JWTClaims), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAuthService) Logout(ctx context.Context, accessToken string) error {
	args := m.Called(ctx, accessToken)
	return args.Error(0)
}

type MockAuditService struct {
	mock.Mock
}

func (m *MockAuditService) Ingest(ctx context.Context, req models.IngestAuditRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockAuditService) Query(ctx context.Context, filter models.AuditFilter, page, limit int) ([]models.AuditLog, *models.PaginationMeta, error) {
	args := m.Called(ctx, filter, page, limit)
	var logs []models.AuditLog
	if v := args.Get(0); v != nil {
		logs = v.([]models.AuditLog)
	}
	meta, _ := args.Get(1).(*models.PaginationMeta)
	return logs, meta, args.Error(2)
}

func (m *MockAuditService) GetMetrics(ctx context.Context) (*models.AuditMetrics, error) {
	args := m.Called(ctx)
	if args.Get(0) != nil {
		return args.Get(0).(*models.AuditMetrics), args.Error(1)
	}
	return nil, args.Error(1)
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
