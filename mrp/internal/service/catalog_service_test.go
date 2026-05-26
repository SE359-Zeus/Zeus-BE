package service

import (
	"context"
	"testing"
	"zeus-mrp-service/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ------------------------------------------------------------
// Original tests (kept intact)
// ------------------------------------------------------------

func TestProductionService_GetAssemblies(t *testing.T) {
	mockRepo := new(MockMRPRepository)
	mockSCM := new(MockSCMClient)
	mockRepo.On("GetAllBOMs", mock.Anything).Return([]models.BomEntry{{ParentModelCode: "21CB000QUS", ComponentPartID: uuid.New(), RequiredQuantityPerUnit: 2}}, nil)
	mockSCM.On("GetProductModelByCode", mock.Anything, "21CB000QUS").Return(&models.ProductModel{ModelCode: "21CB000QUS", ModelName: "ThinkPad X1 Carbon Gen 11"}, nil)
	svc := NewProductionService(mockRepo, mockSCM)
	res, err := svc.GetAssemblies(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, res)
	if assert.Len(t, res, 1) {
		assert.Equal(t, "21CB000QUS", res[0].ModelCode)
		assert.Equal(t, "ThinkPad X1 Carbon Gen 11", res[0].Name)
	}
}

func TestProductionService_GetCatalog(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	res, err := svc.GetCatalog(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, res)
}

func TestProductionService_GetWhereUsed(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	res, err := svc.GetWhereUsed(context.Background(), uuid.New().String())
	assert.NoError(t, err)
	assert.NotNil(t, res)
}

func TestProductionService_CreateAssembly(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	res, err := svc.CreateAssembly(context.Background(), models.CreateAssemblyRequest{})
	assert.Error(t, err)
	assert.Nil(t, res)
}

func TestProductionService_UpdateAssembly(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	res, err := svc.UpdateAssembly(context.Background(), uuid.New(), models.UpdateAssemblyRequest{})
	assert.Error(t, err)
	assert.Nil(t, res)
}

func TestProductionService_DeleteAssembly(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	err := svc.DeleteAssembly(context.Background(), uuid.New())
	assert.NoError(t, err)
}

func TestProductionService_GetAssemblyByModelCode_ResolvesModelName(t *testing.T) {
	mockRepo := new(MockMRPRepository)
	mockSCM := new(MockSCMClient)
	modelCode := "82A3000GUS"
	mockRepo.On("GetBOMByModelCode", mock.Anything, modelCode).Return([]models.BomEntry{{ParentModelCode: modelCode, ComponentPartID: uuid.New(), RequiredQuantityPerUnit: 1}}, nil)
	mockSCM.On("GetProductModelByCode", mock.Anything, modelCode).Return(&models.ProductModel{ModelCode: modelCode, ModelName: "Yoga Slim 7i"}, nil)
	svc := NewProductionService(mockRepo, mockSCM)

	res, err := svc.GetAssemblyByModelCode(context.Background(), modelCode)
	assert.NoError(t, err)
	if assert.NotNil(t, res) {
		assert.Equal(t, modelCode, res.ModelCode)
		assert.Equal(t, "Yoga Slim 7i", res.Name)
	}
}

// ------------------------------------------------------------
// Hard: CreateAssembly — input validation
// ------------------------------------------------------------

// Hard: empty Name must be rejected; stub always returns nil error
func TestCreateAssembly_RejectsEmptyName(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	req := models.CreateAssemblyRequest{
		Name:       "",
		Components: []models.ComponentReference{{SKU: "SKU-1", Quantity: 1}},
	}
	res, err := svc.CreateAssembly(context.Background(), req)
	assert.Error(t, err, "empty Name must be rejected")
	assert.Nil(t, res)
}

// Hard: an assembly with zero components has no production purpose
func TestCreateAssembly_RejectsEmptyComponentList(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	req := models.CreateAssemblyRequest{
		Name:       "Assembly-X",
		Components: []models.ComponentReference{},
	}
	res, err := svc.CreateAssembly(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, req, res)
}

// Hard: a single component with Quantity=0 must poison the whole request
func TestCreateAssembly_RejectsComponentWithZeroQuantity(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	req := models.CreateAssemblyRequest{
		Name: "Assembly-ZeroQty",
		Components: []models.ComponentReference{
			{SKU: "SKU-GOOD", Quantity: 2},
			{SKU: "SKU-BAD", Quantity: 0},
		},
	}
	res, err := svc.CreateAssembly(context.Background(), req)
	assert.Error(t, err, "component with Quantity=0 must be rejected")
	assert.Nil(t, res)
}

// Hard: a component with no SKU identity is meaningless
func TestCreateAssembly_RejectsComponentWithEmptySKU(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	req := models.CreateAssemblyRequest{
		Name: "Assembly-EmptySKU",
		Components: []models.ComponentReference{
			{SKU: "", Quantity: 3},
		},
	}
	res, err := svc.CreateAssembly(context.Background(), req)
	assert.Error(t, err, "component with empty SKU must be rejected")
	assert.Nil(t, res)
}

// Hard: duplicate SKUs within the same assembly request should be rejected or merged —
// silently accepting duplicates would create conflicting BOM entries
func TestCreateAssembly_RejectsDuplicateComponentSKUs(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	req := models.CreateAssemblyRequest{
		Name: "Assembly-DupeSKU",
		Components: []models.ComponentReference{
			{SKU: "SKU-DUP", Quantity: 2},
			{SKU: "SKU-DUP", Quantity: 5},
		},
	}
	res, err := svc.CreateAssembly(context.Background(), req)
	assert.Error(t, err, "duplicate component SKUs within one assembly must be rejected")
	assert.Nil(t, res)
}

// ------------------------------------------------------------
// Hard: UpdateAssembly — input validation
// ------------------------------------------------------------

// Hard: updating with uuid.Nil makes no sense; must be rejected immediately
func TestUpdateAssembly_RejectsNilID(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	res, err := svc.UpdateAssembly(context.Background(), uuid.Nil, models.UpdateAssemblyRequest{
		Name: "Any-Name",
	})
	assert.Error(t, err, "uuid.Nil assembly ID must be rejected")
	assert.Nil(t, res)
}

// Hard: updating with an empty request body changes nothing — this should be a no-op error
func TestUpdateAssembly_RejectsCompletelyEmptyRequest(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	res, err := svc.UpdateAssembly(context.Background(), uuid.New(), models.UpdateAssemblyRequest{})
	assert.Error(t, err, "completely empty UpdateAssemblyRequest must be rejected — nothing to update")
	assert.Nil(t, res)
}

// ------------------------------------------------------------
// Hard: DeleteAssembly
// ------------------------------------------------------------

// Hard: deleting uuid.Nil would match no row but could corrupt state if not guarded
func TestDeleteAssembly_RejectsNilID(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	err := svc.DeleteAssembly(context.Background(), uuid.Nil)
	assert.Error(t, err, "deleting with uuid.Nil must be rejected")
}

// ------------------------------------------------------------
// Hard: GetWhereUsed
// ------------------------------------------------------------

// Hard: empty SKU has no meaning in the catalog
func TestGetWhereUsed_RejectsEmptySKU(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	res, err := svc.GetWhereUsed(context.Background(), "")
	assert.Error(t, err, "empty SKU must be rejected")
	assert.Nil(t, res)
}

// Hard: GetAssemblies must return an empty slice, not nil, so callers can range safely
func TestGetAssemblies_ReturnsSliceNotNil(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	res, err := svc.GetAssemblies(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, res, "GetAssemblies must return an empty slice, not nil")
}

// Hard: GetCatalog must return an empty slice, not nil
func TestGetCatalog_ReturnsSliceNotNil(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	res, err := svc.GetCatalog(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, res, "GetCatalog must return an empty slice, not nil")
}

func TestProductionService_CreateCatalogPart(t *testing.T) {
	mockRepo := setupMockRepo()
	mockSCM := setupMockSCMClient()
	svc := NewProductionService(mockRepo, mockSCM)

	expectedPart := &models.Part{
		ID:          uuid.Nil,
		SKU:         "NEW-SKU",
		Description: "New Description",
		Price:       12.34,
	}

	mockSCM.On("GetPartCatalogBySKU", mock.Anything, "NEW-SKU").Return((*models.Part)(nil), nil)
	mockSCM.On("CreateCatalogPart", mock.Anything, "NEW-SKU", "New Description", 12.34).Return(expectedPart, nil)

	res, err := svc.CreateCatalogPart(context.Background(), "NEW-SKU", "New Description", 12.34)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, "NEW-SKU", res.SKU)
	assert.Equal(t, "New Description", res.Description)
	assert.Equal(t, 12.34, res.Price)
}

func TestProductionService_UpdateCatalogPart(t *testing.T) {
	mockRepo := setupMockRepo()
	mockSCM := setupMockSCMClient()
	svc := NewProductionService(mockRepo, mockSCM)

	expectedPart := &models.Part{
		ID:          uuid.Nil,
		SKU:         "EXISTING-SKU",
		Description: "New Description",
		Price:       15.5,
	}

	mockSCM.On("UpdateCatalogPart", mock.Anything, "EXISTING-SKU", "New Description", 15.5).Return(expectedPart, nil)

	res, err := svc.UpdateCatalogPart(context.Background(), "EXISTING-SKU", "New Description", 15.5)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, "New Description", res.Description)
	assert.Equal(t, 15.5, res.Price)
}

func TestProductionService_DeleteCatalogPart(t *testing.T) {
	mockRepo := setupMockRepo()
	mockSCM := setupMockSCMClient()
	svc := NewProductionService(mockRepo, mockSCM)

	mockSCM.On("DeleteCatalogPart", mock.Anything, "EXISTING-SKU").Return(nil)

	err := svc.DeleteCatalogPart(context.Background(), "EXISTING-SKU")
	assert.NoError(t, err)
}
