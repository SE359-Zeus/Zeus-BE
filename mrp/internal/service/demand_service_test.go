package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
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
