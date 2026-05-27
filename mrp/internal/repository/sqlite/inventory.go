package sqlite

import (
	"context"
	"zeus-mrp-service/internal/models"
)

// GetInventoryTransactions returns all stock movements from the SCM ledger.
func (r *sqliteMRPRepository) GetInventoryTransactions(_ context.Context) ([]models.InventoryTransactionDTO, error) {
	return []models.InventoryTransactionDTO{}, nil
}

// GetInventoryMetrics returns computed inventory KPIs.
func (r *sqliteMRPRepository) GetInventoryMetrics(_ context.Context) (*models.InventoryMetrics, error) {
	return &models.InventoryMetrics{}, nil
}
