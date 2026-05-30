package handler_test

import (
	"bytes"
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

func TestPOHandler_ListPOs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockSvc := new(service.MockPOService)
	h := handler.NewPOHandler(mockSvc)

	r := gin.New()
	r.GET("/purchase-orders", h.ListPOs)

	pos := []models.PurchaseOrder{
		{
			ID:          "PO-2024-110",
			VendorName:  "Samsung Electronics",
			TargetBuild: "Titan Gaming Pro",
			Status:      models.POStatusDraft,
			TotalValue:  150.0,
			Notes:       "Please deliver ASAP",
		},
	}
	meta := &pagination.Meta{
		TotalRows:  1,
		Page:       1,
		Limit:      15,
		TotalPages: 1,
	}

	mockSvc.On("ListPOs", mock.Anything, pagination.Params{Page: 1, Limit: 15, Sort: "created_at", Order: "desc"}, "PO-2024").
		Return(pos, meta, nil)

	req, _ := http.NewRequest("GET", "/purchase-orders?q=PO-2024&page=1&limit=15", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "PO-2024-110")
	assert.Contains(t, w.Body.String(), "Samsung Electronics")
	assert.Contains(t, w.Body.String(), "Titan Gaming Pro")
	assert.Contains(t, w.Body.String(), "Please deliver ASAP")
	mockSvc.AssertExpectations(t)
}

func TestPOHandler_GetPO(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockSvc := new(service.MockPOService)
	h := handler.NewPOHandler(mockSvc)

	r := gin.New()
	r.GET("/purchase-orders/:poId", h.GetPO)

	po := &models.PurchaseOrder{
		ID:          "PO-2024-110",
		VendorName:  "Samsung Electronics",
		TargetBuild: "Titan Gaming Pro",
		Status:      models.POStatusDraft,
		TotalValue:  100.0,
		LineItems: []models.POLineItem{
			{
				ID:          uuid.New(),
				POID:        "PO-2024-110",
				SKU:         "COMP-1",
				Description: "Widget 1",
				OrderedQty:  10,
				UnitPrice:   10.0,
			},
		},
	}

	mockSvc.On("GetPO", mock.Anything, "PO-2024-110").Return(po, nil)

	req, _ := http.NewRequest("GET", "/purchase-orders/PO-2024-110", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "PO-2024-110")
	assert.Contains(t, w.Body.String(), "Samsung Electronics")
	assert.Contains(t, w.Body.String(), "Titan Gaming Pro")
	assert.Contains(t, w.Body.String(), "COMP-1")
	assert.Contains(t, w.Body.String(), "Widget 1")
	mockSvc.AssertExpectations(t)
}

func TestPOHandler_CreatePO(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockSvc := new(service.MockPOService)
	h := handler.NewPOHandler(mockSvc)

	r := gin.New()
	r.POST("/purchase-orders", h.CreatePO)

	vendorID := uuid.New()
	mockSvc.On("CreatePO", mock.Anything, mock.MatchedBy(func(po *models.PurchaseOrder) bool {
		return po.ID == "PO-2024-110" &&
			po.VendorID == vendorID &&
			po.TargetBuild == "Titan Gaming Pro" &&
			po.Notes == "Special order" &&
			len(po.LineItems) == 1 &&
			po.LineItems[0].SKU == "COMP-1" &&
			po.LineItems[0].OrderedQty == 20
	})).Return(nil)

	reqBody := `{"id":"PO-2024-110","expected_delivery":"2026-06-15T00:00:00Z","vendor_id":"` + vendorID.String() + `","target_build":"Titan Gaming Pro","items":[{"sku":"COMP-1","qty":20}],"notes":"Special order"}`
	req, _ := http.NewRequest("POST", "/purchase-orders", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 201, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestPOHandler_ExportPOReport(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockSvc := new(service.MockPOService)
	h := handler.NewPOHandler(mockSvc)

	r := gin.New()
	r.GET("/purchase-orders/export", h.ExportPOReport)

	pos := []models.PurchaseOrder{
		{
			ID:               "PO-2024-110",
			VendorName:       "Samsung Electronics",
			TargetBuild:      "Titan Gaming Pro",
			Status:           models.POStatusDraft,
			ExpectedDelivery: mustParseRFC3339(t, "2026-06-15T00:00:00Z"),
			TotalValue:       500.0,
			Notes:            "Special order",
			LineItems: []models.POLineItem{
				{
					SKU:         "COMP-1",
					Description: "Widget 1",
					OrderedQty:  20,
					UnitPrice:   25.0,
				},
			},
		},
	}

	mockSvc.On("FindAllPOs", mock.Anything).Return(pos, nil)

	req, _ := http.NewRequest("GET", "/purchase-orders/export", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "text/csv", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Content-Disposition"), "purchase_orders_report.csv")
	assert.Contains(t, w.Body.String(), "PO ID,Vendor Name,Target Build,Status,Expected Delivery")
	assert.Contains(t, w.Body.String(), "PO-2024-110,Samsung Electronics,Titan Gaming Pro,Draft,2026-06-15T00:00:00Z,500.00,Special order,COMP-1,Widget 1,20,25.00,500.00")
	mockSvc.AssertExpectations(t)
}

func mustParseRFC3339(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse time: %v", err)
	}
	return parsed
}
