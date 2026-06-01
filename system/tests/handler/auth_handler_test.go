package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"zeus-system-service/internal/handler"
	"zeus-system-service/internal/models"
	"zeus-system-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupAuthTest() (*gin.Engine, *handler.MockAuthService) {
	gin.SetMode(gin.TestMode)
	mockSvc := new(handler.MockAuthService)
	h := handler.NewAuthHandler(mockSvc)
	r := gin.New()

	auth := r.Group("/auth")
	{
		auth.POST("/login", h.Login)
		auth.POST("/refresh", h.Refresh)
		auth.POST("/change-password", h.ChangePassword)
	}

	return r, mockSvc
}

func setupAuthAuditTest() (*gin.Engine, *handler.MockAuthService, *handler.MockAuditService) {
	gin.SetMode(gin.TestMode)
	authSvc := new(handler.MockAuthService)
	auditSvc := new(handler.MockAuditService)
	h := handler.NewAuthHandler(authSvc, auditSvc)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", uuid.MustParse("11111111-1111-1111-1111-111111111111"))
		c.Set("email", "admin@zeus.com")
		c.Next()
	})

	auth := r.Group("/auth")
	{
		auth.POST("/login", h.Login)
		auth.POST("/refresh", h.Refresh)
		auth.POST("/change-password", h.ChangePassword)
	}

	return r, authSvc, auditSvc
}

func TestAuthHandler_Login_200(t *testing.T) {
	r, mockSvc := setupAuthTest()

	req := models.LoginRequest{Email: "admin@zeus.com", Password: "pass123"}
	userID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	result := &models.AuthLoginResult{
		Tokens: &models.TokenPair{
			AccessToken:  "access-token-value",
			RefreshToken: "refresh-token-value",
			TokenType:    "Bearer",
			ExpiresIn:    900,
		},
		User: &models.User{
			ID:       userID,
			Email:    req.Email,
			FullName: "Admin User",
			Role:     "admin",
			Status:   models.AccountStatusActive,
		},
	}
	body, _ := json.Marshal(req)

	mockSvc.On("Login", mock.Anything, mock.AnythingOfType("models.LoginRequest")).Return(result, nil)

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("POST", "/auth/login", bytes.NewReader(body))
	reqHTTP.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusOK, w.Code)
	// Refresh token must be in Set-Cookie, not in the body.
	assert.Contains(t, w.Header().Get("Set-Cookie"), models.RefreshTokenCookieName)

	var env struct {
		StatusCode int             `json:"statusCode"`
		Data       json.RawMessage `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &env)
	assert.NoError(t, err)
	assert.Equal(t, 200, env.StatusCode)
	var data struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		User        struct {
			Email string `json:"email"`
			Role  string `json:"role"`
		} `json:"user"`
	}
	err = json.Unmarshal(env.Data, &data)
	assert.NoError(t, err)
	assert.Equal(t, result.Tokens.AccessToken, data.AccessToken)
	assert.Equal(t, "Bearer", data.TokenType)
	assert.Equal(t, req.Email, data.User.Email)
	assert.Equal(t, "admin", data.User.Role)
	mockSvc.AssertExpectations(t)
}

func TestAuthHandler_Login_200_WritesAudit(t *testing.T) {
	r, authSvc, auditSvc := setupAuthAuditTest()

	req := models.LoginRequest{Email: "admin@zeus.com", Password: "pass123"}
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	result := &models.AuthLoginResult{
		Tokens: &models.TokenPair{
			AccessToken:  "access-token-value",
			RefreshToken: "refresh-token-value",
			TokenType:    "Bearer",
			ExpiresIn:    900,
		},
		User: &models.User{
			ID:       userID,
			Email:    req.Email,
			FullName: "Admin User",
			Role:     "admin",
			Status:   models.AccountStatusActive,
		},
	}
	claims := &service.JWTClaims{
		UserID:   userID,
		Email:    req.Email,
		FullName: "Admin User",
		Role:     "admin",
		Status:   "ACTIVE",
	}
	body, _ := json.Marshal(req)

	authSvc.On("Login", mock.Anything, mock.AnythingOfType("models.LoginRequest")).Return(result, nil)
	authSvc.On("VerifyAccessToken", result.Tokens.AccessToken).Return(claims, nil)
	auditSvc.On("Ingest", mock.Anything, mock.MatchedBy(func(req models.IngestAuditRequest) bool {
		return req.UserID == claims.UserID &&
			req.UserEmail == claims.Email &&
			req.ActionType == models.ActionType("LOGIN") &&
			req.TargetResource == "auth/login" &&
			req.Details == "Successful login" &&
			!req.IsSecurityEvent
	})).Return(nil)

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("POST", "/auth/login", bytes.NewReader(body))
	reqHTTP.Header.Set("Content-Type", "application/json")
	reqHTTP.Header.Set("X-Forwarded-For", "203.0.113.10")
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Set-Cookie"), models.RefreshTokenCookieName)
	var env struct {
		StatusCode int             `json:"statusCode"`
		Data       json.RawMessage `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &env)
	assert.NoError(t, err)
	assert.Equal(t, 200, env.StatusCode)
	var data struct {
		AccessToken string `json:"access_token"`
	}
	err = json.Unmarshal(env.Data, &data)
	assert.NoError(t, err)
	assert.Equal(t, result.Tokens.AccessToken, data.AccessToken)
	authSvc.AssertExpectations(t)
	auditSvc.AssertExpectations(t)
}

func TestAuthHandler_Login_400_InvalidBody(t *testing.T) {
	r, _ := setupAuthTest()

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("POST", "/auth/login", bytes.NewReader([]byte(`not json`)))
	reqHTTP.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthHandler_Login_401_InvalidCredentials(t *testing.T) {
	r, mockSvc := setupAuthTest()

	req := models.LoginRequest{Email: "admin@zeus.com", Password: "wrong"}
	body, _ := json.Marshal(req)

	mockSvc.On("Login", mock.Anything, mock.AnythingOfType("models.LoginRequest")).Return(nil, service.ErrUnauthorized)

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("POST", "/auth/login", bytes.NewReader(body))
	reqHTTP.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestAuthHandler_Login_401_Inactive(t *testing.T) {
	r, mockSvc := setupAuthTest()

	req := models.LoginRequest{Email: "inactive@zeus.com", Password: "pass"}
	body, _ := json.Marshal(req)

	mockSvc.On("Login", mock.Anything, mock.AnythingOfType("models.LoginRequest")).Return(nil, service.ErrInactiveAccount)

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("POST", "/auth/login", bytes.NewReader(body))
	reqHTTP.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestAuthHandler_Refresh_200(t *testing.T) {
	r, mockSvc := setupAuthTest()

	userID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	// Send refresh token via cookie (the new primary path).
	result := &models.AuthLoginResult{
		Tokens: &models.TokenPair{
			AccessToken:  "new-access-token",
			RefreshToken: "new-refresh-token",
			TokenType:    "Bearer",
			ExpiresIn:    900,
		},
		User: &models.User{
			ID:     userID,
			Email:  "admin@zeus.com",
			Role:   "admin",
			Status: models.AccountStatusActive,
		},
	}

	mockSvc.On("Refresh", mock.Anything, mock.AnythingOfType("models.RefreshRequest")).Return(result, nil)

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("POST", "/auth/refresh", nil)
	reqHTTP.Header.Set("Content-Type", "application/json")
	reqHTTP.AddCookie(&http.Cookie{Name: models.RefreshTokenCookieName, Value: "valid-refresh-token"})
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Set-Cookie"), models.RefreshTokenCookieName)

	var env struct {
		StatusCode int             `json:"statusCode"`
		Data       json.RawMessage `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &env)
	assert.NoError(t, err)
	assert.Equal(t, 200, env.StatusCode)
	var data struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
	err = json.Unmarshal(env.Data, &data)
	assert.NoError(t, err)
	assert.Equal(t, result.Tokens.AccessToken, data.AccessToken)
	assert.Equal(t, "Bearer", data.TokenType)
	mockSvc.AssertExpectations(t)
}

func TestAuthHandler_Refresh_401_MissingCookie(t *testing.T) {
	r, mockSvc := setupAuthTest()

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("POST", "/auth/refresh", nil)
	reqHTTP.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestAuthHandler_Refresh_200_FromBody(t *testing.T) {
	r, mockSvc := setupAuthTest()

	userID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	result := &models.AuthLoginResult{
		Tokens: &models.TokenPair{
			AccessToken:  "new-access-token",
			RefreshToken: "new-refresh-token",
			TokenType:    "Bearer",
			ExpiresIn:    900,
		},
		User: &models.User{
			ID:     userID,
			Email:  "admin@zeus.com",
			Role:   "admin",
			Status: models.AccountStatusActive,
		},
	}

	mockSvc.On("Refresh", mock.Anything, mock.AnythingOfType("models.RefreshRequest")).Return(result, nil)

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("POST", "/auth/refresh", bytes.NewReader([]byte(`{"refresh_token":"valid-refresh-token"}`)))
	reqHTTP.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Set-Cookie"), models.RefreshTokenCookieName)

	var env struct {
		StatusCode int             `json:"statusCode"`
		Data       json.RawMessage `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &env)
	assert.NoError(t, err)
	assert.Equal(t, 200, env.StatusCode)
	mockSvc.AssertExpectations(t)
}

func TestAuthHandler_Refresh_401_InvalidCookie(t *testing.T) {
	r, mockSvc := setupAuthTest()

	mockSvc.On("Refresh", mock.Anything, mock.AnythingOfType("models.RefreshRequest")).Return(nil, service.ErrUnauthorized)

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("POST", "/auth/refresh", nil)
	reqHTTP.Header.Set("Content-Type", "application/json")
	reqHTTP.AddCookie(&http.Cookie{Name: models.RefreshTokenCookieName, Value: "expired-or-invalid"})
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestAuthHandler_ChangePassword_200_WritesAudit(t *testing.T) {
	r, authSvc, auditSvc := setupAuthAuditTest()

	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	oldPassword := "oldpass123"
	newPassword := "newpass123"
	updated := &models.User{
		ID:       userID,
		Email:    "admin@zeus.com",
		FullName: "Admin User",
		Role:     "admin",
		Status:   models.AccountStatusActive,
	}
	req := models.ChangePasswordRequest{OldPassword: oldPassword, NewPassword: newPassword}
	body, _ := json.Marshal(req)

	authSvc.On("ChangePassword", mock.Anything, userID, oldPassword, newPassword).Return(updated, nil)
	auditSvc.On("Ingest", mock.Anything, mock.MatchedBy(func(req models.IngestAuditRequest) bool {
		return req.UserID == userID &&
			req.UserEmail == "admin@zeus.com" &&
			req.ActionType == models.ActionType("UPDATE") &&
			req.TargetResource == "users/"+userID.String()+"/password" &&
			req.Details == "Changed password" &&
			!req.IsSecurityEvent
	})).Return(nil)

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("POST", "/auth/change-password", bytes.NewReader(body))
	reqHTTP.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusOK, w.Code)
	var env struct {
		StatusCode int             `json:"statusCode"`
		Data       json.RawMessage `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &env)
	assert.NoError(t, err)
	assert.Equal(t, 200, env.StatusCode)
	authSvc.AssertExpectations(t)
	auditSvc.AssertExpectations(t)
}

func TestAuthHandler_ChangePassword_400_InvalidBody(t *testing.T) {
	r, _ := setupAuthTest()

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("POST", "/auth/change-password", bytes.NewReader([]byte(`not json`)))
	reqHTTP.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthHandler_Refresh_400(t *testing.T) {
	r, _ := setupAuthTest()

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("POST", "/auth/refresh", bytes.NewReader([]byte(`not json`)))
	reqHTTP.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
