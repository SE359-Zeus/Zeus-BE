package service_test

import (
	"context"
	"testing"

	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/pagination"
	"zeus-scm-service/internal/repository"
	"zeus-scm-service/internal/service"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupInventorySvc() (service.IInventoryService, *repository.MockInventoryRepository) {
	repo := new(repository.MockInventoryRepository)
	svc := service.NewInventoryService(repo)
	return svc, repo
}

func TestInventoryService_GetProduct_Success(t *testing.T) {
	svc, repo := setupInventorySvc()
	id := uuid.New()
	expected := &models.Product{ID: id, ProductName: "Test Product"}

	repo.On("GetProductByID", anyCtx, id).Return(expected, nil)

	result, err := svc.GetProduct(context.Background(), id)
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	repo.AssertExpectations(t)
}

func TestInventoryService_GetProduct_NotFound(t *testing.T) {
	svc, repo := setupInventorySvc()
	id := uuid.New()

	repo.On("GetProductByID", anyCtx, id).Return(nil, assert.AnError)

	result, err := svc.GetProduct(context.Background(), id)
	assert.ErrorIs(t, err, service.ErrNotFound)
	assert.Nil(t, result)
	repo.AssertExpectations(t)
}

func TestInventoryService_ListProducts_Success(t *testing.T) {
	svc, repo := setupInventorySvc()
	params := pagination.Params{Page: 1, Limit: 15}
	expected := []models.Product{{ProductName: "P1"}, {ProductName: "P2"}}
	meta := &pagination.Meta{Page: 1, Limit: 15, TotalRows: 2, TotalPages: 1}

	repo.On("ListProducts", anyCtx, params, "").Return(expected, meta, nil)

	products, resultMeta, err := svc.ListProducts(context.Background(), params, "")
	assert.NoError(t, err)
	assert.Len(t, products, 2)
	assert.Equal(t, int64(2), resultMeta.TotalRows)
	repo.AssertExpectations(t)
}

func TestInventoryService_CreateProduct_Success(t *testing.T) {
	svc, repo := setupInventorySvc()
	p := &models.Product{ProductName: "New Product"}

	repo.On("CreateProduct", anyCtx, p).Return(nil)

	err := svc.CreateProduct(context.Background(), p)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestInventoryService_UpdateProduct_Success(t *testing.T) {
	svc, repo := setupInventorySvc()
	id := uuid.New()
	fields := map[string]any{"product_name": "Updated"}
	updated := &models.Product{ID: id, ProductName: "Updated"}

	repo.On("UpdateProduct", anyCtx, id, fields).Return(int64(1), nil)
	repo.On("GetProductByID", anyCtx, id).Return(updated, nil)

	result, err := svc.UpdateProduct(context.Background(), id, fields)
	assert.NoError(t, err)
	assert.Equal(t, "Updated", result.ProductName)
	repo.AssertExpectations(t)
}

func TestInventoryService_UpdateProduct_NotFound(t *testing.T) {
	svc, repo := setupInventorySvc()
	id := uuid.New()

	repo.On("UpdateProduct", anyCtx, id, mock.Anything).Return(int64(0), nil)

	result, err := svc.UpdateProduct(context.Background(), id, map[string]any{"name": "X"})
	assert.ErrorIs(t, err, service.ErrNotFound)
	assert.Nil(t, result)
	repo.AssertExpectations(t)
}

func TestInventoryService_GetProductModel_Success(t *testing.T) {
	svc, repo := setupInventorySvc()
	expected := &models.ProductModel{ModelCode: "M100", ModelName: "Model 100"}

	repo.On("GetProductModelByCode", anyCtx, "M100").Return(expected, nil)

	result, err := svc.GetProductModel(context.Background(), "M100")
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	repo.AssertExpectations(t)
}

func TestInventoryService_GetProductModel_NotFound(t *testing.T) {
	svc, repo := setupInventorySvc()

	repo.On("GetProductModelByCode", anyCtx, "UNKNOWN").Return(nil, assert.AnError)

	result, err := svc.GetProductModel(context.Background(), "UNKNOWN")
	assert.ErrorIs(t, err, service.ErrNotFound)
	assert.Nil(t, result)
	repo.AssertExpectations(t)
}

func TestInventoryService_CreateProductModel_Success(t *testing.T) {
	svc, repo := setupInventorySvc()
	m := &models.ProductModel{ModelCode: "M200", ModelName: "Model 200"}

	repo.On("CreateProductModel", anyCtx, m).Return(nil)

	err := svc.CreateProductModel(context.Background(), m)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestInventoryService_GetPart_Success(t *testing.T) {
	svc, repo := setupInventorySvc()
	id := uuid.New()
	expected := &models.Part{ID: id, SerialNumber: "SN-001"}

	repo.On("GetPartByID", anyCtx, id).Return(expected, nil)

	result, err := svc.GetPart(context.Background(), id)
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	repo.AssertExpectations(t)
}

func TestInventoryService_GetPart_NotFound(t *testing.T) {
	svc, repo := setupInventorySvc()
	id := uuid.New()

	repo.On("GetPartByID", anyCtx, id).Return(nil, assert.AnError)

	result, err := svc.GetPart(context.Background(), id)
	assert.ErrorIs(t, err, service.ErrNotFound)
	assert.Nil(t, result)
	repo.AssertExpectations(t)
}

func TestInventoryService_ListParts_Success(t *testing.T) {
	svc, repo := setupInventorySvc()
	params := pagination.Params{Page: 1, Limit: 15}
	expected := []models.Part{{SerialNumber: "SN-001"}}
	meta := &pagination.Meta{Page: 1, Limit: 15, TotalRows: 1}

	repo.On("ListParts", anyCtx, (*uuid.UUID)(nil), (*uuid.UUID)(nil), (*int32)(nil), params, "").
		Return(expected, meta, nil)

	parts, resultMeta, err := svc.ListParts(context.Background(), nil, nil, nil, params, "")
	assert.NoError(t, err)
	assert.Len(t, parts, 1)
	assert.Equal(t, int64(1), resultMeta.TotalRows)
	repo.AssertExpectations(t)
}

func TestInventoryService_CreatePart_Success(t *testing.T) {
	svc, repo := setupInventorySvc()
	p := &models.Part{SerialNumber: "SN-002"}

	repo.On("CreatePart", anyCtx, p).Return(nil)

	err := svc.CreatePart(context.Background(), p)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestInventoryService_UpdatePart_Success(t *testing.T) {
	svc, repo := setupInventorySvc()
	id := uuid.New()
	fields := map[string]any{"serial_number": "SN-UPDATED"}
	updated := &models.Part{ID: id, SerialNumber: "SN-UPDATED"}

	repo.On("UpdatePart", anyCtx, id, fields).Return(int64(1), nil)
	repo.On("GetPartByID", anyCtx, id).Return(updated, nil)

	result, err := svc.UpdatePart(context.Background(), id, fields)
	assert.NoError(t, err)
	assert.Equal(t, "SN-UPDATED", result.SerialNumber)
	repo.AssertExpectations(t)
}

func TestInventoryService_UpdatePart_NotFound(t *testing.T) {
	svc, repo := setupInventorySvc()
	id := uuid.New()

	repo.On("UpdatePart", anyCtx, id, mock.Anything).Return(int64(0), nil)

	result, err := svc.UpdatePart(context.Background(), id, map[string]any{"name": "X"})
	assert.ErrorIs(t, err, service.ErrNotFound)
	assert.Nil(t, result)
	repo.AssertExpectations(t)
}

func TestInventoryService_UpdatePartCondition_Success(t *testing.T) {
	svc, repo := setupInventorySvc()
	id := uuid.New()

	repo.On("UpdatePartFields", anyCtx, id, map[string]interface{}{
		"part_condition_id": int32(2),
	}).Return(int64(1), nil)

	err := svc.UpdatePartCondition(context.Background(), id, 2)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestInventoryService_UpdatePartCondition_NotFound(t *testing.T) {
	svc, repo := setupInventorySvc()
	id := uuid.New()

	repo.On("UpdatePartFields", anyCtx, id, mock.Anything).Return(int64(0), nil)

	err := svc.UpdatePartCondition(context.Background(), id, 2)
	assert.ErrorIs(t, err, service.ErrNotFound)
	repo.AssertExpectations(t)
}

func TestInventoryService_MarkPartScrapped_Success(t *testing.T) {
	svc, repo := setupInventorySvc()
	id := uuid.New()

	repo.On("UpdatePartFields", anyCtx, id, mock.MatchedBy(func(updates map[string]interface{}) bool {
		_, ok := updates["scrapped_date"]
		return ok
	})).Return(int64(1), nil)

	err := svc.MarkPartScrapped(context.Background(), id)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestInventoryService_MarkPartScrapped_NotFound(t *testing.T) {
	svc, repo := setupInventorySvc()
	id := uuid.New()

	repo.On("UpdatePartFields", anyCtx, id, mock.Anything).Return(int64(0), nil)

	err := svc.MarkPartScrapped(context.Background(), id)
	assert.ErrorIs(t, err, service.ErrNotFound)
	repo.AssertExpectations(t)
}

func TestInventoryService_InstallPart_Success(t *testing.T) {
	svc, repo := setupInventorySvc()
	partID := uuid.New()
	productID := uuid.New()

	repo.On("UpdatePartFields", anyCtx, partID, mock.MatchedBy(func(updates map[string]interface{}) bool {
		pid, ok := updates["product_id"]
		_, hasDate := updates["installation_date"]
		return ok && pid == productID && hasDate
	})).Return(int64(1), nil)

	err := svc.InstallPart(context.Background(), partID, productID)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestInventoryService_InstallPart_NotFound(t *testing.T) {
	svc, repo := setupInventorySvc()
	partID := uuid.New()
	productID := uuid.New()

	repo.On("UpdatePartFields", anyCtx, partID, mock.Anything).Return(int64(0), nil)

	err := svc.InstallPart(context.Background(), partID, productID)
	assert.ErrorIs(t, err, service.ErrNotFound)
	repo.AssertExpectations(t)
}

func TestInventoryService_RemovePart_Success(t *testing.T) {
	svc, repo := setupInventorySvc()
	id := uuid.New()

	repo.On("UpdatePartFields", anyCtx, id, mock.MatchedBy(func(updates map[string]interface{}) bool {
		_, hasNil := updates["product_id"]
		_, hasDate := updates["removal_date"]
		return hasNil && hasDate
	})).Return(int64(1), nil)

	err := svc.RemovePart(context.Background(), id)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestInventoryService_RemovePart_NotFound(t *testing.T) {
	svc, repo := setupInventorySvc()
	id := uuid.New()

	repo.On("UpdatePartFields", anyCtx, id, mock.Anything).Return(int64(0), nil)

	err := svc.RemovePart(context.Background(), id)
	assert.ErrorIs(t, err, service.ErrNotFound)
	repo.AssertExpectations(t)
}

func TestInventoryService_GetPartCatalog_Success(t *testing.T) {
	svc, repo := setupInventorySvc()
	id := uuid.New()
	expected := &models.PartCatalog{ID: id, PartNumber: "PN-001"}

	repo.On("GetPartCatalogByID", anyCtx, id).Return(expected, nil)

	result, err := svc.GetPartCatalog(context.Background(), id)
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	repo.AssertExpectations(t)
}

func TestInventoryService_GetPartCatalog_NotFound(t *testing.T) {
	svc, repo := setupInventorySvc()
	id := uuid.New()

	repo.On("GetPartCatalogByID", anyCtx, id).Return(nil, assert.AnError)

	result, err := svc.GetPartCatalog(context.Background(), id)
	assert.ErrorIs(t, err, service.ErrNotFound)
	assert.Nil(t, result)
	repo.AssertExpectations(t)
}

func TestInventoryService_ListPartCatalog_Success(t *testing.T) {
	svc, repo := setupInventorySvc()
	typeID := int32(1)
	params := pagination.Params{Page: 1, Limit: 15}
	expected := []models.PartCatalog{{PartNumber: "PN-001"}}
	meta := &pagination.Meta{Page: 1, Limit: 15, TotalRows: 1}

	repo.On("ListPartCatalog", anyCtx, &typeID, params, "").Return(expected, meta, nil)

	catalogs, resultMeta, err := svc.ListPartCatalog(context.Background(), &typeID, params, "")
	assert.NoError(t, err)
	assert.Len(t, catalogs, 1)
	assert.Equal(t, int64(1), resultMeta.TotalRows)
	repo.AssertExpectations(t)
}

func TestInventoryService_ListParts_WithProductID(t *testing.T) {
	svc, repo := setupInventorySvc()
	params := pagination.Params{Page: 1, Limit: 15}
	productID := uuid.New()
	expected := []models.Part{{SerialNumber: "SN-001"}}
	meta := &pagination.Meta{Page: 1, Limit: 15, TotalRows: 1}

	repo.On("ListParts", anyCtx, (*uuid.UUID)(nil), &productID, (*int32)(nil), params, "").
		Return(expected, meta, nil)

	parts, resultMeta, err := svc.ListParts(context.Background(), nil, &productID, nil, params, "")
	assert.NoError(t, err)
	assert.Len(t, parts, 1)
	assert.Equal(t, int64(1), resultMeta.TotalRows)
	repo.AssertExpectations(t)
}

func TestInventoryService_CreateComponentStock_ChoosesPrimarySupplierFromMappings(t *testing.T) {
	svc, repo := setupInventorySvc()
	sku := "SOC-XM100-PRO"
	primarySupplierID := uuid.New()
	secondarySupplierID := uuid.New()
	primarySupplier := &models.Supplier{ID: primarySupplierID, Name: "Intel Corporation", LeadTimeDays: 14}
	secondarySupplier := &models.Supplier{ID: secondarySupplierID, Name: "Arrow Electronics", LeadTimeDays: 21}
	stock := &models.ComponentStock{SKU: sku, Name: "Zeus SOC XM100 Pro", Category: "Processor", StockQty: 245, ReorderPoint: 100, UnitCost: 580}

	repo.On("FindSkuMappingsBySKU", anyCtx, sku).Return([]models.SkuMapping{
		{SupplierID: primarySupplierID, SKU: sku},
		{SupplierID: secondarySupplierID, SKU: sku},
	}, nil)
	repo.On("GetSupplierByID", anyCtx, primarySupplierID).Return(primarySupplier, nil).Maybe()
	repo.On("GetSupplierByID", anyCtx, secondarySupplierID).Return(secondarySupplier, nil).Maybe()
	repo.On("CreateComponentStock", anyCtx, mock.MatchedBy(func(created *models.ComponentStock) bool {
		if created == nil {
			return false
		}
		return created.SKU == sku && created.PrimarySupplierID != uuid.Nil && created.PrimarySupplier != "" && created.LeadTimeDays > 0
	})).Return(nil)

	err := svc.CreateComponentStock(context.Background(), stock)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}
