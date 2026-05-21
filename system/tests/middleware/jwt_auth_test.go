package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"zeus-system-service/internal/handler"
	"zeus-system-service/internal/handler/middleware"
	"zeus-system-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func setupMiddlewareTest() (*gin.Engine, *handler.MockAuthService) {
	gin.SetMode(gin.TestMode)
	mockSvc := new(handler.MockAuthService)
	r := gin.New()

	r.Use(middleware.JWTAuth(mockSvc))
	r.GET("/protected", func(c *gin.Context) {
		userID, _ := c.Get(middleware.ContextKeyUserID)
		role, _ := c.Get(middleware.ContextKeyRole)
		email, _ := c.Get(middleware.ContextKeyEmail)
		c.JSON(http.StatusOK, gin.H{
			"user_id": userID,
			"role":    role,
			"email":   email,
		})
	})

	return r, mockSvc
}

func TestJWTAuth_ValidToken(t *testing.T) {
	r, mockSvc := setupMiddlewareTest()
	userID := uuid.New()

	mockSvc.On("VerifyAccessToken", "valid-token").Return(&service.JWTClaims{
		UserID:   userID,
		Role:     "admin",
		Email:    "admin@zeus.com",
		FullName: "System Administrator",
		Status:   "ACTIVE",
	}, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestJWTAuth_MissingHeader(t *testing.T) {
	r, _ := setupMiddlewareTest()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuth_InvalidFormat(t *testing.T) {
	r, _ := setupMiddlewareTest()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "InvalidFormat")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuth_ExpiredToken(t *testing.T) {
	r, mockSvc := setupMiddlewareTest()

	mockSvc.On("VerifyAccessToken", "expired-token").Return(nil, service.ErrUnauthorized)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer expired-token")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	mockSvc.AssertExpectations(t)
}
