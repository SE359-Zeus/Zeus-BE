package sqlite

// inventory.go — MRP read-only inventory proxy stubs.
//
// Inventory data (transactions, balances) is owned by the SCM service.
// MRP must not query SCM's tables directly; it must call the SCM HTTP API.
// These stubs satisfy the DbRepository interface while that HTTP client
// is being wired up. Replace each method body with the real HTTP call.

import (
	"context"
	"zeus-mrp-service/internal/models"

	"github.com/google/uuid"
)

// GetPartInventory returns the on-hand quantity for a part.
// TODO: replace stub with HTTP call to SCM inventory API.
func (r *sqliteMRPRepository) GetPartInventory(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

// GetInventoryTransactions returns all stock movements from the SCM ledger.
// TODO: replace stub with HTTP call to SCM inventory API.
func (r *sqliteMRPRepository) GetInventoryTransactions(_ context.Context) ([]models.InventoryTransactionDTO, error) {
	return []models.InventoryTransactionDTO{}, nil
}

// GetInventoryMetrics returns computed inventory KPIs.
// TODO: replace stub with HTTP call to SCM inventory API.
func (r *sqliteMRPRepository) GetInventoryMetrics(_ context.Context) (*models.InventoryMetrics, error) {
	return &models.InventoryMetrics{}, nil
}
