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
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupSeedingTest() (*gin.Engine, *service.MockSeedingService) {
	gin.SetMode(gin.TestMode)
	mockSvc := new(service.MockSeedingService)
	h := handler.NewSeedingHandler(mockSvc)
	r := gin.New()
	v1 := r.Group("/api/v1")
	{
		v1.POST("/seeding/products", h.CreateProduct)
		v1.POST("/seeding/products/register", h.RegisterProduct)
		v1.POST("/seeding/parts", h.CreatePart)
	}
	return r, mockSvc
}

func TestSeedingHandler_CreateProduct_201(t *testing.T) {
	r, mockSvc := setupSeedingTest()

	mockSvc.On("CreateProduct", mock.Anything, mock.AnythingOfType("*models.Product")).Return(nil)

	body, _ := json.Marshal(map[string]string{
		"product_model_code": "Z-1000",
		"product_name":       "New Product",
		"serial_number":      "SN-001",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/seeding/products", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestSeedingHandler_CreateProduct_400_InvalidBody(t *testing.T) {
	r, _ := setupSeedingTest()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/seeding/products", bytes.NewReader([]byte(`not json`)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSeedingHandler_RegisterProduct_201(t *testing.T) {
	r, mockSvc := setupSeedingTest()

	mockSvc.On("CreateProduct", mock.Anything, mock.AnythingOfType("*models.Product")).Return(nil)

	body, _ := json.Marshal(map[string]any{
		"product_model_code": "82SN003JVN",
		"customer_id":        uuid.New().String(),
		"product_name":       "IdeaPad 5 Pro",
		"serial_number":      "SN-82SN003JVN-99",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/seeding/products/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestSeedingHandler_CreatePart_201(t *testing.T) {
	r, mockSvc := setupSeedingTest()

	mockSvc.On("CreatePart", mock.Anything, mock.AnythingOfType("*models.Part")).Return(nil)

	body, _ := json.Marshal(map[string]string{
		"serial_number": "SN-002",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/seeding/parts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockSvc.AssertExpectations(t)
}
