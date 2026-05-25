package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
	"zeus-scm-service/internal/handler/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type MockAuditPublisher struct {
	mu        sync.Mutex
	published []any
}

func (m *MockAuditPublisher) PublishToAudit(msg any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.published = append(m.published, msg)
	return nil
}

func (m *MockAuditPublisher) GetPublished() []any {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := make([]any, len(m.published))
	copy(copied, m.published)
	return copied
}

func setupTestRouter(pub middleware.AuditPublisher, method string, status int, setupCtx func(*gin.Context)) (*httptest.ResponseRecorder, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.Use(func(c *gin.Context) {
		if setupCtx != nil {
			setupCtx(c)
		}
		c.Next()
	})

	r.Use(middleware.Audit(pub))

	r.Handle(method, "/test-resource", func(c *gin.Context) {
		c.Status(status)
	})

	return httptest.NewRecorder(), r
}

func TestAuditMiddleware_POST_Success(t *testing.T) {
	pub := &MockAuditPublisher{}
	userID := uuid.New()
	email := "operator@zeus.com"

	w, r := setupTestRouter(pub, "POST", http.StatusCreated, func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Set("email", email)
	})

	req, _ := http.NewRequest("POST", "/test-resource?foo=bar", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	// Wait briefly for asynchronous publishing goroutine
	time.Sleep(50 * time.Millisecond)

	published := pub.GetPublished()
	assert.Len(t, published, 1)

	msg, ok := published[0].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, userID, msg["user_id"])
	assert.Equal(t, email, msg["user_email"])
	assert.Equal(t, "CREATE", msg["action_type"])
	assert.Equal(t, "/test-resource", msg["target_resource"])
	assert.Equal(t, "Created resource at /test-resource", msg["details"])
}

func TestAuditMiddleware_PUT_Success(t *testing.T) {
	pub := &MockAuditPublisher{}
	userID := uuid.New()

	w, r := setupTestRouter(pub, "PUT", http.StatusOK, func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Set("auth_method", "api_key")
		c.Set("api_key_name", "test-key")
	})

	req, _ := http.NewRequest("PUT", "/test-resource", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	time.Sleep(50 * time.Millisecond)

	published := pub.GetPublished()
	assert.Len(t, published, 1)

	msg := published[0].(map[string]any)
	assert.Equal(t, userID, msg["user_id"])
	assert.Equal(t, "api_key:test-key", msg["user_email"])
	assert.Equal(t, "UPDATE", msg["action_type"])
	assert.Equal(t, "Updated resource at /test-resource", msg["details"])
}

func TestAuditMiddleware_DELETE_Success(t *testing.T) {
	pub := &MockAuditPublisher{}
	userID := uuid.New()

	w, r := setupTestRouter(pub, "DELETE", http.StatusNoContent, func(c *gin.Context) {
		c.Set("user_id", userID.String()) // string UUID format
	})

	req, _ := http.NewRequest("DELETE", "/test-resource", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	time.Sleep(50 * time.Millisecond)

	published := pub.GetPublished()
	assert.Len(t, published, 1)

	msg := published[0].(map[string]any)
	assert.Equal(t, userID, msg["user_id"])
	assert.Equal(t, "system@zeus.local", msg["user_email"])
	assert.Equal(t, "DELETE", msg["action_type"])
	assert.Equal(t, "Deleted resource at /test-resource", msg["details"])
}

func TestAuditMiddleware_GET_Ignored(t *testing.T) {
	pub := &MockAuditPublisher{}

	w, r := setupTestRouter(pub, "GET", http.StatusOK, nil)
	req, _ := http.NewRequest("GET", "/test-resource", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	time.Sleep(50 * time.Millisecond)

	assert.Empty(t, pub.GetPublished())
}

func TestAuditMiddleware_Error_Ignored(t *testing.T) {
	pub := &MockAuditPublisher{}

	w, r := setupTestRouter(pub, "POST", http.StatusBadRequest, nil)
	req, _ := http.NewRequest("POST", "/test-resource", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	time.Sleep(50 * time.Millisecond)

	assert.Empty(t, pub.GetPublished())
}

func TestAuditMiddleware_NilPublisher_Graceful(t *testing.T) {
	w, r := setupTestRouter(nil, "POST", http.StatusOK, nil)
	req, _ := http.NewRequest("POST", "/test-resource", nil)

	assert.NotPanics(t, func() {
		r.ServeHTTP(w, req)
	})
	assert.Equal(t, http.StatusOK, w.Code)
}
