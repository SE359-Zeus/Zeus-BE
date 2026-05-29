package service

import (
	"context"
	"testing"
	"time"

	"zeus-mrp-service/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ------------------------------------------------------------
// Original tests (kept intact)
// ------------------------------------------------------------

func TestProductionService_GetDemandSummary(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	res, err := svc.GetDemandSummary(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, res)
}

func TestProductionService_GeneratePOsForShortages(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	err := svc.GeneratePOsForShortages(context.Background())
	assert.NoError(t, err)
}

func TestProductionService_GeneratePickList(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	res, err := svc.GeneratePickList(context.Background(), uuid.New())
	assert.NoError(t, err)
	assert.NotNil(t, res)
}

func TestProductionService_GetAggregatedDemand(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	res, err := svc.GetAggregatedDemand(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, res)
}

// ------------------------------------------------------------
// Hard: GetDemandSummary
// ------------------------------------------------------------

// Hard: stub returns nil — callers will range over this; must be an empty slice
func TestGetDemandSummary_ReturnsSliceNotNil(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	res, err := svc.GetDemandSummary(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, res, "GetDemandSummary must return an empty slice, not nil")
}

// Hard: every row in the summary must have a non-empty OrderID and non-negative MissingCount
func TestGetDemandSummary_RowFieldsAreValid(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	rows, err := svc.GetDemandSummary(context.Background())
	require.NoError(t, err)

	for i, row := range rows {
		assert.NotEmpty(t, row.OrderID,
			"row[%d]: OrderID must not be empty", i)
		assert.NotEmpty(t, row.TargetBuild,
			"row[%d]: TargetBuild must not be empty", i)
		assert.Greater(t, row.Quantity, 0,
			"row[%d]: Quantity must be > 0", i)
		assert.GreaterOrEqual(t, row.MissingCount, 0,
			"row[%d]: MissingCount must be >= 0", i)
	}
}

// Hard: context cancellation must be respected
func TestGetDemandSummary_CancelledContext(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := svc.GetDemandSummary(ctx)
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

// ------------------------------------------------------------
// Hard: GeneratePickList
// ------------------------------------------------------------

// Hard: uuid.Nil is not a valid order reference
func TestGeneratePickList_RejectsNilOrderID(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	res, err := svc.GeneratePickList(context.Background(), uuid.Nil)
	assert.Error(t, err, "uuid.Nil must be rejected immediately")
	assert.Nil(t, res)
}

// Hard: the returned DTO must echo back the requested order ID
func TestGeneratePickList_OrderIDIsEchoedInResponse(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	id := uuid.New()
	res, err := svc.GeneratePickList(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, id, res.OrderID,
		"PickListDTO.OrderID must match the requested order ID")
}

// Hard: Components must be a non-nil slice so callers can range safely
func TestGeneratePickList_ComponentsAreNonNil(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	res, err := svc.GeneratePickList(context.Background(), uuid.New())
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.NotNil(t, res.Components,
		"PickListDTO.Components must be an empty slice, not nil")
}

// Hard: every item in the pick list must have a positive qty, non-empty SKU, and valid PartID
func TestGeneratePickList_EachComponentHasValidFields(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	res, err := svc.GeneratePickList(context.Background(), uuid.New())
	require.NoError(t, err)
	require.NotNil(t, res)

	for i, item := range res.Components {
		assert.Greater(t, item.Quantity, 0,
			"pick list item[%d]: Quantity must be > 0", i)
		assert.NotEmpty(t, item.SKU,
			"pick list item[%d]: SKU must not be empty", i)
		assert.NotEqual(t, uuid.Nil, item.PartID,
			"pick list item[%d]: PartID must not be nil", i)
	}
}

// Hard: no pick list item should appear more than once — duplicates indicate a BOM join bug
func TestGeneratePickList_NoDuplicatePartIDs(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	res, err := svc.GeneratePickList(context.Background(), uuid.New())
	require.NoError(t, err)
	require.NotNil(t, res)

	seen := make(map[uuid.UUID]int)
	for _, item := range res.Components {
		seen[item.PartID]++
	}
	for partID, count := range seen {
		assert.Equal(t, 1, count,
			"PartID %s appears %d times in pick list — parts must be deduplicated", partID, count)
	}
}

// ------------------------------------------------------------
// Hard: GetAggregatedDemand
// ------------------------------------------------------------

// Hard: aggregated demand must not contain duplicate PartIDs — aggregation means collapsing per-part
func TestGetAggregatedDemand_NoDuplicatePartIDs(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	results, err := svc.GetAggregatedDemand(context.Background())
	require.NoError(t, err)

	seen := make(map[uuid.UUID]int)
	for _, r := range results {
		seen[r.PartID]++
	}
	for partID, count := range seen {
		assert.Equal(t, 1, count,
			"PartID %s appears %d times — aggregation must produce one entry per part", partID, count)
	}
}

// Hard: every aggregated result must have TotalRequiredQty > 0 — zero-demand entries are noise
func TestGetAggregatedDemand_TotalRequiredQtyIsPositive(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	results, err := svc.GetAggregatedDemand(context.Background())
	require.NoError(t, err)

	for i, r := range results {
		assert.Greater(t, r.TotalRequiredQty, 0,
			"result[%d]: TotalRequiredQty must be > 0 in aggregated demand", i)
	}
}

// ------------------------------------------------------------
// Hard: GeneratePOsForShortages — idempotency
// ------------------------------------------------------------

// Hard: calling twice must not return an error on the second call (idempotent SCM handoff)
func TestGeneratePOsForShortages_IsIdempotent(t *testing.T) {
	svc := NewProductionService(setupMockRepo())

	err := svc.GeneratePOsForShortages(context.Background())
	assert.NoError(t, err, "first call must succeed")

	err = svc.GeneratePOsForShortages(context.Background())
	assert.NoError(t, err, "second call must also succeed — operation must be idempotent")
}

// Hard: cancelled context must be respected before any Redis publish
func TestGeneratePOsForShortages_CancelledContext(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := svc.GeneratePOsForShortages(ctx)
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestDemandService_DynamicCalculationsAndHandoff(t *testing.T) {
	mockRepo := new(MockMRPRepository)
	mockSCM := new(MockSCMClient)
	mockAudit := new(MockAuditPublisher)

	orderID := uuid.New()
	partID := uuid.New()
	vendorID := uuid.New()

	orders := []models.ProductionOrder{
		{
			ID:               orderID,
			ProductModelCode: "ZW-X1-TITAN",
			TargetQuantity:   10,
			Status:           models.StatusShortage,
			ScheduledAt:      time.Now(),
			CreatedAt:        time.Now(),
		},
	}

	bomEntries := []models.BomEntry{
		{
			ParentModelCode:         "ZW-X1-TITAN",
			ComponentPartID:         partID,
			RequiredQuantityPerUnit: 2,
		},
	}

	partCatalog := &models.Part{
		ID:       partID,
		SKU:      "PART-SKU-1",
		StockQty: 5,
	}

	mockRepo.On("GetOpenProductionOrders", mock.Anything).Return(orders, nil)
	mockRepo.On("GetProductionOrder", mock.Anything, orderID).Return(&orders[0], nil)
	mockRepo.On("GetBOMByModelCode", mock.Anything, "ZW-X1-TITAN").Return(bomEntries, nil)
	mockSCM.On("GetPartCatalogByID", mock.Anything, partID).Return(partCatalog, nil)
	mockSCM.On("GetProductModelByCode", mock.Anything, "ZW-X1-TITAN").Return(&models.ProductModel{ModelCode: "ZW-X1-TITAN", ModelName: "Zeus Workstation X1"}, nil)

	svc := NewProductionService(mockRepo, mockSCM, mockAudit)

	mockSCM.On("GetOptimalSupplier", mock.Anything, "PART-SKU-1").Return(vendorID, 12.50, nil)

	summary, err := svc.GetDemandSummary(context.Background())
	assert.NoError(t, err)
	assert.Len(t, summary, 1)
	assert.Equal(t, "Zeus Workstation X1", summary[0].ProductName)
	assert.Equal(t, "ZW-X1-TITAN", summary[0].TargetBuild)
	assert.Equal(t, 10, summary[0].Quantity)
	assert.Equal(t, 2, summary[0].QtyReady)
	assert.Equal(t, "HIGH", summary[0].Priority)
	assert.Equal(t, 1, summary[0].POCount)

	agg, err := svc.GetAggregatedDemand(context.Background())
	assert.NoError(t, err)
	assert.Len(t, agg, 1)
	assert.Equal(t, partID, agg[0].PartID)
	assert.Equal(t, 15, agg[0].TotalRequiredQty)

	mockSCM.On("CreateDraftPO", mock.Anything, vendorID, "ZW-X1-TITAN").Return("PO-DRAFT-001", nil)
	mockAudit.On("PublishJSON", mock.Anything, "system.deficit.pool", mock.Anything).Return(nil)
	mockSCM.On("AddLineItemWithLock", mock.Anything, "PO-DRAFT-001", "PART-SKU-1", 15).Return(nil)

	err = svc.GeneratePOsForShortages(context.Background())
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
	mockSCM.AssertExpectations(t)
	mockAudit.AssertExpectations(t)
}

