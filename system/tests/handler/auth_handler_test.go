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
	}

	return r, mockSvc
}

func setupAuthAuditTest() (*gin.Engine, *handler.MockAuthService, *handler.MockAuditService) {
	gin.SetMode(gin.TestMode)
	authSvc := new(handler.MockAuthService)
	auditSvc := new(handler.MockAuditService)
	h := handler.NewAuthHandler(authSvc, auditSvc)
	r := gin.New()

	auth := r.Group("/auth")
	{
		auth.POST("/login", h.Login)
		auth.POST("/refresh", h.Refresh)
	}

	return r, authSvc, auditSvc
}

func TestAuthHandler_Login_200(t *testing.T) {
	r, mockSvc := setupAuthTest()

	req := models.LoginRequest{Email: "admin@zeus.com", Password: "pass123"}
	pair := &models.TokenPair{
		AccessToken:  "access-token-value",
		RefreshToken: "refresh-token-value",
		TokenType:    "Bearer",
		ExpiresIn:    900,
	}
	body, _ := json.Marshal(req)

	mockSvc.On("Login", mock.Anything, mock.AnythingOfType("models.LoginRequest")).Return(pair, nil)

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("POST", "/auth/login", bytes.NewReader(body))
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
	var data struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	err = json.Unmarshal(env.Data, &data)
	assert.NoError(t, err)
	assert.Equal(t, pair.AccessToken, data.AccessToken)
	assert.Equal(t, pair.RefreshToken, data.RefreshToken)
	mockSvc.AssertExpectations(t)
}

func TestAuthHandler_Login_200_WritesAudit(t *testing.T) {
	r, authSvc, auditSvc := setupAuthAuditTest()

	req := models.LoginRequest{Email: "admin@zeus.com", Password: "pass123"}
	pair := &models.TokenPair{
		AccessToken:  "access-token-value",
		RefreshToken: "refresh-token-value",
		TokenType:    "Bearer",
		ExpiresIn:    900,
	}
	claims := &service.JWTClaims{
		UserID:   uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		Email:    req.Email,
		FullName: "Admin User",
		Role:     "admin",
		Status:   "ACTIVE",
	}
	body, _ := json.Marshal(req)

	authSvc.On("Login", mock.Anything, mock.AnythingOfType("models.LoginRequest")).Return(pair, nil)
	authSvc.On("VerifyAccessToken", pair.AccessToken).Return(claims, nil)
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
	var env struct {
		StatusCode int             `json:"statusCode"`
		Data       json.RawMessage `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &env)
	assert.NoError(t, err)
	assert.Equal(t, 200, env.StatusCode)
	var data struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	err = json.Unmarshal(env.Data, &data)
	assert.NoError(t, err)
	assert.Equal(t, pair.AccessToken, data.AccessToken)
	assert.Equal(t, pair.RefreshToken, data.RefreshToken)
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

	req := models.RefreshRequest{RefreshToken: "valid-refresh-token"}
	pair := &models.TokenPair{
		AccessToken:  "new-access-token",
		RefreshToken: "new-refresh-token",
		TokenType:    "Bearer",
		ExpiresIn:    900,
	}
	body, _ := json.Marshal(req)

	mockSvc.On("Refresh", mock.Anything, mock.AnythingOfType("models.RefreshRequest")).Return(pair, nil)

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("POST", "/auth/refresh", bytes.NewReader(body))
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
	var data models.TokenPair
	err = json.Unmarshal(env.Data, &data)
	assert.NoError(t, err)
	mockSvc.AssertExpectations(t)
}

func TestAuthHandler_Refresh_401(t *testing.T) {
	r, mockSvc := setupAuthTest()

	req := models.RefreshRequest{RefreshToken: "expired-or-invalid"}
	body, _ := json.Marshal(req)

	mockSvc.On("Refresh", mock.Anything, mock.AnythingOfType("models.RefreshRequest")).Return(nil, service.ErrUnauthorized)

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("POST", "/auth/refresh", bytes.NewReader(body))
	reqHTTP.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestAuthHandler_Refresh_400(t *testing.T) {
	r, _ := setupAuthTest()

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("POST", "/auth/refresh", bytes.NewReader([]byte(`not json`)))
	reqHTTP.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
