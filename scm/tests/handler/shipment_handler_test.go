package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"zeus-scm-service/internal/handler"
	"zeus-scm-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupShipmentTest() (*gin.Engine, *service.MockShipmentService) {
	gin.SetMode(gin.TestMode)
	mockSvc := new(service.MockShipmentService)
	h := handler.NewShipmentHandler(mockSvc)
	r := gin.New()
	v1 := r.Group("/api/v1")
	{
		v1.POST("/shipments/:shipmentId/lock", h.AcquireDispatchLock)
		v1.POST("/shipments/:shipmentId/dispatch", h.DispatchShipment)
	}
	return r, mockSvc
}

func TestShipmentHandler_AcquireDispatchLock_200(t *testing.T) {
	r, mockSvc := setupShipmentTest()

	mockSvc.On("AcquireDispatchLock", mock.Anything, "SH-2025-001", "operator-1").Return(nil)

	body, _ := json.Marshal(map[string]string{"operator_id": "operator-1"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/shipments/SH-2025-001/lock", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestShipmentHandler_AcquireDispatchLock_400_InvalidBody(t *testing.T) {
	r, _ := setupShipmentTest()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/shipments/SH-2025-001/lock", bytes.NewReader([]byte(`not json`)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestShipmentHandler_AcquireDispatchLock_404(t *testing.T) {
	r, mockSvc := setupShipmentTest()

	mockSvc.On("AcquireDispatchLock", mock.Anything, "SH-UNKNOWN", "operator-1").Return(service.ErrNotFound)

	body, _ := json.Marshal(map[string]string{"operator_id": "operator-1"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/shipments/SH-UNKNOWN/lock", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestShipmentHandler_DispatchShipment_200(t *testing.T) {
	r, mockSvc := setupShipmentTest()

	mockSvc.On("DispatchShipment", mock.Anything, "SH-2025-001", "operator-1").Return(nil)

	body, _ := json.Marshal(map[string]string{
		"operator_id": "operator-1",
		"carrier":     "DHL",
		"tracking_no": "TRACK-001",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/shipments/SH-2025-001/dispatch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestShipmentHandler_DispatchShipment_400_InvalidBody(t *testing.T) {
	r, _ := setupShipmentTest()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/shipments/SH-2025-001/dispatch", bytes.NewReader([]byte(`not json`)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestShipmentHandler_DispatchShipment_404(t *testing.T) {
	r, mockSvc := setupShipmentTest()

	mockSvc.On("DispatchShipment", mock.Anything, "SH-UNKNOWN", "operator-1").Return(service.ErrNotFound)

	body, _ := json.Marshal(map[string]string{"operator_id": "operator-1"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/shipments/SH-UNKNOWN/dispatch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestShipmentHandler_DispatchShipment_400_InvalidTransition(t *testing.T) {
	r, mockSvc := setupShipmentTest()

	mockSvc.On("DispatchShipment", mock.Anything, "SH-2025-001", "operator-1").Return(service.ErrInvalidTransition)

	body, _ := json.Marshal(map[string]string{"operator_id": "operator-1"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/shipments/SH-2025-001/dispatch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockSvc.AssertExpectations(t)
}
