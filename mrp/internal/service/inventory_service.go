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
	if s.scmClient != nil {
		if txns, err := s.inventoryLedgerFromSCM(ctx); err == nil {
			return txns, nil
		}
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
	if s.scmClient != nil {
		if metrics, err := s.inventoryMetricsFromSCM(ctx); err == nil {
			return metrics, nil
		}
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
	if m.StockAccuracy > 100 {
		m.StockAccuracy = 100
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
	txns, err := s.GetInventoryLedger(ctx)
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
	if s.scmClient != nil {
		stock, err := s.scmClient.GetStockBySKU(ctx, sku)
		if err == nil {
			if stock == nil {
				return 0, nil
			}
			return stock.StockQty, nil
		}
	}
	pid, err := uuid.Parse(sku)
	if err != nil {
		return 0, fmt.Errorf("sku must be a UUID")
	}
	return s.repo.GetPartInventory(ctx, pid)
}

func (s *ProductionService) inventoryLedgerFromSCM(ctx context.Context) ([]models.InventoryTransactionDTO, error) {
	if s.scmClient == nil {
		return nil, nil
	}

	page := 1
	limit := 100
	rows := make([]models.InventoryTransactionDTO, 0)

	for {
		items, hasMore, err := s.scmClient.ListStocks(ctx, page, limit, "sku", "asc", "")
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			rows = append(rows, models.InventoryTransactionDTO{
				ID:             item.SKU,
				SKU:            item.SKU,
				Type:           "stock_snapshot",
				QtyChange:      item.StockQty,
				RunningBalance: item.StockQty,
				Location:       item.Location,
				Timestamp:      item.UpdatedAt,
				Operator:       "SCM",
				Reference:      item.Name,
			})
		}
		if !hasMore || len(items) == 0 {
			break
		}
		page++
	}

	return rows, nil
}

func (s *ProductionService) inventoryMetricsFromSCM(ctx context.Context) (*models.InventoryMetrics, error) {
	if s.scmClient == nil {
		return nil, nil
	}

	page := 1
	limit := 100
	activeSKUs := 0
	cycleCountGaps := 0

	for {
		items, hasMore, err := s.scmClient.ListStocks(ctx, page, limit, "sku", "asc", "")
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			activeSKUs++
			if item.StockQty < item.ReorderPoint {
				cycleCountGaps++
			}
		}
		if !hasMore || len(items) == 0 {
			break
		}
		page++
	}

	return &models.InventoryMetrics{
		ActiveSKUs:        activeSKUs,
		StockAccuracy:     100,
		InventoryTurnover: 0,
		CycleCountGaps:    cycleCountGaps,
	}, nil
}
