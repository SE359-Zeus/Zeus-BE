package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strings"
	"time"

	"zeus-mrp-service/internal/infrastructure/observability"
	"zeus-mrp-service/internal/models"

	"github.com/google/uuid"
)

func (s *ProductionService) GetInventoryLedger(ctx context.Context) ([]models.InventoryTransactionDTO, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	observability.DefaultRegistry.Counter(observability.MetricInventoryLedgerReads).Inc()
	if s.scmClient != nil {
		txns, err := s.inventoryLedgerFromSCM(ctx)
		if err != nil {
			return nil, fmt.Errorf("SCM inventory ledger unavailable: %w", err)
		}
		return txns, nil
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
		metrics, err := s.inventoryMetricsFromSCM(ctx)
		if err != nil {
			return nil, fmt.Errorf("SCM inventory metrics unavailable: %w", err)
		}
		return metrics, nil
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
	txnID = strings.TrimSpace(txnID)
	if txnID == "" {
		return nil, fmt.Errorf("transaction ID is required")
	}

	if s.scmClient != nil {
		entry, err := s.scmClient.GetInventoryTransactionByID(ctx, txnID)
		if err != nil {
			return nil, fmt.Errorf("SCM transaction lookup unavailable: %w", err)
		}
		if entry == nil {
			return nil, fmt.Errorf("transaction not found")
		}
		dto := &models.InventoryTransactionDTO{
			ID:             entry.ID,
			SKU:            entry.SKU,
			Type:           entry.Type,
			QtyChange:      entry.QtyChange,
			RunningBalance: entry.RunningBalance,
			Location:       entry.Location,
			Timestamp:      entry.CreatedAt,
			Operator:       entry.OperatorName,
			Reference:      entry.Reference,
		}
		return dto, nil
	}

	// Fallback to local repo
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
	sku = strings.TrimSpace(sku)
	if sku == "" {
		return 0, fmt.Errorf("sku cannot be empty")
	}
	if _, err := uuid.Parse(sku); err == nil {
		return 0, fmt.Errorf("sku must be a part number, not a UUID")
	}
	if s.scmClient != nil {
		stock, err := s.scmClient.GetStockBySKU(ctx, sku)
		if err == nil {
			if stock == nil {
				return 0, nil
			}
			return stock.StockQty, nil
		}
	}
	return 0, fmt.Errorf("SCM client is required to resolve inventory by SKU")
}

func (s *ProductionService) inventoryLedgerFromSCM(ctx context.Context) ([]models.InventoryTransactionDTO, error) {
	if s.scmClient == nil {
		return nil, nil
	}

	page := 1
	limit := 100
	rows := make([]models.InventoryTransactionDTO, 0)

	for {
		entries, hasMore, err := s.scmClient.GetInventoryLedger(ctx, page, limit, "created_at", "desc", "", "")
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			rows = append(rows, models.InventoryTransactionDTO{
				ID:             e.ID,
				SKU:            e.SKU,
				Type:           e.Type,
				QtyChange:      e.QtyChange,
				RunningBalance: e.RunningBalance,
				Location:       e.Location,
				Timestamp:      e.CreatedAt,
				Operator:       e.OperatorName,
				Reference:      e.Reference,
			})
		}
		if !hasMore || len(entries) == 0 {
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

	scmMetrics, err := s.scmClient.GetInventoryMetrics(ctx)
	if err != nil {
		return nil, err
	}

	cycleCountGaps := scmMetrics.LowStock + scmMetrics.OutOfStock

	return &models.InventoryMetrics{
		ActiveSKUs:        scmMetrics.TotalSKUs,
		StockAccuracy:     100,
		InventoryTurnover: 0,
		CycleCountGaps:    cycleCountGaps,
	}, nil
}
