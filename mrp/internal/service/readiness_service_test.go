package service

import (
	"context"
	"sync"
	"testing"
	"zeus-mrp-service/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ------------------------------------------------------------
// Original tests (kept intact)
// ------------------------------------------------------------

func TestProductionService_CheckClearToBuild_Strict(t *testing.T) {
	svc := NewProductionService(setupMockRepo())

	tests := []struct {
		name        string
		orderID     uuid.UUID
		expectError bool
		expectedCTB bool
	}{
		{
			name:        "All Parts Available",
			orderID:     uuid.New(),
			expectError: false,
			expectedCTB: true,
		},
		{
			name:        "Critical Shortage (1 part missing)",
			orderID:     uuid.New(),
			expectError: false,
			expectedCTB: false,
		},
		{
			name:        "Multiple Parts Missing",
			orderID:     uuid.New(),
			expectError: false,
			expectedCTB: false,
		},
		{
			name:        "Partial Availability (e.g. 5 needed, 3 in stock)",
			orderID:     uuid.New(),
			expectError: false,
			expectedCTB: false,
		},
		{
			name:        "Invalid Order ID",
			orderID:     uuid.Nil,
			expectError: true,
			expectedCTB: false,
		},
		{
			name:        "Order ID Not Found in DB",
			orderID:     uuid.New(),
			expectError: true,
			expectedCTB: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctb, err := svc.CheckClearToBuild(context.Background(), tt.orderID)

			if tt.expectError {
				assert.Error(t, err, "Expected an error for invalid input or missing data")
			} else {
				assert.NoError(t, err)
				if tt.expectedCTB {
					assert.True(t, ctb, "CTB must be true when 100% components are available")
				} else {
					assert.False(t, ctb, "CTB must be false if any component is missing or partial")
				}
			}
		})
	}
}

func TestProductionService_BOMExplosion_EdgeCases(t *testing.T) {
	svc := NewProductionService(setupMockRepo())

	tests := []struct {
		name             string
		orderID          uuid.UUID
		mockBOM          []models.BomEntry
		mockInventory    map[uuid.UUID]int
		expectedError    bool
		expectedShortage bool
	}{
		{
			name:             "Sufficient Stock",
			orderID:          uuid.New(),
			expectedError:    false,
			expectedShortage: false,
		},
		{
			name:             "Exact Stock Match",
			orderID:          uuid.New(),
			expectedError:    false,
			expectedShortage: false,
		},
		{
			name:             "One Part Missing - Shortage",
			orderID:          uuid.New(),
			expectedError:    false,
			expectedShortage: true,
		},
		{
			name:             "Partial Stock - Shortage",
			orderID:          uuid.New(),
			expectedError:    false,
			expectedShortage: true,
		},
		{
			name:             "Empty BOM (No components required)",
			orderID:          uuid.New(),
			expectedError:    false,
			expectedShortage: false,
		},
		{
			name:             "Invalid Order ID",
			orderID:          uuid.Nil,
			expectedError:    true,
			expectedShortage: false,
		},
		{
			name:             "BOM Entry with Zero Quantity Required",
			orderID:          uuid.New(),
			expectedError:    true,
			expectedShortage: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := svc.RunBOMExplosion(context.Background(), tt.orderID)

			if tt.expectedError {
				assert.Error(t, err, "Expected an error for invalid input or state")
			} else {
				assert.NoError(t, err)
				if tt.expectedShortage {
					hasShortage := false
					for _, r := range results {
						if r.IsShortage {
							hasShortage = true
						}
					}
					assert.True(t, hasShortage, "Should detect shortage in this scenario")
				}
			}
		})
	}
}

// NOTE: original called svc.GetReadinessMatrix(ctx) — wrong signature.
// Correct signature is GetReadinessMatrix(ctx, filter, page). Fixed below.
func TestProductionService_GetReadinessMatrix(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	filter := models.ReadinessFilter{}
	page := models.PaginationParams{Page: 1, PerPage: 20}
	res, err := svc.GetReadinessMatrix(context.Background(), filter, page)
	assert.NoError(t, err)
	assert.NotNil(t, res)
}

func TestProductionService_GetReadinessMetrics(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	res, err := svc.GetReadinessMetrics(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, res)
}

func TestProductionService_ExportReadinessReport(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	res, err := svc.ExportReadinessReport(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, res)
}

// ------------------------------------------------------------
// Hard: CheckClearToBuild
// ------------------------------------------------------------

// Hard: uuid.Nil must be rejected immediately without hitting the DB
func TestCheckClearToBuild_RejectsNilOrderID(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	ctb, err := svc.CheckClearToBuild(context.Background(), uuid.Nil)
	assert.Error(t, err, "uuid.Nil must be rejected immediately")
	assert.False(t, ctb)
}

// Hard: stub ignores context — must propagate context.Canceled
func TestCheckClearToBuild_CancelledContext(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := svc.CheckClearToBuild(ctx, uuid.New())
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

// Hard: 30 concurrent calls must not data-race
// Run with: go test -race
func TestCheckClearToBuild_ConcurrentCallsAreRaceFree(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	const goroutines = 30
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			svc.CheckClearToBuild(context.Background(), uuid.New()) //nolint:errcheck
		}()
	}
	wg.Wait()
}

// ------------------------------------------------------------
// Hard: RunBOMExplosion
// ------------------------------------------------------------

// Hard: uuid.Nil must return an error and a nil slice
func TestRunBOMExplosion_RejectsNilOrderID(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	results, err := svc.RunBOMExplosion(context.Background(), uuid.Nil)
	assert.Error(t, err)
	assert.Nil(t, results)
}

// Hard: every result entry must carry a non-nil PartID
func TestRunBOMExplosion_ResultsHaveNonNilPartIDs(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	results, err := svc.RunBOMExplosion(context.Background(), uuid.New())
	require.NoError(t, err)

	for i, r := range results {
		assert.NotEqual(t, uuid.Nil, r.PartID,
			"result[%d] has nil PartID — BOM explosion must resolve real part references", i)
	}
}

// Hard: IsShortage flag must be consistent with qty arithmetic
// A stub returning empty slices passes vacuously; real data must satisfy this exactly.
func TestRunBOMExplosion_ShortageLogicIsConsistent(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	results, err := svc.RunBOMExplosion(context.Background(), uuid.New())
	require.NoError(t, err)

	for i, r := range results {
		shouldBeShortage := r.AvailableQty < r.TotalRequiredQty
		assert.Equal(t, shouldBeShortage, r.IsShortage,
			"result[%d]: IsShortage=%v but Available=%d, Required=%d — inconsistent",
			i, r.IsShortage, r.AvailableQty, r.TotalRequiredQty)
	}
}

// Hard: TotalRequiredQty must be > 0 — a BOM entry with 0 required qty is corrupt data
func TestRunBOMExplosion_RejectsZeroRequiredQty(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	results, err := svc.RunBOMExplosion(context.Background(), uuid.New())
	require.NoError(t, err)

	for i, r := range results {
		assert.Greater(t, r.TotalRequiredQty, 0,
			"result[%d]: TotalRequiredQty must be > 0; zero-qty BOM entries are invalid", i)
	}
}

// ------------------------------------------------------------
// Hard: GetReadinessMatrix — pagination + filter contracts
// ------------------------------------------------------------

// Hard: must return an empty slice (not nil) so callers can range safely
func TestGetReadinessMatrix_ReturnsSliceNotNilOnEmptyData(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	filter := models.ReadinessFilter{}
	page := models.PaginationParams{Page: 1, PerPage: 20}

	res, err := svc.GetReadinessMatrix(context.Background(), filter, page)
	require.NoError(t, err)
	assert.NotNil(t, res, "GetReadinessMatrix must return an empty slice, not nil")
}

// Hard: page numbers are 1-indexed; page=0 is nonsensical
func TestGetReadinessMatrix_PaginationZeroPageIsRejected(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	_, err := svc.GetReadinessMatrix(
		context.Background(),
		models.ReadinessFilter{},
		models.PaginationParams{Page: 0, PerPage: 20},
	)
	assert.Error(t, err, "page=0 must be rejected — pages are 1-indexed")
}

// Hard: negative per_page is nonsensical
func TestGetReadinessMatrix_NegativePerPageIsRejected(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	_, err := svc.GetReadinessMatrix(
		context.Background(),
		models.ReadinessFilter{},
		models.PaginationParams{Page: 1, PerPage: -1},
	)
	assert.Error(t, err, "negative per_page must be rejected")
}

// Hard: unknown status filter value must be rejected rather than silently ignored
func TestGetReadinessMatrix_UnknownStatusFilterIsRejected(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	_, err := svc.GetReadinessMatrix(
		context.Background(),
		models.ReadinessFilter{Status: "INVALID_STATUS_XYZ"},
		models.PaginationParams{Page: 1, PerPage: 10},
	)
	assert.Error(t, err, "unknown status filter value must be rejected")
}

// Hard: each returned row must have a valid OrderID, non-empty TargetBuild, and Quantity > 0
func TestGetReadinessMatrix_EachRowHasNonNilOrderID(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	rows, err := svc.GetReadinessMatrix(
		context.Background(),
		models.ReadinessFilter{},
		models.PaginationParams{Page: 1, PerPage: 100},
	)
	require.NoError(t, err)

	for i, row := range rows {
		assert.NotEqual(t, uuid.Nil, row.OrderID, "row[%d]: OrderID must not be nil", i)
		assert.NotEmpty(t, row.TargetBuild, "row[%d]: TargetBuild must not be empty", i)
		assert.Greater(t, row.Quantity, 0, "row[%d]: Quantity must be > 0", i)
	}
}

// ------------------------------------------------------------
// Hard: GetReadinessMetrics — field-level invariants
// ------------------------------------------------------------

// Hard: stub returns nil — a real impl must return a populated struct
func TestGetReadinessMetrics_ReturnsNonNilStruct(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	m, err := svc.GetReadinessMetrics(context.Background())
	require.NoError(t, err)
	require.NotNil(t, m, "GetReadinessMetrics must return a struct, not nil")
}

// Hard: SupplyReadinessRate is a percentage — must be 0–100, not a 0–1 fraction
func TestGetReadinessMetrics_SupplyReadinessRateIsBounded(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	m, err := svc.GetReadinessMetrics(context.Background())
	require.NoError(t, err)
	require.NotNil(t, m)

	assert.GreaterOrEqual(t, m.SupplyReadinessRate, 0.0, "SupplyReadinessRate must be >= 0.0")
	assert.LessOrEqual(t, m.SupplyReadinessRate, 100.0, "SupplyReadinessRate must be <= 100.0 (percent, not fraction)")
}

// Hard: all count fields must be non-negative
func TestGetReadinessMetrics_CountFieldsAreNonNegative(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	m, err := svc.GetReadinessMetrics(context.Background())
	require.NoError(t, err)
	require.NotNil(t, m)

	assert.GreaterOrEqual(t, m.TotalOpenOrders, 0)
	assert.GreaterOrEqual(t, m.ComponentsInShortage, 0)
	assert.GreaterOrEqual(t, m.BlockedOrders, 0)
}

// Hard: BlockedOrders can never exceed TotalOpenOrders — a blocked order is still an open order
func TestGetReadinessMetrics_BlockedOrdersDoNotExceedTotal(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	m, err := svc.GetReadinessMetrics(context.Background())
	require.NoError(t, err)
	require.NotNil(t, m)

	assert.LessOrEqual(t, m.BlockedOrders, m.TotalOpenOrders,
		"BlockedOrders (%d) cannot exceed TotalOpenOrders (%d)",
		m.BlockedOrders, m.TotalOpenOrders)
}

// ------------------------------------------------------------
// Hard: ExportReadinessReport
// ------------------------------------------------------------

// Hard: stub returns nil — a real impl must return a non-empty byte slice
func TestExportReadinessReport_ReturnsNonEmptyBytes(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	data, err := svc.ExportReadinessReport(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, data, "ExportReadinessReport must return a non-empty byte slice")
}
