package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"time"
	"zeus-mrp-service/internal/models"

	"github.com/google/uuid"
)

func (s *ProductionService) GetInventoryLedger(ctx context.Context) ([]models.InventoryTransactionDTO, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	txns, err := s.repo.GetInventoryTransactions(ctx)
	if err != nil {
		return nil, err
	}
	if txns == nil {
		return []models.InventoryTransactionDTO{}, nil
	}
	return txns, nil
}

func (s *ProductionService) GetInventoryMetrics(ctx context.Context) (*models.InventoryMetrics, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m, err := s.repo.GetInventoryMetrics(ctx)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return &models.InventoryMetrics{}, nil
	}
	// enforce bounds for safety
	if m.StockAccuracy < 0 {
		m.StockAccuracy = 0
	}
	if m.StockAccuracy > 1 {
		m.StockAccuracy = 1
	}
	if m.ActiveSKUs < 0 {
		m.ActiveSKUs = 0
	}
	if m.CycleCountGaps < 0 {
		m.CycleCountGaps = 0
	}
	return m, nil
}

func (s *ProductionService) ExportInventoryCSV(ctx context.Context) ([]byte, error) {
	txns, err := s.GetInventoryLedger(ctx)
	if err != nil {
		return nil, err
	}

	buf := &bytes.Buffer{}
	w := csv.NewWriter(buf)
	// header
	_ = w.Write([]string{"id", "sku", "type", "qty_change", "running_balance", "location", "timestamp", "operator", "reference"})
	for _, t := range txns {
		_ = w.Write([]string{t.ID, t.SKU, t.Type, fmt.Sprintf("%d", t.QtyChange), fmt.Sprintf("%d", t.RunningBalance), t.Location, t.Timestamp.Format(time.RFC3339), t.Operator, t.Reference})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *ProductionService) GetInventoryTransactionByID(ctx context.Context, txnID string) (*models.InventoryTransactionDTO, error) {
	txns, err := s.repo.GetInventoryTransactions(ctx)
	if err != nil {
		return nil, err
	}
	for _, t := range txns {
		if t.ID == txnID {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("transaction not found")
}

func (s *ProductionService) GetInventoryBalanceBySKU(ctx context.Context, sku string) (int, error) {
	pid, err := uuid.Parse(sku)
	if err != nil {
		return 0, fmt.Errorf("sku must be a UUID")
	}
	return s.repo.GetPartInventory(ctx, pid)
}
