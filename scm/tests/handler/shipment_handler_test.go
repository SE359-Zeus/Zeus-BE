package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"zeus-scm-service/internal/handler"
	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/pagination"
	"zeus-scm-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
		v1.DELETE("/shipments/:shipmentId/lock", h.ReleaseDispatchLock)
		v1.POST("/shipments/:shipmentId/dispatch", h.DispatchShipment)
		v1.GET("/shipments", h.ListShipments)
		v1.GET("/shipments/metrics", h.GetMetrics)
		v1.GET("/shipments/carriers", h.ListCarriers)
		v1.GET("/shipments/:shipmentId", h.GetShipment)
		v1.POST("/shipments", h.CreateShipment)
	}
	return r, mockSvc
}

func TestShipmentHandler_AcquireDispatchLock_200(t *testing.T) {
	r, mockSvc := setupShipmentTest()

	expiresAt := time.Now().Add(30 * time.Minute)
	mockSvc.On("AcquireDispatchLock", mock.Anything, "SH-2025-001", "operator-1").Return(&expiresAt, nil)

	body, _ := json.Marshal(map[string]string{"operator_id": "operator-1"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/shipments/SH-2025-001/lock", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Message string `json:"message"`
		Data    struct {
			Message     string     `json:"message"`
			ShipmentID  string     `json:"shipment_id"`
			LockedBy    string     `json:"locked_by"`
			LockExpires *time.Time `json:"lock_expires_at"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "dispatch lock acquired", resp.Data.Message)
	assert.Equal(t, "SH-2025-001", resp.Data.ShipmentID)
	assert.NotNil(t, resp.Data.LockExpires)
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

	mockSvc.On("AcquireDispatchLock", mock.Anything, "SH-UNKNOWN", "operator-1").Return(nil, service.ErrNotFound)

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

func TestShipmentHandler_ListShipments_200(t *testing.T) {
	r, mockSvc := setupShipmentTest()

	params := pagination.Params{Page: 1, Limit: 15, Sort: "created_at", Order: "desc"}
	shipments := []models.Shipment{
		{ID: "SHP-2026-001", PORef: "PO-2026-001", Status: models.ShipmentStatusScheduled, SupplierName: "DHL Logistics"},
	}
	meta := &pagination.Meta{Page: 1, Limit: 15, TotalRows: 1, TotalPages: 1}
	mockSvc.On("ListShipments", mock.Anything, "", params).Return(shipments, meta, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/shipments?page=1&limit=15", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data []struct {
			ID           string `json:"id"`
			SupplierName string `json:"supplier_name"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "SHP-2026-001", resp.Data[0].ID)
	assert.Equal(t, "DHL Logistics", resp.Data[0].SupplierName)
	mockSvc.AssertExpectations(t)
}

func TestShipmentHandler_ListShipments_WithStatusFilter(t *testing.T) {
	r, mockSvc := setupShipmentTest()

	params := pagination.Params{Page: 1, Limit: 15, Sort: "created_at", Order: "desc"}
	shipments := []models.Shipment{
		{ID: "SHP-2026-001", PORef: "PO-2026-001", Status: models.ShipmentStatusInTransit, SupplierName: "DHL Logistics"},
	}
	meta := &pagination.Meta{Page: 1, Limit: 15, TotalRows: 1, TotalPages: 1}
	mockSvc.On("ListShipments", mock.Anything, "In Transit", params).Return(shipments, meta, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/shipments?page=1&limit=15&status=In+Transit", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data []struct {
			ID           string `json:"id"`
			SupplierName string `json:"supplier_name"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "DHL Logistics", resp.Data[0].SupplierName)
	mockSvc.AssertExpectations(t)
}

func TestShipmentHandler_GetShipment_200(t *testing.T) {
	r, mockSvc := setupShipmentTest()

	shipmentID := "SHP-2026-001"
	supplierID := uuid.New()
	shipment := &models.Shipment{
		ID:           shipmentID,
		PORef:        "PO-2026-001",
		SupplierID:   supplierID,
		SupplierName: "DHL Express",
		Status:       models.ShipmentStatusScheduled,
		Carrier:      "DHL Express",
		TrackingNo:   "TRACK-001",
		Origin:       "Hillsboro, OR",
		ShipDate:     time.Now(),
		Items: []models.ShipmentItem{
			{SKU: "SKU-001", Description: "Laptop Module", Qty: 10},
		},
	}
	mockSvc.On("GetShipment", mock.Anything, shipmentID).Return(shipment, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/shipments/SHP-2026-001", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data struct {
			ID           string `json:"id"`
			SupplierName string `json:"supplier_name"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, shipmentID, resp.Data.ID)
	assert.Equal(t, "DHL Express", resp.Data.SupplierName)
	mockSvc.AssertExpectations(t)
}

func TestShipmentHandler_GetShipment_404(t *testing.T) {
	r, mockSvc := setupShipmentTest()

	mockSvc.On("GetShipment", mock.Anything, "SHP-NOT-FOUND").Return(nil, service.ErrNotFound)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/shipments/SHP-NOT-FOUND", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestShipmentHandler_GetMetrics_200(t *testing.T) {
	r, mockSvc := setupShipmentTest()

	mockSvc.On("GetMetrics", mock.Anything).Return(int64(10), int64(3), int64(1), float64(95.5), nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/shipments/metrics", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestShipmentHandler_ListCarriers_200(t *testing.T) {
	r, mockSvc := setupShipmentTest()

	carriers := []models.Carrier{
		{ID: 1, Name: "DHL Express", Code: "DHL"},
		{ID: 2, Name: "FedEx", Code: "FEDEX"},
	}
	mockSvc.On("ListCarriers", mock.Anything).Return(carriers, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/shipments/carriers", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestShipmentHandler_CreateShipment_201(t *testing.T) {
	r, mockSvc := setupShipmentTest()

	supplierID := uuid.New()
	mockSvc.On("CreateShipment", mock.Anything, mock.AnythingOfType("*models.Shipment")).Return(nil)

	body, _ := json.Marshal(map[string]interface{}{
		"po_ref":      "PO-2026-001",
		"supplier_id": supplierID.String(),
		"carrier":     "DHL Express",
		"tracking_no": "TRACK-ABC",
		"origin":      "Hillsboro, OR",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/shipments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestShipmentHandler_CreateShipment_400_MissingBody(t *testing.T) {
	r, _ := setupShipmentTest()

	body, _ := json.Marshal(map[string]interface{}{
		// Missing required fields: po_ref, supplier_id, carrier
		"tracking_no": "TRACK-ABC",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/shipments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestShipmentHandler_CreateShipment_404_PONotFound(t *testing.T) {
	r, mockSvc := setupShipmentTest()

	supplierID := uuid.New()
	mockSvc.On("CreateShipment", mock.Anything, mock.AnythingOfType("*models.Shipment")).Return(service.ErrNotFound)

	body, _ := json.Marshal(map[string]interface{}{
		"po_ref":      "PO-INVALID",
		"supplier_id": supplierID.String(),
		"carrier":     "DHL Express",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/shipments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}
