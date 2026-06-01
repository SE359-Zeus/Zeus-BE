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
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupInventoryTest() (*gin.Engine, *service.MockInventoryService) {
	gin.SetMode(gin.TestMode)
	mockSvc := new(service.MockInventoryService)
	h := handler.NewInventoryHandler(mockSvc)
	r := gin.New()
	v1 := r.Group("/api/v1")
	{
		v1.GET("/inventory/products", h.ListProducts)
		v1.POST("/inventory/products", h.CreateProduct)
		v1.POST("/inventory/products/register", h.RegisterProduct)
		v1.GET("/inventory/products/:id", h.GetProduct)
		v1.GET("/inventory/product-models/:code", h.GetProductModel)
		v1.POST("/inventory/product-models", h.CreateProductModel)
		v1.DELETE("/inventory/product-models/:code", h.DeleteProductModel)
		v1.GET("/inventory/parts", h.ListParts)
		v1.POST("/inventory/parts", h.CreatePart)
		v1.GET("/inventory/parts/:id", h.GetPart)
		v1.PUT("/inventory/parts/:id/condition", h.UpdatePartCondition)
		v1.POST("/inventory/parts/:id/scrap", h.MarkPartScrapped)
		v1.POST("/inventory/parts/:id/install", h.InstallPart)
		v1.POST("/inventory/parts/:id/remove", h.RemovePart)
		v1.GET("/inventory/part-catalog", h.ListPartCatalog)
		v1.GET("/inventory/part-catalog/:id", h.GetPartCatalog)
		v1.PUT("/inventory/products/:id", h.UpdateProduct)
		v1.PUT("/inventory/parts/:id", h.UpdatePart)
		v1.GET("/inventory/stocks", h.ListStocks)
		v1.POST("/inventory/stocks", h.CreateComponentStock)
		v1.GET("/inventory/stocks/:sku", h.GetStockBySKU)
	}
	return r, mockSvc
}

func TestInventoryHandler_GetProduct_200(t *testing.T) {
	r, mockSvc := setupInventoryTest()
	id := uuid.New()
	product := &models.Product{ID: id, ProductName: "Test Product"}

	mockSvc.On("GetProduct", mock.Anything, id).Return(product, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/inventory/products/"+id.String(), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestInventoryHandler_GetProduct_404(t *testing.T) {
	r, mockSvc := setupInventoryTest()
	id := uuid.New()

	mockSvc.On("GetProduct", mock.Anything, id).Return(nil, service.ErrNotFound)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/inventory/products/"+id.String(), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestInventoryHandler_GetProduct_400_InvalidID(t *testing.T) {
	r, _ := setupInventoryTest()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/inventory/products/not-a-uuid", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestInventoryHandler_ListProducts_200(t *testing.T) {
	r, mockSvc := setupInventoryTest()
	products := []models.Product{{ProductName: "P1"}, {ProductName: "P2"}}
	meta := &pagination.Meta{Page: 1, Limit: 15, TotalRows: 2, TotalPages: 1}

	mockSvc.On("ListProducts", mock.Anything, mock.AnythingOfType("pagination.Params"), "", (*uuid.UUID)(nil)).Return(products, meta, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/inventory/products", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestInventoryHandler_CreateProduct_201(t *testing.T) {
	r, mockSvc := setupInventoryTest()

	mockSvc.On("CreateProduct", mock.Anything, mock.AnythingOfType("*models.Product")).Return(nil)

	body, _ := json.Marshal(map[string]string{
		"product_name":  "New Product",
		"serial_number": "SN-001",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/inventory/products", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestInventoryHandler_CreateProduct_400_InvalidBody(t *testing.T) {
	r, _ := setupInventoryTest()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/inventory/products", bytes.NewReader([]byte(`not json`)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestInventoryHandler_GetProductModel_200(t *testing.T) {
	r, mockSvc := setupInventoryTest()
	model := &models.ProductModel{ModelCode: "M100", ModelName: "Model 100"}

	mockSvc.On("GetProductModel", mock.Anything, "M100").Return(model, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/inventory/product-models/M100", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestInventoryHandler_GetProductModel_404(t *testing.T) {
	r, mockSvc := setupInventoryTest()

	mockSvc.On("GetProductModel", mock.Anything, "UNKNOWN").Return(nil, service.ErrNotFound)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/inventory/product-models/UNKNOWN", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestInventoryHandler_CreateProductModel_201(t *testing.T) {
	r, mockSvc := setupInventoryTest()

	mockSvc.On("CreateProductModel", mock.Anything, mock.AnythingOfType("*models.ProductModel")).Return(nil)

	body, _ := json.Marshal(map[string]string{
		"model_code": "M200",
		"model_name": "Model 200",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/inventory/product-models", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestInventoryHandler_GetPart_200(t *testing.T) {
	r, mockSvc := setupInventoryTest()
	id := uuid.New()
	part := &models.Part{ID: id, SerialNumber: "SN-001"}

	mockSvc.On("GetPart", mock.Anything, id).Return(part, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/inventory/parts/"+id.String(), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestInventoryHandler_GetPart_404(t *testing.T) {
	r, mockSvc := setupInventoryTest()
	id := uuid.New()

	mockSvc.On("GetPart", mock.Anything, id).Return(nil, service.ErrNotFound)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/inventory/parts/"+id.String(), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestInventoryHandler_GetPart_400_InvalidID(t *testing.T) {
	r, _ := setupInventoryTest()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/inventory/parts/not-a-uuid", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestInventoryHandler_ListParts_200(t *testing.T) {
	r, mockSvc := setupInventoryTest()
	parts := []models.Part{{SerialNumber: "SN-001"}}
	meta := &pagination.Meta{Page: 1, Limit: 15, TotalRows: 1}

	mockSvc.On("ListParts", mock.Anything, (*uuid.UUID)(nil), (*uuid.UUID)(nil), (*int32)(nil), mock.AnythingOfType("pagination.Params"), "").Return(parts, meta, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/inventory/parts", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestInventoryHandler_CreatePart_201(t *testing.T) {
	r, mockSvc := setupInventoryTest()

	mockSvc.On("CreatePart", mock.Anything, mock.AnythingOfType("*models.Part")).Return(nil)

	body, _ := json.Marshal(map[string]string{
		"serial_number": "SN-002",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/inventory/parts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestInventoryHandler_UpdatePartCondition_200(t *testing.T) {
	r, mockSvc := setupInventoryTest()
	id := uuid.New()

	mockSvc.On("UpdatePartCondition", mock.Anything, id, int32(2)).Return(nil)

	body, _ := json.Marshal(map[string]int32{"condition_id": 2})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/inventory/parts/"+id.String()+"/condition", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestInventoryHandler_MarkPartScrapped_200(t *testing.T) {
	r, mockSvc := setupInventoryTest()
	id := uuid.New()

	mockSvc.On("MarkPartScrapped", mock.Anything, id).Return(nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/inventory/parts/"+id.String()+"/scrap", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestInventoryHandler_InstallPart_200(t *testing.T) {
	r, mockSvc := setupInventoryTest()
	partID := uuid.New()
	productID := uuid.New()

	mockSvc.On("InstallPart", mock.Anything, partID, productID).Return(nil)

	body, _ := json.Marshal(map[string]string{"product_id": productID.String()})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/inventory/parts/"+partID.String()+"/install", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestInventoryHandler_RemovePart_200(t *testing.T) {
	r, mockSvc := setupInventoryTest()
	id := uuid.New()

	mockSvc.On("RemovePart", mock.Anything, id).Return(nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/inventory/parts/"+id.String()+"/remove", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestInventoryHandler_GetPartCatalog_200(t *testing.T) {
	r, mockSvc := setupInventoryTest()
	id := uuid.New()
	catalog := &models.PartCatalog{ID: id, PartNumber: "PN-001"}

	mockSvc.On("GetPartCatalog", mock.Anything, id).Return(catalog, nil)
	mockSvc.On("GetPartCatalogBySKU", mock.Anything, "PN-001").Return(catalog, 150.0, 100, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/inventory/part-catalog/"+id.String(), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestInventoryHandler_GetPartCatalog_404(t *testing.T) {
	r, mockSvc := setupInventoryTest()
	id := uuid.New()

	mockSvc.On("GetPartCatalog", mock.Anything, id).Return(nil, service.ErrNotFound)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/inventory/part-catalog/"+id.String(), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestInventoryHandler_ListPartCatalog_200(t *testing.T) {
	r, mockSvc := setupInventoryTest()
	catalogs := []models.PartCatalog{{PartNumber: "PN-001"}}
	meta := &pagination.Meta{Page: 1, Limit: 15, TotalRows: 1}

	mockSvc.On("ListPartCatalog", mock.Anything, (*int32)(nil), mock.AnythingOfType("pagination.Params"), "").Return(catalogs, meta, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/inventory/part-catalog", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestInventoryHandler_ListStocks_WithStatus_200(t *testing.T) {
	r, mockSvc := setupInventoryTest()
	stocks := []models.ComponentStock{{SKU: "SOC-XM100-PRO", Status: models.ComponentStatusLowStock, PrimarySupplier: "Intel Corporation"}}
	meta := &pagination.Meta{Page: 1, Limit: 15, TotalRows: 1}

	mockSvc.On("ListStocks", mock.Anything, mock.AnythingOfType("pagination.Params"), "Low Stock", "", (*uuid.UUID)(nil)).Return(stocks, meta, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/inventory/stocks?status=Low+Stock", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data []struct {
			SKU             string `json:"sku"`
			PrimarySupplier string `json:"primary_supplier"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "Intel Corporation", resp.Data[0].PrimarySupplier)
	mockSvc.AssertExpectations(t)
}

func TestInventoryHandler_CreateComponentStock_201(t *testing.T) {
	r, mockSvc := setupInventoryTest()

	mockSvc.On("CreateComponentStock", mock.Anything, mock.AnythingOfType("*models.ComponentStock")).Return(nil)

	body, _ := json.Marshal(map[string]any{
		"sku":           "SOC-XM100-PRO",
		"name":          "Zeus SOC XM100 Pro (14-Core)",
		"category":      "Processor",
		"stock_qty":     245,
		"reorder_point": 100,
		"unit_cost":     580.00,
		"location":      "WH-A / Zone-C1",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/inventory/stocks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestInventoryHandler_UpdateProduct_200(t *testing.T) {
	r, mockSvc := setupInventoryTest()
	id := uuid.New()
	updated := &models.Product{ID: id, ProductName: "Updated Name"}

	mockSvc.On("UpdateProduct", mock.Anything, id, mock.AnythingOfType("map[string]interface {}")).Return(updated, nil)

	body, _ := json.Marshal(map[string]string{"product_name": "Updated Name"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/inventory/products/"+id.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestInventoryHandler_UpdatePart_200(t *testing.T) {
	r, mockSvc := setupInventoryTest()
	id := uuid.New()
	updated := &models.Part{ID: id, SerialNumber: "SN-UPDATED"}

	mockSvc.On("UpdatePart", mock.Anything, id, mock.AnythingOfType("map[string]interface {}")).Return(updated, nil)

	body, _ := json.Marshal(map[string]string{"serial_number": "SN-UPDATED"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/inventory/parts/"+id.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestInventoryHandler_RegisterProduct_201(t *testing.T) {
	r, mockSvc := setupInventoryTest()

	mockSvc.On("CreateProduct", mock.Anything, mock.AnythingOfType("*models.Product")).Return(nil)

	body, _ := json.Marshal(map[string]any{
		"product_model_code": "82SN003JVN",
		"customer_id":        uuid.New().String(),
		"product_name":       "IdeaPad 5 Pro",
		"serial_number":      "SN-82SN003JVN-99",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/inventory/products/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestInventoryHandler_ListParts_WithProductID_200(t *testing.T) {
	r, mockSvc := setupInventoryTest()
	productID := uuid.New()
	parts := []models.Part{{SerialNumber: "SN-001"}}
	meta := &pagination.Meta{Page: 1, Limit: 15, TotalRows: 1}

	mockSvc.On("ListParts", mock.Anything, (*uuid.UUID)(nil), &productID, (*int32)(nil), mock.AnythingOfType("pagination.Params"), "").Return(parts, meta, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/inventory/parts?product_id="+productID.String(), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestInventoryHandler_DeleteProductModel_204(t *testing.T) {
	r, mockSvc := setupInventoryTest()

	mockSvc.On("DeleteProductModel", mock.Anything, "M100").Return(nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/inventory/product-models/M100", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	mockSvc.AssertExpectations(t)
}
