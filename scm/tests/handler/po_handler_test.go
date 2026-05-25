package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"zeus-scm-service/internal/handler"
	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupPOTest() (*gin.Engine, *service.MockPOService) {
	gin.SetMode(gin.TestMode)
	mockSvc := new(service.MockPOService)
	h := handler.NewPOHandler(mockSvc)
	r := gin.New()
	v1 := r.Group("/api/v1")
	{
		v1.POST("/purchase-orders/draft", h.CreateDraft)
		v1.POST("/purchase-orders/:poId/line-items", h.AddLineItemWithLock)
		v1.POST("/purchase-orders/:poId/approve", h.ApprovePO)
		v1.PUT("/purchase-orders/:poId/state", h.TransitionState)
	}
	return r, mockSvc
}

func TestPOHandler_CreateDraft_201(t *testing.T) {
	r, mockSvc := setupPOTest()
	vendorID := uuid.New()
	po := &models.PurchaseOrder{
		ID:       "PO-2025-001",
		VendorID: vendorID,
		Status:   models.POStatusDraft,
	}

	mockSvc.On("CreateDraft", mock.Anything, vendorID, "Build-1").Return(po, nil)

	body, _ := json.Marshal(map[string]string{
		"vendor_id":    vendorID.String(),
		"target_build": "Build-1",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/purchase-orders/draft", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp handler.ResponseEnvelope
	json.Unmarshal(w.Body.Bytes(), &resp)
	mockSvc.AssertExpectations(t)
}

func TestPOHandler_CreateDraft_400_InvalidBody(t *testing.T) {
	r, _ := setupPOTest()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/purchase-orders/draft", bytes.NewReader([]byte(`not json`)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPOHandler_CreateDraft_400_InvalidVendorID(t *testing.T) {
	r, _ := setupPOTest()

	body, _ := json.Marshal(map[string]string{
		"vendor_id":    "not-a-uuid",
		"target_build": "Build-1",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/purchase-orders/draft", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPOHandler_CreateDraft_400_MonoVendorViolation(t *testing.T) {
	r, mockSvc := setupPOTest()
	vendorID := uuid.New()

	mockSvc.On("CreateDraft", mock.Anything, vendorID, "Build-1").Return(nil, service.ErrMonoVendorViolation)

	body, _ := json.Marshal(map[string]string{
		"vendor_id":    vendorID.String(),
		"target_build": "Build-1",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/purchase-orders/draft", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestPOHandler_AddLineItemWithLock_200(t *testing.T) {
	r, mockSvc := setupPOTest()

	mockSvc.On("AddLineItemWithLock", mock.Anything, "PO-2025-001", "SOC-001", 10).Return(nil)

	body, _ := json.Marshal(map[string]interface{}{
		"sku": "SOC-001",
		"qty": 10,
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/purchase-orders/PO-2025-001/line-items", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestPOHandler_AddLineItemWithLock_400_InvalidBody(t *testing.T) {
	r, _ := setupPOTest()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/purchase-orders/PO-2025-001/line-items", bytes.NewReader([]byte(`not json`)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPOHandler_ApprovePO_200(t *testing.T) {
	r, mockSvc := setupPOTest()

	mockSvc.On("ApprovePO", mock.Anything, "PO-2025-001").Return(nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/purchase-orders/PO-2025-001/approve", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestPOHandler_ApprovePO_404(t *testing.T) {
	r, mockSvc := setupPOTest()

	mockSvc.On("ApprovePO", mock.Anything, "PO-UNKNOWN").Return(service.ErrNotFound)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/purchase-orders/PO-UNKNOWN/approve", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestPOHandler_TransitionState_200(t *testing.T) {
	r, mockSvc := setupPOTest()

	mockSvc.On("TransitionState", mock.Anything, "PO-2025-001", models.POStatusInTransit).Return(nil)

	body, _ := json.Marshal(map[string]string{
		"new_state": "In Transit",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/purchase-orders/PO-2025-001/state", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestPOHandler_TransitionState_400_InvalidBody(t *testing.T) {
	r, _ := setupPOTest()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/purchase-orders/PO-2025-001/state", bytes.NewReader([]byte(`not json`)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPOHandler_TransitionState_400_StateRegression(t *testing.T) {
	r, mockSvc := setupPOTest()

	mockSvc.On("TransitionState", mock.Anything, "PO-2025-001", models.POStatusDraft).Return(service.ErrStateRegression)

	body, _ := json.Marshal(map[string]string{
		"new_state": "Draft",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/purchase-orders/PO-2025-001/state", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestPOHandler_TransitionState_400_NotFound(t *testing.T) {
	r, mockSvc := setupPOTest()

	mockSvc.On("TransitionState", mock.Anything, "PO-UNKNOWN", models.POStatusApproved).Return(service.ErrNotFound)

	body, _ := json.Marshal(map[string]string{
		"new_state": "Approved",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/purchase-orders/PO-UNKNOWN/state", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}
