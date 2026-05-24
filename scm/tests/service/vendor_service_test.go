package service_test

import (
	"context"
	"testing"

	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/repository"
	"zeus-scm-service/internal/service"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var anyCtx = mock.Anything

func setupVendorSvc() (service.IVendorService, *repository.MockVendorRepository) {
	repo := new(repository.MockVendorRepository)
	svc := service.NewVendorService(repo)
	return svc, repo
}

func TestVendorService_GetOptimalSupplier_Success(t *testing.T) {
	svc, repo := setupVendorSvc()
	sku := "SOC-XM100-PRO"

	supplier := &models.Supplier{
		ID:           uuid.New(),
		Name:         "Best Supplier",
		QualityScore: 95,
		OnTimeRate:   90,
	}
	mapping := models.SkuMapping{
		ID:         uuid.New(),
		SupplierID: supplier.ID,
		SKU:        sku,
		Name:       "SoC XM100 Pro",
		UnitPrice:  150.0,
	}

	repo.On("FindSkuMappingsBySKU", anyCtx, sku).Return([]models.SkuMapping{mapping}, nil)
	repo.On("GetSupplierByID", anyCtx, mapping.SupplierID).Return(supplier, nil)

	result, resultMapping, err := svc.GetOptimalSupplier(context.Background(), sku)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, resultMapping)
	assert.Equal(t, supplier.Name, result.Name)
	assert.Equal(t, mapping.SKU, resultMapping.SKU)
	repo.AssertExpectations(t)
}

func TestVendorService_GetOptimalSupplier_NoMappings(t *testing.T) {
	svc, repo := setupVendorSvc()

	repo.On("FindSkuMappingsBySKU", anyCtx, "UNKNOWN").Return(nil, nil)

	result, _, err := svc.GetOptimalSupplier(context.Background(), "UNKNOWN")
	assert.ErrorIs(t, err, service.ErrNoOptimalSupplier)
	assert.Nil(t, result)
	repo.AssertExpectations(t)
}

func TestVendorService_GetOptimalSupplier_NoSupplierFound(t *testing.T) {
	svc, repo := setupVendorSvc()

	mapping := models.SkuMapping{
		ID:         uuid.New(),
		SupplierID: uuid.New(),
		SKU:        "SKU-001",
		UnitPrice:  100.0,
	}

	repo.On("FindSkuMappingsBySKU", anyCtx, "SKU-001").Return([]models.SkuMapping{mapping}, nil)
	repo.On("GetSupplierByID", anyCtx, mapping.SupplierID).Return(nil, assert.AnError)

	result, _, err := svc.GetOptimalSupplier(context.Background(), "SKU-001")
	assert.ErrorIs(t, err, service.ErrNoOptimalSupplier)
	assert.Nil(t, result)
	repo.AssertExpectations(t)
}

func TestVendorService_GetOptimalSupplier_PicksHighestScore(t *testing.T) {
	svc, repo := setupVendorSvc()
	sku := "SKU-002"

	supplierLow := &models.Supplier{ID: uuid.New(), Name: "Low", QualityScore: 50, OnTimeRate: 50}
	supplierHigh := &models.Supplier{ID: uuid.New(), Name: "High", QualityScore: 95, OnTimeRate: 95}

	mappingLow := models.SkuMapping{ID: uuid.New(), SupplierID: supplierLow.ID, SKU: sku, UnitPrice: 200}
	mappingHigh := models.SkuMapping{ID: uuid.New(), SupplierID: supplierHigh.ID, SKU: sku, UnitPrice: 100}

	repo.On("FindSkuMappingsBySKU", anyCtx, sku).Return([]models.SkuMapping{mappingLow, mappingHigh}, nil)
	repo.On("GetSupplierByID", anyCtx, supplierLow.ID).Return(supplierLow, nil)
	repo.On("GetSupplierByID", anyCtx, supplierHigh.ID).Return(supplierHigh, nil)

	result, resultMapping, err := svc.GetOptimalSupplier(context.Background(), sku)
	assert.NoError(t, err)
	assert.Equal(t, "High", result.Name)
	assert.Equal(t, mappingHigh.ID, resultMapping.ID)
	repo.AssertExpectations(t)
}

func TestVendorService_UpdateSupplierMetrics_NoReceipts(t *testing.T) {
	svc, repo := setupVendorSvc()
	id := uuid.New()

	repo.On("CountGoodsReceiptsByVendor", anyCtx, id).Return(int64(0), nil)
	repo.On("UpdateSupplier", anyCtx, id, mock.MatchedBy(func(updates map[string]interface{}) bool {
		return updates["on_time_rate"] == 0 && updates["quality_score"] == 100
	})).Return(nil)

	err := svc.UpdateSupplierMetrics(context.Background(), id)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestVendorService_UpdateSupplierMetrics_WithReceipts(t *testing.T) {
	svc, repo := setupVendorSvc()
	id := uuid.New()
	grID := "GR-2025-001"

	repo.On("CountGoodsReceiptsByVendor", anyCtx, id).Return(int64(2), nil)
	repo.On("FindGoodsReceiptsByVendor", anyCtx, id).Return([]models.GoodsReceipt{
		{ID: grID, Status: models.GRStatusComplete},
		{ID: "GR-2025-002", Status: models.GRStatusInspected},
	}, nil)

	repo.On("FindGRLineItemsByGRID", anyCtx, grID).Return([]models.GRLineItem{
		{SKU: "SOC-001", DefectiveQty: intPtr(1)},
	}, nil)
	repo.On("FindGRLineItemsByGRID", anyCtx, "GR-2025-002").Return([]models.GRLineItem{
		{SKU: "SOC-002", DefectiveQty: intPtr(0)},
	}, nil)

	repo.On("UpdateSupplier", anyCtx, id, mock.MatchedBy(func(updates map[string]interface{}) bool {
		onTime, ok1 := updates["on_time_rate"].(float64)
		quality, ok2 := updates["quality_score"].(float64)
		return ok1 && ok2 && onTime == 50 && quality == 50
	})).Return(nil)

	err := svc.UpdateSupplierMetrics(context.Background(), id)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func intPtr(v int) *int {
	return &v
}
