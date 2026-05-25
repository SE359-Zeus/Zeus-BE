package service

import (
	"context"
	"strings"
	"testing"

	"zeus-mrp-service/internal/models"

	"github.com/google/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ------------------------------------------------------------
// Original tests (kept intact)
// ------------------------------------------------------------

func TestProductionService_GetInventoryLedger(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	res, err := svc.GetInventoryLedger(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, res)
}

func TestProductionService_GetInventoryMetrics(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	res, err := svc.GetInventoryMetrics(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, res)
}

func TestProductionService_GetInventoryLedger_UsesSCMClient(t *testing.T) {
	mockRepo := setupMockRepo()
	mockSCM := setupMockSCMClient()
	mockSCM.On("ListStocks", mock.Anything, 1, 100, "sku", "asc", "").Return([]models.ComponentStock{{SKU: "SKU-1", Name: "Widget", StockQty: 12}}, false, nil)
	svc := NewProductionService(mockRepo, mockSCM)

	res, err := svc.GetInventoryLedger(context.Background())
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, "SKU-1", res[0].ID)
	assert.Equal(t, 12, res[0].QtyChange)
	assert.Equal(t, 12, res[0].RunningBalance)
}

func TestProductionService_GetInventoryBalanceBySKU_UsesSCMClient(t *testing.T) {
	mockRepo := setupMockRepo()
	mockSCM := setupMockSCMClient()
	mockSCM.On("GetStockBySKU", mock.Anything, "SKU-1").Return(&models.ComponentStock{SKU: "SKU-1", StockQty: 27}, nil)
	svc := NewProductionService(mockRepo, mockSCM)

	qty, err := svc.GetInventoryBalanceBySKU(context.Background(), "SKU-1")
	require.NoError(t, err)
	assert.Equal(t, 27, qty)
}

func TestProductionService_GetInventoryBalanceBySKU_RejectsUUIDSKU(t *testing.T) {
	svc := NewProductionService(setupMockRepo(), setupMockSCMClient())
	qty, err := svc.GetInventoryBalanceBySKU(context.Background(), uuid.New().String())
	require.Error(t, err)
	assert.Equal(t, 0, qty)
}

func TestProductionService_ExportInventoryCSV(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	res, err := svc.ExportInventoryCSV(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, res)
}

// ------------------------------------------------------------
// Hard: GetInventoryLedger
// ------------------------------------------------------------

// Hard: stub returns nil — callers will range over this; must be an empty slice
func TestGetInventoryLedger_ReturnsSliceNotNil(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	res, err := svc.GetInventoryLedger(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, res, "GetInventoryLedger must return an empty slice, not nil")
}

// Hard: every ledger entry must have a non-empty SKU, non-empty ID, and a valid Timestamp
func TestGetInventoryLedger_EntryFieldsAreValid(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	entries, err := svc.GetInventoryLedger(context.Background())
	require.NoError(t, err)

	for i, e := range entries {
		assert.NotEmpty(t, e.ID,
			"entry[%d]: ID must not be empty", i)
		assert.NotEmpty(t, e.SKU,
			"entry[%d]: SKU must not be empty", i)
		assert.False(t, e.Timestamp.IsZero(),
			"entry[%d]: Timestamp must not be zero — unset timestamps indicate a mapping bug", i)
	}
}

// Hard: RunningBalance must be consistent with the sum of QtyChange values up to that row
// A running balance that doesn't match the cumulative sum indicates a ledger integrity bug.
func TestGetInventoryLedger_RunningBalanceIsConsistent(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	entries, err := svc.GetInventoryLedger(context.Background())
	require.NoError(t, err)

	// Group by SKU and verify running totals independently per SKU
	type skuLedger struct {
		running int
		entries []int // index into entries slice
	}
	bySKU := make(map[string]*skuLedger)

	for i, e := range entries {
		if bySKU[e.SKU] == nil {
			bySKU[e.SKU] = &skuLedger{}
		}
		bySKU[e.SKU].running += e.QtyChange
		bySKU[e.SKU].entries = append(bySKU[e.SKU].entries, i)

		assert.Equal(t, bySKU[e.SKU].running, e.RunningBalance,
			"entry[%d] SKU=%s: RunningBalance=%d does not match cumulative QtyChange sum=%d",
			i, e.SKU, e.RunningBalance, bySKU[e.SKU].running)
	}
}

// Hard: context cancellation must propagate
func TestGetInventoryLedger_CancelledContext(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := svc.GetInventoryLedger(ctx)
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

// ------------------------------------------------------------
// Hard: GetInventoryMetrics
// ------------------------------------------------------------

// Hard: stub returns nil — a real impl must return a populated struct
func TestGetInventoryMetrics_ReturnsNonNilStruct(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	m, err := svc.GetInventoryMetrics(context.Background())
	require.NoError(t, err)
	require.NotNil(t, m, "GetInventoryMetrics must return a struct, not nil")
}

// Hard: StockAccuracy is a percentage — must be 0–100, not a 0–1 fraction
func TestGetInventoryMetrics_StockAccuracyIsBounded(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	m, err := svc.GetInventoryMetrics(context.Background())
	require.NoError(t, err)
	require.NotNil(t, m)

	assert.GreaterOrEqual(t, m.StockAccuracy, 0.0,
		"StockAccuracy must be >= 0.0")
	assert.LessOrEqual(t, m.StockAccuracy, 100.0,
		"StockAccuracy is a percentage — must not exceed 100 (got %f)", m.StockAccuracy)
}

// Hard: InventoryTurnover must not be negative — negative turnover has no physical meaning
func TestGetInventoryMetrics_TurnoverIsNonNegative(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	m, err := svc.GetInventoryMetrics(context.Background())
	require.NoError(t, err)
	require.NotNil(t, m)

	assert.GreaterOrEqual(t, m.InventoryTurnover, 0.0,
		"InventoryTurnover must not be negative")
}

// Hard: ActiveSKUs and CycleCountGaps must both be non-negative
func TestGetInventoryMetrics_CountFieldsAreNonNegative(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	m, err := svc.GetInventoryMetrics(context.Background())
	require.NoError(t, err)
	require.NotNil(t, m)

	assert.GreaterOrEqual(t, m.ActiveSKUs, 0,
		"ActiveSKUs must be >= 0")
	assert.GreaterOrEqual(t, m.CycleCountGaps, 0,
		"CycleCountGaps must be >= 0")
}

// Hard: CycleCountGaps can never exceed ActiveSKUs — a gap only exists for an active SKU
func TestGetInventoryMetrics_CycleCountGapsDoNotExceedActiveSKUs(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	m, err := svc.GetInventoryMetrics(context.Background())
	require.NoError(t, err)
	require.NotNil(t, m)

	assert.LessOrEqual(t, m.CycleCountGaps, m.ActiveSKUs,
		"CycleCountGaps (%d) cannot exceed ActiveSKUs (%d) — a gap requires an active SKU",
		m.CycleCountGaps, m.ActiveSKUs)
}

// ------------------------------------------------------------
// Hard: ExportInventoryCSV
// ------------------------------------------------------------

// Hard: stub returns nil — a real impl must return non-empty bytes
func TestExportInventoryCSV_ReturnsNonEmptyBytes(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	data, err := svc.ExportInventoryCSV(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, data, "ExportInventoryCSV must return non-empty data")
}

// Hard: CSV output must have a header row with comma-separated column names
func TestExportInventoryCSV_HasCSVHeader(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	data, err := svc.ExportInventoryCSV(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, data)

	content := string(data)
	firstLine := strings.SplitN(content, "\n", 2)[0]
	assert.Contains(t, firstLine, ",",
		"first line of CSV must be a comma-separated header row, got: %q", firstLine)
}

// Hard: CSV output must contain expected column names in the header
func TestExportInventoryCSV_HeaderContainsExpectedColumns(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	data, err := svc.ExportInventoryCSV(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, data)

	header := strings.SplitN(string(data), "\n", 2)[0]
	for _, col := range []string{"sku", "qty_change", "running_balance", "timestamp"} {
		assert.Contains(t, strings.ToLower(header), col,
			"CSV header must contain column %q", col)
	}
}

// Hard: each data row must have the same number of fields as the header
func TestExportInventoryCSV_AllRowsHaveSameColumnCount(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	data, err := svc.ExportInventoryCSV(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, data)

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) < 2 {
		return // only a header, nothing to check
	}

	headerCols := len(strings.Split(lines[0], ","))
	for i, line := range lines[1:] {
		if strings.TrimSpace(line) == "" {
			continue
		}
		cols := len(strings.Split(line, ","))
		assert.Equal(t, headerCols, cols,
			"data row %d has %d columns but header has %d — malformed CSV", i+1, cols, headerCols)
	}
}
