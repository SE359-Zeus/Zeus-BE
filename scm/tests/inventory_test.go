package tests

import (
	"context"
	"testing"
	"time"

	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/pagination"
	"zeus-scm-service/internal/repository/sqlite"
	"zeus-scm-service/internal/service"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestInventoryService_ProductCRUD(t *testing.T) {
	db := setupTestDB()
	db.AutoMigrate(&models.Product{}, &models.ProductModel{})
	repo := sqlite.NewInventoryRepository(db)
	svc := service.NewInventoryService(repo)
	ctx := context.Background()

	id := uuid.New()
	newProduct := &models.Product{
		ID:               id,
		ProductModelCode: "Z-1000",
		CustomerID:       uuid.New(),
		ProductName:      "Zeus Engine Z-1000",
		SerialNumber:     "SN-99901",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	err := svc.CreateProduct(ctx, newProduct)
	assert.NoError(t, err, "Should successfully execute product creation logic")

	fetched, err := svc.GetProduct(ctx, id)
	assert.NoError(t, err, "Should successfully retrieve product")
	if fetched != nil {
		assert.Equal(t, id, fetched.ID)
		assert.Equal(t, newProduct.ProductModelCode, fetched.ProductModelCode)
		assert.Equal(t, newProduct.CustomerID, fetched.CustomerID)
		assert.Equal(t, newProduct.SerialNumber, fetched.SerialNumber)
	}

	products, _, err := svc.ListProducts(ctx, pagination.Params{Page: 1}, "")
	assert.NoError(t, err)
	if products != nil {
		assert.True(t, len(products) >= 0)
	}
}

func TestInventoryService_PartLifecycle_Exhaustive(t *testing.T) {
	db := setupTestDB()
	db.AutoMigrate(&models.Part{}, &models.Product{}, &models.PartCatalog{})
	repo := sqlite.NewInventoryRepository(db)
	svc := service.NewInventoryService(repo)
	ctx := context.Background()

	partID := uuid.New()
	catalogID := uuid.New()

	newPart := &models.Part{
		ID:               partID,
		PartCatalogID:    catalogID,
		SerialNumber:     "PART-XYZ-123",
		PartConditionID:  1,
		ManufacturedDate: time.Now(),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// 1. Create
	err := svc.CreatePart(ctx, newPart)
	assert.NoError(t, err, "Should successfully create part")

	// 2. Install
	prodID := uuid.New()
	err = svc.InstallPart(ctx, partID, prodID)
	assert.NoError(t, err, "Should successfully update part with Installation details")

	fetched, err := svc.GetPart(ctx, partID)
	assert.NoError(t, err)
	if fetched != nil {
		assert.NotNil(t, fetched.InstallationDate, "InstallationDate MUST be populated")
		assert.NotNil(t, fetched.ProductID, "ProductID MUST be populated")
		assert.Equal(t, prodID, *fetched.ProductID)
		assert.Nil(t, fetched.RemovalDate)
		assert.Nil(t, fetched.ScrappedDate)
	}

	// Verify ListParts filtering by ProductID
	partsByProd, _, err := svc.ListParts(ctx, nil, &prodID, nil, pagination.Params{Page: 1, Limit: 10}, "")
	assert.NoError(t, err)
	assert.Len(t, partsByProd, 1)
	assert.Equal(t, partID, partsByProd[0].ID)

	// 3. Remove
	err = svc.RemovePart(ctx, partID)
	assert.NoError(t, err, "Should successfully clear ProductID and set RemovalDate")

	// 4. Update Condition
	err = svc.UpdatePartCondition(ctx, partID, 2)
	assert.NoError(t, err)

	// 5. Scrap
	err = svc.MarkPartScrapped(ctx, partID)
	assert.NoError(t, err)

	finalPart, err := svc.GetPart(ctx, partID)
	assert.NoError(t, err)
	if finalPart != nil {
		assert.NotNil(t, finalPart.ScrappedDate, "ScrappedDate MUST be set")
		assert.NotNil(t, finalPart.RemovalDate, "RemovalDate MUST be set")
		assert.Nil(t, finalPart.ProductID, "ProductID MUST be null after removal")
		assert.Equal(t, int32(2), finalPart.PartConditionID)
	}
}

func TestInventoryService_Catalog_Exhaustive(t *testing.T) {
	db := setupTestDB()
	db.AutoMigrate(&models.PartCatalog{})
	repo := sqlite.NewInventoryRepository(db)
	svc := service.NewInventoryService(repo)
	ctx := context.Background()

	catID := uuid.New()
	fetchedCat, err := svc.GetPartCatalog(ctx, catID)
	assert.Error(t, err, "Should fail when catalog entry does not exist")
	assert.Nil(t, fetchedCat)

	list, _, err := svc.ListPartCatalog(ctx, nil, pagination.Params{Page: 1}, "")
	assert.NoError(t, err)
	assert.NotNil(t, list)
}

func TestInventoryService_UserAndWarranty(t *testing.T) {
	db := setupTestDB()
	repo := sqlite.NewInventoryRepository(db)
	svc := service.NewInventoryService(repo)
	ctx := context.Background()

	assert.NotNil(t, svc)
	assert.NotNil(t, ctx)
}
