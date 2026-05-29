package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"zeus-scm-service/internal/handler"
	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/pagination"
	"zeus-scm-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupGRTest() (*gin.Engine, *service.MockGoodsReceiptService) {
	gin.SetMode(gin.TestMode)
	mockSvc := new(service.MockGoodsReceiptService)
	h := handler.NewGoodsReceiptHandler(mockSvc)
	r := gin.New()
	v1 := r.Group("/api/v1")
	{
		v1.POST("/goods-receipts/:grId/lock", h.AcquireLock)
		v1.POST("/goods-receipts/:grId/process", h.ProcessBlindReceipt)
		v1.DELETE("/goods-receipts/:grId/lock", h.ReleaseLock)
		v1.GET("/goods-receipts", h.ListGRs)
		v1.GET("/goods-receipts/metrics", h.GetMetrics)
		v1.GET("/goods-receipts/:grId", h.GetGR)
	}
	return r, mockSvc
}

func TestGoodsReceiptHandler_AcquireLock_200(t *testing.T) {
	r, mockSvc := setupGRTest()

	mockSvc.On("AcquireLock", mock.Anything, "GR-2025-001", "operator-1").Return(nil)

	body, _ := json.Marshal(map[string]string{"operator_id": "operator-1"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/goods-receipts/GR-2025-001/lock", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestGoodsReceiptHandler_AcquireLock_400_InvalidBody(t *testing.T) {
	r, _ := setupGRTest()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/goods-receipts/GR-2025-001/lock", bytes.NewReader([]byte(`not json`)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGoodsReceiptHandler_AcquireLock_404(t *testing.T) {
	r, mockSvc := setupGRTest()

	mockSvc.On("AcquireLock", mock.Anything, "GR-UNKNOWN", "operator-1").Return(service.ErrNotFound)

	body, _ := json.Marshal(map[string]string{"operator_id": "operator-1"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/goods-receipts/GR-UNKNOWN/lock", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestGoodsReceiptHandler_AcquireLock_409_AlreadyLocked(t *testing.T) {
	r, mockSvc := setupGRTest()

	mockSvc.On("AcquireLock", mock.Anything, "GR-2025-001", "operator-1").Return(service.ErrAlreadyLocked)

	body, _ := json.Marshal(map[string]string{"operator_id": "operator-1"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/goods-receipts/GR-2025-001/lock", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestGoodsReceiptHandler_ProcessBlindReceipt_200(t *testing.T) {
	r, mockSvc := setupGRTest()

	mockSvc.On("ProcessBlindReceipt", mock.Anything, "GR-2025-001", "operator-1",
		mock.MatchedBy(func(c map[string]struct {
			Received  int
			Defective int
		}) bool { return c["SOC-001"].Received == 10 })).Return(nil)

	body, _ := json.Marshal(map[string]interface{}{
		"operator_id": "operator-1",
		"counts": map[string]map[string]int{
			"SOC-001": {"received": 10, "defective": 1},
		},
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/goods-receipts/GR-2025-001/process", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestGoodsReceiptHandler_ProcessBlindReceipt_400_InvalidBody(t *testing.T) {
	r, _ := setupGRTest()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/goods-receipts/GR-2025-001/process", bytes.NewReader([]byte(`not json`)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGoodsReceiptHandler_ProcessBlindReceipt_404(t *testing.T) {
	r, mockSvc := setupGRTest()

	mockSvc.On("ProcessBlindReceipt", mock.Anything, "GR-UNKNOWN", "operator-1",
		mock.AnythingOfType("map[string]struct { Received int; Defective int }")).
		Return(service.ErrNotFound)

	body, _ := json.Marshal(map[string]interface{}{
		"operator_id": "operator-1",
		"counts":      map[string]map[string]int{},
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/goods-receipts/GR-UNKNOWN/process", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestGoodsReceiptHandler_ReleaseLock_200(t *testing.T) {
	r, mockSvc := setupGRTest()

	mockSvc.On("ReleaseLock", mock.Anything, "GR-2025-001").Return(nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/goods-receipts/GR-2025-001/lock", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestGoodsReceiptHandler_ReleaseLock_404(t *testing.T) {
	r, mockSvc := setupGRTest()

	mockSvc.On("ReleaseLock", mock.Anything, "GR-UNKNOWN").Return(service.ErrNotFound)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/goods-receipts/GR-UNKNOWN/lock", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestGoodsReceiptHandler_ListGRs_200(t *testing.T) {
	r, mockSvc := setupGRTest()

	grs := []models.GoodsReceipt{
		{ID: "GR-1", Status: models.GRStatusPending},
	}
	meta := &pagination.Meta{TotalRows: 1, Page: 1, Limit: 15, TotalPages: 1}
	mockSvc.On("ListGRs", mock.Anything, "Pending", mock.Anything).Return(grs, meta, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/goods-receipts?status=Pending", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	
	var resp struct {
		StatusCode int                  `json:"statusCode"`
		Data       []models.GoodsReceipt `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Len(t, resp.Data, 1)
	assert.Equal(t, "GR-1", resp.Data[0].ID)
	mockSvc.AssertExpectations(t)
}

func TestGoodsReceiptHandler_GetGR_200(t *testing.T) {
	r, mockSvc := setupGRTest()

	gr := &models.GoodsReceipt{ID: "GR-1", Status: models.GRStatusPending}
	mockSvc.On("GetGR", mock.Anything, "GR-1").Return(gr, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/goods-receipts/GR-1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	
	var resp struct {
		StatusCode int                 `json:"statusCode"`
		Data       models.GoodsReceipt `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "GR-1", resp.Data.ID)
	mockSvc.AssertExpectations(t)
}

func TestGoodsReceiptHandler_GetGR_404(t *testing.T) {
	r, mockSvc := setupGRTest()

	mockSvc.On("GetGR", mock.Anything, "GR-UNKNOWN").Return(nil, service.ErrNotFound)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/goods-receipts/GR-UNKNOWN", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestGoodsReceiptHandler_GetMetrics_200(t *testing.T) {
	r, mockSvc := setupGRTest()

	mockSvc.On("GetMetrics", mock.Anything).Return(int64(5), int64(10), int64(2), int64(1), nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/goods-receipts/metrics", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		StatusCode int `json:"statusCode"`
		Data       map[string]int64 `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), resp.Data["pending_receipts"])
	assert.Equal(t, int64(10), resp.Data["completed_today"])
	assert.Equal(t, int64(2), resp.Data["active_discrepancies"])
	assert.Equal(t, int64(1), resp.Data["inspection_queue"])
	mockSvc.AssertExpectations(t)
}
