package handler_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"zeus-scm-service/internal/handler"
	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/pagination"
	"zeus-scm-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestVendorHandler_ListSuppliers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockSvc := new(service.MockVendorService)
	h := handler.NewVendorHandler(mockSvc)

	r := gin.New()
	r.GET("/vendors", h.ListSuppliers)

	suppliers := []models.Supplier{
		{
			ID:           uuid.New(),
			Name:         "Vendor 1",
			Contact:      "Contact 1",
			Tier:         models.SupplierTier1,
			LeadTimeDays: 5,
			QualityScore: 98,
			OnTimeRate:   95,
		},
	}
	meta := &pagination.Meta{
		TotalRows:  1,
		Page:       1,
		Limit:      15,
		TotalPages: 1,
	}

	mockSvc.On("ListSuppliers", mock.Anything, "Tier 1", pagination.Params{Page: 1, Limit: 15, Sort: "created_at", Order: "desc"}, "Vendor").
		Return(suppliers, meta, nil)

	req, _ := http.NewRequest("GET", "/vendors?tier=Tier%201&q=Vendor&page=1&limit=15", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestVendorHandler_GetSupplierMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockSvc := new(service.MockVendorService)
	h := handler.NewVendorHandler(mockSvc)

	r := gin.New()
	r.GET("/vendors/metrics", h.GetSupplierMetrics)

	mockSvc.On("GetSupplierMetrics", mock.Anything).
		Return(int64(5), 92.5, nil)

	req, _ := http.NewRequest("GET", "/vendors/metrics", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), `"total_active_suppliers":5`)
	assert.Contains(t, w.Body.String(), `"on_time_delivery_rate":92.5`)
	mockSvc.AssertExpectations(t)
}

func TestVendorHandler_CreateSupplier(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockSvc := new(service.MockVendorService)
	h := handler.NewVendorHandler(mockSvc)

	r := gin.New()
	r.POST("/vendors", h.CreateSupplier)

	mockSvc.On("CreateSupplier", mock.Anything, mock.MatchedBy(func(s *models.Supplier) bool {
		return s.Name == "New Supplier" && s.Tier == models.SupplierTier1
	})).Return(nil)

	reqBody := `{"name":"New Supplier","contact":"test@test.com","tier":"Tier 1","lead_time_days":3}`
	req, _ := http.NewRequest("POST", "/vendors", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 201, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestVendorHandler_CreateSkuMapping(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockSvc := new(service.MockVendorService)
	h := handler.NewVendorHandler(mockSvc)

	r := gin.New()
	r.POST("/vendors/:id/sku-mappings", h.CreateSkuMapping)

	supplierID := uuid.New()
	mockSvc.On("CreateSkuMapping", mock.Anything, mock.MatchedBy(func(m *models.SkuMapping) bool {
		return m.SupplierID == supplierID && m.SKU == "COMP-SKU" && m.UnitPrice == 15.5
	})).Return(nil)

	reqBody := `{"sku":"COMP-SKU","name":"Widget","unit_price":15.5,"lead_time_days":5,"min_order_qty":10}`
	req, _ := http.NewRequest("POST", "/vendors/"+supplierID.String()+"/sku-mappings", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 201, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestVendorHandler_ExportSuppliersReport(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockSvc := new(service.MockVendorService)
	h := handler.NewVendorHandler(mockSvc)

	r := gin.New()
	r.GET("/vendors/export", h.ExportSuppliersReport)

	supplierID := uuid.New()
	suppliers := []models.Supplier{
		{
			ID:           supplierID,
			Name:         "Vendor A",
			Contact:      "Contact A",
			Tier:         models.SupplierTier1,
			LeadTimeDays: 4,
			QualityScore: 99,
			OnTimeRate:   98,
			SkuMappings: []models.SkuMapping{
				{
					ID:           uuid.New(),
					SupplierID:   supplierID,
					SKU:          "SKU-1",
					Name:         "Mapping 1",
					UnitPrice:    10.5,
					LeadTimeDays: 3,
					MinOrderQty:  5,
				},
			},
		},
	}

	mockSvc.On("FindAllSuppliersWithMappings", mock.Anything).Return(suppliers, nil)

	req, _ := http.NewRequest("GET", "/vendors/export", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "text/csv", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "Supplier Name")
	assert.Contains(t, w.Body.String(), "Vendor A")
	assert.Contains(t, w.Body.String(), "SKU-1")
	mockSvc.AssertExpectations(t)
}
