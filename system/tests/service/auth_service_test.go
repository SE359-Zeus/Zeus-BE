package service_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"
	"time"

	"zeus-system-service/internal/models"
	"zeus-system-service/internal/service"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

type jwtAccessClaims struct {
	UserID   string `json:"user_id"`
	Role     string `json:"role"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Status   string `json:"status"`
	jwt.RegisteredClaims
}

type jwtRefreshClaims struct {
	JTI uuid.UUID `json:"jti"`
	SUB uuid.UUID `json:"sub"`
	jwt.RegisteredClaims
}

type mockRefreshRepo struct {
	mock.Mock
}

func (m *mockRefreshRepo) SaveRefreshToken(ctx context.Context, jti, userID string) error {
	args := m.Called(ctx, jti, userID)
	return args.Error(0)
}

func (m *mockRefreshRepo) ValidateRefreshToken(ctx context.Context, jti string) (string, error) {
	args := m.Called(ctx, jti)
	return args.String(0), args.Error(1)
}

func (m *mockRefreshRepo) DeleteUserTokens(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *mockRefreshRepo) DeleteExpired(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockRefreshRepo) BlacklistAccessToken(ctx context.Context, jti string, ttl time.Duration) error {
	args := m.Called(ctx, jti, ttl)
	return args.Error(0)
}

func (m *mockRefreshRepo) IsAccessTokenBlacklisted(ctx context.Context, jti string) (bool, error) {
	args := m.Called(ctx, jti)
	return args.Bool(0), args.Error(1)
}

func generateTestKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NoError(t, err)
	return key
}

func assertTokenPair(t *testing.T, pair *models.TokenPair) {
	t.Helper()
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
	assert.Equal(t, "Bearer", pair.TokenType)
	assert.Equal(t, int64(900), pair.ExpiresIn)
}

func setupAuthSvc(t *testing.T) (service.AuthService, *service.MockUserRepository, *mockRefreshRepo, *service.MockSessionRepository) {
	t.Helper()
	userRepo := new(service.MockUserRepository)
	refreshRepo := new(mockRefreshRepo)
	sessionRepo := new(service.MockSessionRepository)
	key := generateTestKey(t)

	rbacSvc := new(service.MockEndpointRBACService)
	rbacSvc.On("ValidateRole", anyCtx, mock.AnythingOfType("string")).Return(nil)

	userSvc := service.NewUserService(userRepo, rbacSvc, nil, nil)
	svc := service.NewAuthService(userSvc, refreshRepo, sessionRepo, key)

	return svc, userRepo, refreshRepo, sessionRepo
}

func hashPassword(t *testing.T, password string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	assert.NoError(t, err)
	return string(hash)
}

func TestAuthService_Login_Success(t *testing.T) {
	svc, userRepo, refreshRepo, sessionRepo := setupAuthSvc(t)
	email := "admin@zeus.com"
	password := "securepass123"
	userID := uuid.New()

	userRepo.On("GetByEmail", anyCtx, email).Return(&models.User{
		ID:           userID,
		Email:        email,
		PasswordHash: hashPassword(t, password),
		FullName:     "Admin User",
		Role:         "admin",
		Status:       models.AccountStatusActive,
	}, nil)
	refreshRepo.On("SaveRefreshToken", anyCtx, mock.AnythingOfType("string"), userID.String()).Return(nil)
	refreshRepo.On("IsAccessTokenBlacklisted", anyCtx, mock.AnythingOfType("string")).Return(false, nil)
	sessionRepo.On("Create", anyCtx, mock.AnythingOfType("*models.Session")).Return(nil)

	pair, err := svc.Login(context.Background(), models.LoginRequest{Email: email, Password: password})
	assert.NoError(t, err)
	assertTokenPair(t, pair)

	claims, err := svc.VerifyAccessToken(pair.AccessToken)
	assert.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, "admin", claims.Role)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, "Admin User", claims.FullName)
	assert.Equal(t, "ACTIVE", claims.Status)

	userRepo.AssertExpectations(t)
	refreshRepo.AssertExpectations(t)
	sessionRepo.AssertExpectations(t)
}

func TestAuthService_Login_ContinuesWhenRefreshCacheUnavailable(t *testing.T) {
	svc, userRepo, refreshRepo, sessionRepo := setupAuthSvc(t)
	email := "admin@zeus.com"
	password := "securepass123"
	userID := uuid.New()

	userRepo.On("GetByEmail", anyCtx, email).Return(&models.User{
		ID:           userID,
		Email:        email,
		PasswordHash: hashPassword(t, password),
		FullName:     "Admin User",
		Role:         "admin",
		Status:       models.AccountStatusActive,
	}, nil)
	refreshRepo.On("SaveRefreshToken", anyCtx, mock.AnythingOfType("string"), userID.String()).Return(errors.New("cache unavailable"))
	sessionRepo.On("Create", anyCtx, mock.AnythingOfType("*models.Session")).Return(nil)

	pair, err := svc.Login(context.Background(), models.LoginRequest{Email: email, Password: password})
	assert.NoError(t, err)
	assertTokenPair(t, pair)

	userRepo.AssertExpectations(t)
	refreshRepo.AssertExpectations(t)
	sessionRepo.AssertExpectations(t)
}

func TestAuthService_Login_InactiveUser(t *testing.T) {
	svc, userRepo, _, _ := setupAuthSvc(t)
	email := "inactive@zeus.com"

	userRepo.On("GetByEmail", anyCtx, email).Return(&models.User{
		Email:  email,
		Status: models.AccountStatusInactive,
	}, nil)

	pair, err := svc.Login(context.Background(), models.LoginRequest{Email: email, Password: "anypass"})
	assert.ErrorIs(t, err, service.ErrInactiveAccount)
	assert.Nil(t, pair)
	userRepo.AssertExpectations(t)
}

func TestAuthService_Login_InvalidCredentials(t *testing.T) {
	svc, userRepo, _, _ := setupAuthSvc(t)
	email := "admin@zeus.com"

	userRepo.On("GetByEmail", anyCtx, email).Return(&models.User{
		Email:        email,
		PasswordHash: hashPassword(t, "correctpass"),
		Role:         "admin",
		Status:       models.AccountStatusActive,
	}, nil)

	pair, err := svc.Login(context.Background(), models.LoginRequest{Email: email, Password: "wrongpass"})
	assert.ErrorIs(t, err, service.ErrUnauthorized)
	assert.Nil(t, pair)
	userRepo.AssertExpectations(t)
}

func TestAuthService_Refresh_Success(t *testing.T) {
	svc, userRepo, refreshRepo, sessionRepo := setupAuthSvc(t)
	userID := uuid.New()

	userRepo.On("GetByEmail", anyCtx, "admin@zeus.com").Return(&models.User{
		ID:           userID,
		Email:        "admin@zeus.com",
		PasswordHash: hashPassword(t, "pass"),
		Role:         "admin",
		Status:       models.AccountStatusActive,
	}, nil)
	refreshRepo.On("SaveRefreshToken", anyCtx, mock.AnythingOfType("string"), userID.String()).Return(nil)
	sessionRepo.On("Create", anyCtx, mock.AnythingOfType("*models.Session")).Return(nil)

	loginPair, err := svc.Login(context.Background(), models.LoginRequest{Email: "admin@zeus.com", Password: "pass"})
	assert.NoError(t, err)

	refreshClaims := &jwtRefreshClaims{}
	_, _, err = jwt.NewParser().ParseUnverified(loginPair.RefreshToken, refreshClaims)
	assert.NoError(t, err)

	refreshRepo.On("ValidateRefreshToken", anyCtx, refreshClaims.JTI.String()).Return(userID.String(), nil)
	userRepo.On("GetByID", anyCtx, userID).Return(&models.User{
		ID:    userID,
		Email: "admin@zeus.com",
		Role:  "admin",
	}, nil)
	refreshRepo.On("SaveRefreshToken", anyCtx, mock.AnythingOfType("string"), userID.String()).Return(nil)
	sessionRepo.On("Create", anyCtx, mock.AnythingOfType("*models.Session")).Return(nil)
	sessionRepo.On("DeleteByJTI", anyCtx, mock.AnythingOfType("string")).Return(nil)

	pair, err := svc.Refresh(context.Background(), models.RefreshRequest{RefreshToken: loginPair.RefreshToken})
	assert.NoError(t, err)
	assertTokenPair(t, pair)

	userRepo.AssertExpectations(t)
	refreshRepo.AssertExpectations(t)
	sessionRepo.AssertExpectations(t)
}

func TestAuthService_Refresh_FallsBackToSessionStoreWhenCacheUnavailable(t *testing.T) {
	svc, userRepo, refreshRepo, sessionRepo := setupAuthSvc(t)
	userID := uuid.New()

	userRepo.On("GetByEmail", anyCtx, "admin@zeus.com").Return(&models.User{
		ID:           userID,
		Email:        "admin@zeus.com",
		PasswordHash: hashPassword(t, "pass"),
		Role:         "admin",
		Status:       models.AccountStatusActive,
	}, nil)
	refreshRepo.On("SaveRefreshToken", anyCtx, mock.AnythingOfType("string"), userID.String()).Return(errors.New("cache unavailable"))
	sessionRepo.On("Create", anyCtx, mock.AnythingOfType("*models.Session")).Return(nil)

	loginPair, err := svc.Login(context.Background(), models.LoginRequest{Email: "admin@zeus.com", Password: "pass"})
	assert.NoError(t, err)

	refreshClaims := &jwtRefreshClaims{}
	_, _, err = jwt.NewParser().ParseUnverified(loginPair.RefreshToken, refreshClaims)
	assert.NoError(t, err)

	refreshRepo.On("ValidateRefreshToken", anyCtx, refreshClaims.JTI.String()).Return("", errors.New("cache unavailable"))
	sessionRepo.On("GetByJTI", anyCtx, refreshClaims.JTI.String()).Return(&models.Session{UserID: userID, JTI: refreshClaims.JTI.String()}, nil)
	userRepo.On("GetByID", anyCtx, userID).Return(&models.User{
		ID:    userID,
		Email: "admin@zeus.com",
		Role:  "admin",
	}, nil)
	refreshRepo.On("SaveRefreshToken", anyCtx, mock.AnythingOfType("string"), userID.String()).Return(errors.New("cache unavailable"))
	sessionRepo.On("Create", anyCtx, mock.AnythingOfType("*models.Session")).Return(nil)
	sessionRepo.On("DeleteByJTI", anyCtx, refreshClaims.JTI.String()).Return(nil)

	pair, err := svc.Refresh(context.Background(), models.RefreshRequest{RefreshToken: loginPair.RefreshToken})
	assert.NoError(t, err)
	assertTokenPair(t, pair)

	userRepo.AssertExpectations(t)
	refreshRepo.AssertExpectations(t)
	sessionRepo.AssertExpectations(t)
}

func TestAuthService_Refresh_ExpiredToken(t *testing.T) {
	svc, userRepo, refreshRepo, sessionRepo := setupAuthSvc(t)
	userID := uuid.New()

	userRepo.On("GetByEmail", anyCtx, "admin@zeus.com").Return(&models.User{
		ID:           userID,
		Email:        "admin@zeus.com",
		PasswordHash: hashPassword(t, "pass"),
		Role:         "admin",
		Status:       models.AccountStatusActive,
	}, nil)
	refreshRepo.On("SaveRefreshToken", anyCtx, mock.AnythingOfType("string"), userID.String()).Return(nil)
	sessionRepo.On("Create", anyCtx, mock.AnythingOfType("*models.Session")).Return(nil)

	loginPair, err := svc.Login(context.Background(), models.LoginRequest{Email: "admin@zeus.com", Password: "pass"})
	assert.NoError(t, err)

	refreshClaims := &jwtRefreshClaims{}
	_, _, err = jwt.NewParser().ParseUnverified(loginPair.RefreshToken, refreshClaims)
	assert.NoError(t, err)

	refreshRepo.On("ValidateRefreshToken", anyCtx, refreshClaims.JTI.String()).Return("", nil)
	sessionRepo.On("GetByJTI", anyCtx, refreshClaims.JTI.String()).Return(nil, nil)

	pair, err := svc.Refresh(context.Background(), models.RefreshRequest{RefreshToken: loginPair.RefreshToken})
	assert.Error(t, err)
	assert.Nil(t, pair)

	userRepo.AssertExpectations(t)
	refreshRepo.AssertExpectations(t)
	sessionRepo.AssertExpectations(t)
}

func TestAuthService_Refresh_InvalidToken(t *testing.T) {
	svc, _, _, _ := setupAuthSvc(t)

	pair, err := svc.Refresh(context.Background(), models.RefreshRequest{RefreshToken: "not-a-valid-jwt"})
	assert.Error(t, err)
	assert.Nil(t, pair)
}

func TestAuthService_VerifyAccessToken_Success(t *testing.T) {
	svc, userRepo, refreshRepo, sessionRepo := setupAuthSvc(t)
	userID := uuid.New()

	userRepo.On("GetByEmail", anyCtx, "v@z.com").Return(&models.User{
		ID:           userID,
		Email:        "v@z.com",
		PasswordHash: hashPassword(t, "pass"),
		Role:         "admin",
		Status:       models.AccountStatusActive,
	}, nil)
	refreshRepo.On("SaveRefreshToken", anyCtx, mock.AnythingOfType("string"), userID.String()).Return(nil)
	refreshRepo.On("IsAccessTokenBlacklisted", anyCtx, mock.AnythingOfType("string")).Return(false, nil)
	sessionRepo.On("Create", anyCtx, mock.AnythingOfType("*models.Session")).Return(nil)

	pair, err := svc.Login(context.Background(), models.LoginRequest{Email: "v@z.com", Password: "pass"})
	assert.NoError(t, err)

	claimsResult, err := svc.VerifyAccessToken(pair.AccessToken)
	assert.NoError(t, err)
	assert.Equal(t, userID, claimsResult.UserID)
	assert.Equal(t, "admin", claimsResult.Role)
	assert.Equal(t, "v@z.com", claimsResult.Email)
	assert.Equal(t, "", claimsResult.FullName)
	assert.Equal(t, "ACTIVE", claimsResult.Status)
	userRepo.AssertExpectations(t)
	refreshRepo.AssertExpectations(t)
	sessionRepo.AssertExpectations(t)
}

func TestAuthService_VerifyAccessToken_Expired(t *testing.T) {
	svc, _, _, _ := setupAuthSvc(t)

	_, err := svc.VerifyAccessToken("eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjEwMDAwMDAwMDAsInVzZXJfaWQiOiIxIn0.signature")
	assert.Error(t, err)
}

func TestAuthService_Logout_Success(t *testing.T) {
	svc, userRepo, refreshRepo, sessionRepo := setupAuthSvc(t)
	userID := uuid.New()
	email := "user@zeus.com"
	password := "pass123"

	userRepo.On("GetByEmail", anyCtx, email).Return(&models.User{
		ID:           userID,
		Email:        email,
		PasswordHash: hashPassword(t, password),
		Role:         "admin",
		Status:       models.AccountStatusActive,
	}, nil)
	refreshRepo.On("SaveRefreshToken", anyCtx, mock.AnythingOfType("string"), userID.String()).Return(nil)
	sessionRepo.On("Create", anyCtx, mock.AnythingOfType("*models.Session")).Return(nil)

	pair, err := svc.Login(context.Background(), models.LoginRequest{Email: email, Password: password})
	assert.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)

	refreshRepo.On("DeleteUserTokens", anyCtx, userID.String()).Return(nil)
	sessionRepo.On("DeleteByUserID", anyCtx, userID.String()).Return(nil)
	refreshRepo.On("BlacklistAccessToken", anyCtx, mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(nil)

	err = svc.Logout(context.Background(), pair.AccessToken)
	assert.NoError(t, err)

	userRepo.AssertExpectations(t)
	refreshRepo.AssertExpectations(t)
	sessionRepo.AssertExpectations(t)
}

func TestAuthService_VerifyAccessToken_WrongKey(t *testing.T) {
	otherKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	claims := &jwtAccessClaims{
		UserID: uuid.New().String(),
		Role:   "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(otherKey)
	assert.NoError(t, err)

	svc, _, _, _ := setupAuthSvc(t)
	_, err = svc.VerifyAccessToken(tokenString)
	assert.Error(t, err)
}
