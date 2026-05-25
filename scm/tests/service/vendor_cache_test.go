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

type MockVendorCache struct {
	mock.Mock
}

func (m *MockVendorCache) GetOptimalSupplier(ctx context.Context, sku string) (*models.Supplier, *models.SkuMapping, error) {
	args := m.Called(ctx, sku)
	if args.Get(0) != nil {
		return args.Get(0).(*models.Supplier), args.Get(1).(*models.SkuMapping), args.Error(2)
	}
	return nil, nil, args.Error(2)
}

func (m *MockVendorCache) SetOptimalSupplier(ctx context.Context, sku string, supplier *models.Supplier, mapping *models.SkuMapping) error {
	args := m.Called(ctx, sku, supplier, mapping)
	return args.Error(0)
}

func (m *MockVendorCache) DeleteOptimalSupplier(ctx context.Context, sku string) error {
	args := m.Called(ctx, sku)
	return args.Error(0)
}

func TestCachedVendorService_GetOptimalSupplier_CacheHit(t *testing.T) {
	baseSvc := new(service.MockVendorService)
	cacheMock := new(MockVendorCache)
	repoMock := new(repository.MockVendorRepository)
	svc := service.NewCachedVendorService(baseSvc, cacheMock, repoMock)

	sku := "TEST-SKU"
	supplier := &models.Supplier{ID: uuid.New(), Name: "Cached Supplier"}
	mapping := &models.SkuMapping{ID: uuid.New(), SKU: sku}

	cacheMock.On("GetOptimalSupplier", anyCtx, sku).Return(supplier, mapping, nil)

	resSupplier, resMapping, err := svc.GetOptimalSupplier(context.Background(), sku)
	assert.NoError(t, err)
	assert.Equal(t, supplier, resSupplier)
	assert.Equal(t, mapping, resMapping)

	baseSvc.AssertNotCalled(t, "GetOptimalSupplier", anyCtx, anyCtx)
	cacheMock.AssertExpectations(t)
}

func TestCachedVendorService_GetOptimalSupplier_CacheMiss(t *testing.T) {
	baseSvc := new(service.MockVendorService)
	cacheMock := new(MockVendorCache)
	repoMock := new(repository.MockVendorRepository)
	svc := service.NewCachedVendorService(baseSvc, cacheMock, repoMock)

	sku := "TEST-SKU"
	supplier := &models.Supplier{ID: uuid.New(), Name: "Base Supplier"}
	mapping := &models.SkuMapping{ID: uuid.New(), SKU: sku}

	cacheMock.On("GetOptimalSupplier", anyCtx, sku).Return(nil, nil, nil)
	baseSvc.On("GetOptimalSupplier", anyCtx, sku).Return(supplier, mapping, nil)
	cacheMock.On("SetOptimalSupplier", anyCtx, sku, supplier, mapping).Return(nil)

	resSupplier, resMapping, err := svc.GetOptimalSupplier(context.Background(), sku)
	assert.NoError(t, err)
	assert.Equal(t, supplier, resSupplier)
	assert.Equal(t, mapping, resMapping)

	baseSvc.AssertExpectations(t)
	cacheMock.AssertExpectations(t)
}

func TestCachedVendorService_UpdateSupplierMetrics_InvalidatesCache(t *testing.T) {
	baseSvc := new(service.MockVendorService)
	cacheMock := new(MockVendorCache)
	repoMock := new(repository.MockVendorRepository)
	svc := service.NewCachedVendorService(baseSvc, cacheMock, repoMock)

	supplierID := uuid.New()
	mappings := []models.SkuMapping{
		{SKU: "SKU-1"},
		{SKU: "SKU-2"},
	}

	baseSvc.On("UpdateSupplierMetrics", anyCtx, supplierID).Return(nil)
	repoMock.On("FindSkuMappingsBySupplierID", anyCtx, supplierID).Return(mappings, nil)
	cacheMock.On("DeleteOptimalSupplier", anyCtx, "SKU-1").Return(nil)
	cacheMock.On("DeleteOptimalSupplier", anyCtx, "SKU-2").Return(nil)

	err := svc.UpdateSupplierMetrics(context.Background(), supplierID)
	assert.NoError(t, err)

	baseSvc.AssertExpectations(t)
	repoMock.AssertExpectations(t)
	cacheMock.AssertExpectations(t)
}
