package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"zeus-system-service/internal/handler"
	"zeus-system-service/internal/models"
	"zeus-system-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupAuditTest() (*gin.Engine, *handler.MockAuditService) {
	gin.SetMode(gin.TestMode)
	mockSvc := new(handler.MockAuditService)
	h := handler.NewAuditHandler(mockSvc)
	r := gin.New()

	logs := r.Group("/logs")
	{
		logs.POST("/ingest", h.Ingest)
		logs.GET("", h.Query)
		logs.GET("/metrics", h.GetMetrics)
	}

	return r, mockSvc
}

func TestAuditHandler_Ingest_201(t *testing.T) {
	r, mockSvc := setupAuditTest()
	req := models.IngestAuditRequest{
		UserID:         uuid.New(),
		UserEmail:      "user@zeus.com",
		ActionType:     "CREATE",
		TargetResource: "users/abc",
		IPAddress:      "10.0.0.1",
	}
	body, _ := json.Marshal(req)

	mockSvc.On("Ingest", mock.Anything, mock.AnythingOfType("models.IngestAuditRequest")).Return(nil)

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("POST", "/logs/ingest", bytes.NewReader(body))
	reqHTTP.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestAuditHandler_Ingest_400(t *testing.T) {
	r, mockSvc := setupAuditTest()
	req := models.IngestAuditRequest{ActionType: "INVALID", UserID: uuid.New(), UserEmail: "u@z.com", TargetResource: "x"}
	body, _ := json.Marshal(req)

	mockSvc.On("Ingest", mock.Anything, mock.AnythingOfType("models.IngestAuditRequest")).Return(service.ErrInvalidInput)

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("POST", "/logs/ingest", bytes.NewReader(body))
	reqHTTP.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestAuditHandler_Ingest_400_InvalidBody(t *testing.T) {
	r, _ := setupAuditTest()

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("POST", "/logs/ingest", bytes.NewReader([]byte(`not json`)))
	reqHTTP.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuditHandler_Query_200(t *testing.T) {
	r, mockSvc := setupAuditTest()

	logs := []models.AuditLog{
		{
			ID:             uuid.New(),
			Timestamp:      time.Now(),
			ActionType:     "LOGIN",
			TargetResource: "auth/login",
			UserEmail:      "user@zeus.com",
		},
	}
	meta := &models.PaginationMeta{Page: 1, Limit: 15, TotalRows: 1, TotalPages: 1}

	mockSvc.On("Query", mock.Anything, mock.AnythingOfType("models.AuditFilter"), 1, 15).Return(logs, meta, nil)

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("GET", "/logs?action_type=LOGIN", nil)
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusOK, w.Code)
	var env struct {
		StatusCode int             `json:"statusCode"`
		Metadata   json.RawMessage `json:"metadata"`
		Data       json.RawMessage `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &env)
	assert.NoError(t, err)
	assert.Equal(t, 200, env.StatusCode)

	var metadata struct {
		Pagination models.PaginationMeta  `json:"pagination"`
	}
	err = json.Unmarshal(env.Metadata, &metadata)
	assert.NoError(t, err)
	assert.Equal(t, 1, metadata.Pagination.TotalPages)

	var data struct {
		Items []models.AuditLog `json:"items"`
	}
	err = json.Unmarshal(env.Data, &data)
	assert.NoError(t, err)
	assert.Len(t, data.Items, 1)

	mockSvc.AssertExpectations(t)
}

func TestAuditHandler_Query_200_Empty(t *testing.T) {
	r, mockSvc := setupAuditTest()
	meta := &models.PaginationMeta{Page: 1, Limit: 15, TotalRows: 0, TotalPages: 0}

	mockSvc.On("Query", mock.Anything, mock.AnythingOfType("models.AuditFilter"), 1, 15).Return([]models.AuditLog{}, meta, nil)

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("GET", "/logs", nil)
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusOK, w.Code)
	var env struct {
		StatusCode int             `json:"statusCode"`
		Metadata   json.RawMessage `json:"metadata"`
		Data       json.RawMessage `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &env)
	assert.NoError(t, err)
	assert.Equal(t, 200, env.StatusCode)

	var metadata struct {
		Pagination models.PaginationMeta  `json:"pagination"`
	}
	err = json.Unmarshal(env.Metadata, &metadata)
	assert.NoError(t, err)

	var data struct {
		Items []models.AuditLog `json:"items"`
	}
	err = json.Unmarshal(env.Data, &data)
	assert.NoError(t, err)
	assert.Len(t, data.Items, 0)

	mockSvc.AssertExpectations(t)
}

func TestAuditHandler_GetMetrics_200(t *testing.T) {
	r, mockSvc := setupAuditTest()

	metrics := &models.AuditMetrics{
		LoginsToday:          10,
		SecurityEvents:       3,
		ModificationVelocity: 25,
	}

	mockSvc.On("GetMetrics", mock.Anything).Return(metrics, nil)

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("GET", "/logs/metrics", nil)
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusOK, w.Code)
	var env struct {
		StatusCode int             `json:"statusCode"`
		Data       json.RawMessage `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &env)
	assert.NoError(t, err)
	assert.Equal(t, 200, env.StatusCode)
	var data models.AuditMetrics
	err = json.Unmarshal(env.Data, &data)
	assert.NoError(t, err)
	assert.Equal(t, int64(10), data.LoginsToday)
	assert.Equal(t, int64(3), data.SecurityEvents)
	assert.Equal(t, int64(25), data.ModificationVelocity)
	mockSvc.AssertExpectations(t)
}
