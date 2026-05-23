package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"zeus-be/pkg/response"
	"zeus-scm-service/internal/handler"
	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupVendorHandlerTest() (*gin.Engine, *service.MockVendorService) {
	gin.SetMode(gin.TestMode)
	mockSvc := new(service.MockVendorService)
	h := handler.NewVendorHandler(mockSvc)
	r := gin.New()
	v1 := r.Group("/api/v1")
	{
		v1.GET("/vendors/optimal", h.GetOptimalSupplier)
		v1.POST("/vendors/:id/recalc-metrics", h.UpdateSupplierMetrics)
	}
	return r, mockSvc
}

func TestVendorHandler_GetOptimalSupplier_Success(t *testing.T) {
	r, mockSvc := setupVendorHandlerTest()

	supplier := &models.Supplier{
		ID:   uuid.New(),
		Name: "Best Supplier",
	}
	mapping := &models.SkuMapping{
		SKU:  "SOC-XM100-PRO",
		Name: "SoC XM100 Pro",
	}
	mockSvc.On("GetOptimalSupplier", mock.Anything, "SOC-XM100-PRO").
		Return(supplier, mapping, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/vendors/optimal?sku=SOC-XM100-PRO", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp response.SuccessResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	mockSvc.AssertExpectations(t)
}

func TestVendorHandler_GetOptimalSupplier_MissingSKU(t *testing.T) {
	r, mockSvc := setupVendorHandlerTest()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/vendors/optimal", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestVendorHandler_GetOptimalSupplier_NotFound(t *testing.T) {
	r, mockSvc := setupVendorHandlerTest()

	mockSvc.On("GetOptimalSupplier", mock.Anything, "UNKNOWN").
		Return(nil, nil, service.ErrNoOptimalSupplier)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/vendors/optimal?sku=UNKNOWN", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestVendorHandler_UpdateSupplierMetrics_Success(t *testing.T) {
	r, mockSvc := setupVendorHandlerTest()

	id := uuid.New()
	mockSvc.On("UpdateSupplierMetrics", mock.Anything, id).Return(nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/vendors/"+id.String()+"/recalc-metrics", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestVendorHandler_UpdateSupplierMetrics_InvalidID(t *testing.T) {
	r, mockSvc := setupVendorHandlerTest()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/vendors/invalid-uuid/recalc-metrics", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockSvc.AssertExpectations(t)
}
