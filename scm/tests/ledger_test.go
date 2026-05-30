package tests

import (
	"context"
	"testing"
	"time"

	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/pagination"
	sqliteRepo "zeus-scm-service/internal/repository/sqlite"
	"zeus-scm-service/internal/service"

	"github.com/stretchr/testify/assert"
)

func TestLedgerIntegration(t *testing.T) {
	db := setupTestDB()

	// Crucial: AutoMigrate should succeed with models.InventoryLedger
	// maps to TableName() "inventory_ledger"
	err := db.AutoMigrate(&models.InventoryLedger{})
	assert.NoError(t, err)

	repo := sqliteRepo.NewLedgerRepository(db)
	svc := service.NewLedgerService(repo)
	ctx := context.WithValue(context.Background(), "full_name", "Alex Operator")

	// 1. Record first entry
	entry1, err := svc.RecordEntry(ctx, "SKU-TEST-1", models.LedgerTxnTypeIN, 100, "operator-1", "ref-1", models.LedgerRefInitial, "id-1")
	assert.NoError(t, err)
	assert.NotNil(t, entry1)
	assert.Equal(t, 100, entry1.RunningBalance)
	assert.Equal(t, "Alex Operator", entry1.OperatorName)

	time.Sleep(10 * time.Millisecond)

	// 2. Record second entry
	entry2, err := svc.RecordEntry(ctx, "SKU-TEST-1", models.LedgerTxnTypeOUT, -20, "operator-1", "ref-2", models.LedgerRefShipment, "id-2")
	assert.NoError(t, err)
	assert.NotNil(t, entry2)
	assert.Equal(t, 80, entry2.RunningBalance) // 100 - 20 = 80
	assert.Equal(t, "Alex Operator", entry2.OperatorName)

	// 3. Get latest balance
	bal, err := repo.GetLatestBalance(ctx, "SKU-TEST-1")
	assert.NoError(t, err)
	assert.Equal(t, 80, bal)

	// 4. Get by ID
	fetched, err := svc.GetEntryByID(ctx, entry1.ID)
	assert.NoError(t, err)
	assert.NotNil(t, fetched)
	assert.Equal(t, "SKU-TEST-1", fetched.SKU)
	assert.Equal(t, 100, fetched.QtyChange)
	assert.Equal(t, "Alex Operator", fetched.OperatorName)

	// 5. List entries
	params := pagination.Params{Page: 1, Limit: 10}
	entries, meta, err := svc.ListEntries(ctx, params, "", "")
	assert.NoError(t, err)
	assert.Len(t, entries, 2)
	assert.Equal(t, int64(2), meta.TotalRows)

	// List with txnType filter
	entriesFiltered, _, err := svc.ListEntries(ctx, params, string(models.LedgerTxnTypeOUT), "")
	assert.NoError(t, err)
	assert.Len(t, entriesFiltered, 1)
	assert.Equal(t, "SKU-TEST-1", entriesFiltered[0].SKU)
	assert.Equal(t, -20, entriesFiltered[0].QtyChange)
}
