package sqlite

import (
	"context"
	"zeus-mrp-service/internal/models"

	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (r *sqliteMRPRepository) GetPartInventory(ctx context.Context, partID uuid.UUID) (int, error) {
	// Placeholder: actual inventory lookup should query inventory ledger or external service
	if partID == uuid.Nil {
		return 0, nil
	}
	type row struct {
		PartID string
		Qty    int
	}
	var rr row
	err := r.db.WithContext(ctx).Table("inventory_balances").Select("part_id, qty").Where("part_id = ?", partID.String()).Take(&rr).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
		return 0, err
	}
	return rr.Qty, nil
}

func (r *sqliteMRPRepository) GetInventoryTransactions(ctx context.Context) ([]models.InventoryTransactionDTO, error) {
	type row struct {
		ID             string
		SKU            string
		Type           string
		QtyChange      int
		RunningBalance int
		Location       string
		Timestamp      *string
		Operator       string
		Reference      string
	}
	var rows []row
	err := r.db.WithContext(ctx).
		Table("inventory_transactions").
		Select("id, sku, type, qty_change, running_balance, location, timestamp, operator, reference").
		Order("timestamp DESC").
		Find(&rows).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return []models.InventoryTransactionDTO{}, nil
		}
		return nil, err
	}

	res := make([]models.InventoryTransactionDTO, 0, len(rows))
	for _, rw := range rows {
		var ts time.Time
		if rw.Timestamp != nil && *rw.Timestamp != "" {
			parsed, perr := time.Parse(time.RFC3339, *rw.Timestamp)
			if perr == nil {
				ts = parsed
			}
		}
		res = append(res, models.InventoryTransactionDTO{
			ID:             rw.ID,
			SKU:            rw.SKU,
			Type:           rw.Type,
			QtyChange:      rw.QtyChange,
			RunningBalance: rw.RunningBalance,
			Location:       rw.Location,
			Timestamp:      ts,
			Operator:       rw.Operator,
			Reference:      rw.Reference,
		})
	}
	return res, nil
}

func (r *sqliteMRPRepository) GetInventoryMetrics(ctx context.Context) (*models.InventoryMetrics, error) {
	// Best-effort: attempt to compute basic metrics from transactions
	txns, err := r.GetInventoryTransactions(ctx)
	if err != nil {
		return nil, err
	}
	active := map[string]struct{}{}
	for _, t := range txns {
		active[t.SKU] = struct{}{}
	}
	m := &models.InventoryMetrics{
		ActiveSKUs: len(active),
	}
	return m, nil
}
