package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"zeus-scm-service/internal/handler/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCORS_AllowsAnyOriginAndPreflight(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.CORS())
	r.GET("/ping", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	preflight := httptest.NewRecorder()
	preflightReq, _ := http.NewRequest(http.MethodOptions, "/ping", nil)
	preflightReq.Header.Set("Origin", "http://example.com")
	preflightReq.Header.Set("Access-Control-Request-Method", http.MethodGet)
	preflightReq.Header.Set("Access-Control-Request-Headers", "Authorization, X-API-KEY")
	r.ServeHTTP(preflight, preflightReq)

	assert.Equal(t, http.StatusNoContent, preflight.Code)
	assert.Equal(t, "*", preflight.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "GET,POST,PUT,PATCH,DELETE,OPTIONS", preflight.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "Authorization, X-API-KEY", preflight.Header().Get("Access-Control-Allow-Headers"))

	actual := httptest.NewRecorder()
	actualReq, _ := http.NewRequest(http.MethodGet, "/ping", nil)
	actualReq.Header.Set("Origin", "http://example.com")
	r.ServeHTTP(actual, actualReq)

	assert.Equal(t, http.StatusOK, actual.Code)
	assert.Equal(t, "*", actual.Header().Get("Access-Control-Allow-Origin"))
}
