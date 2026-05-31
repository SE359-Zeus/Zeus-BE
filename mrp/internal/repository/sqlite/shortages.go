package sqlite

import (
	"context"
	"fmt"
	"zeus-mrp-service/internal/models"

	"github.com/google/uuid"
)

func (r *sqliteMRPRepository) CreateShortageLog(ctx context.Context, log *models.ShortageLog) error {
	if log == nil {
		return fmt.Errorf("shortage log is nil")
	}
	statusID, status := resolutionStatusDetails(log.ResolutionStatus)
	if log.ResolutionStatus == "" {
		statusID = 1
		status = models.ResolutionStatusPlanned
	}

	type shortageRecord struct {
		ID                 string `gorm:"column:id"`
		ProductionOrderID  string `gorm:"column:production_order_id"`
		PartID             string `gorm:"column:part_id"`
		ShortageQty        int    `gorm:"column:shortage_qty"`
		ResolutionStatusID int    `gorm:"column:resolution_status_id"`
		ResolutionStatus   string `gorm:"column:resolution_status"`
	}

	rec := shortageRecord{
		ID:                 log.ID.String(),
		ProductionOrderID:  log.ProductionOrderID.String(),
		PartID:             log.PartID.String(),
		ShortageQty:        log.ShortageQty,
		ResolutionStatusID: statusID,
		ResolutionStatus:   status,
	}

	return r.db.WithContext(ctx).Table("shortage_logs").Create(&rec).Error
}

func (r *sqliteMRPRepository) GetShortagesByOrderID(ctx context.Context, orderID uuid.UUID) ([]models.ShortageLog, error) {
	if orderID == uuid.Nil {
		return []models.ShortageLog{}, nil
	}

	type row struct {
		ID                 string
		ProductionOrderID  string
		PartID             string
		ShortageQty        int
		ResolutionStatusID int
		ResolutionStatus   string
	}

	var rows []row
	err := r.db.WithContext(ctx).
		Table("shortage_logs").
		Select("id, production_order_id, part_id, shortage_qty, resolution_status_id, resolution_status").
		Where("production_order_id = ?", orderID.String()).
		Order("id ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	logs := make([]models.ShortageLog, 0, len(rows))
	for _, row := range rows {
		id, err := uuid.Parse(row.ID)
		if err != nil {
			continue
		}
		productionOrderID, err := uuid.Parse(row.ProductionOrderID)
		if err != nil {
			continue
		}
		partID, err := uuid.Parse(row.PartID)
		if err != nil {
			continue
		}

		logs = append(logs, models.ShortageLog{
			ID:                 id,
			ProductionOrderID:  productionOrderID,
			PartID:             partID,
			ShortageQty:        row.ShortageQty,
			ResolutionStatusID: row.ResolutionStatusID,
			ResolutionStatus:   row.ResolutionStatus,
		})
	}

	return logs, nil
}

func (r *sqliteMRPRepository) GetAggregatedShortages(ctx context.Context) ([]models.BOMExplosionResult, error) {
	type row struct {
		PartID           string
		TotalRequiredQty int
	}

	var rows []row
	err := r.db.WithContext(ctx).
		Table("shortage_logs").
		Select("part_id, SUM(shortage_qty) AS total_required_qty").
		Group("part_id").
		Order("part_id ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	results := make([]models.BOMExplosionResult, 0, len(rows))
	for _, row := range rows {
		partID, err := uuid.Parse(row.PartID)
		if err != nil {
			continue
		}

		results = append(results, models.BOMExplosionResult{
			PartID:           partID,
			TotalRequiredQty: row.TotalRequiredQty,
			AvailableQty:     0,
			IsShortage:       row.TotalRequiredQty > 0,
		})
	}

	return results, nil
}

func resolutionStatusDetails(status string) (int, string) {
	switch status {
	case models.ResolutionStatusPartial:
		return 2, models.ResolutionStatusPartial
	case models.ResolutionStatusShortage:
		return 3, models.ResolutionStatusShortage
	case models.ResolutionStatusReadyToBuild:
		return 4, models.ResolutionStatusReadyToBuild
	case models.ResolutionStatusPlanned:
		return 1, models.ResolutionStatusPlanned
	default:
		return 1, models.ResolutionStatusPlanned
	}
}

func (r *sqliteMRPRepository) UpdateShortageLog(ctx context.Context, log *models.ShortageLog) error {
	if log == nil {
		return fmt.Errorf("log is nil")
	}
	statusID, status := resolutionStatusDetails(log.ResolutionStatus)
	return r.db.WithContext(ctx).Table("shortage_logs").
		Where("id = ?", log.ID.String()).
		Updates(map[string]any{
			"shortage_qty":         log.ShortageQty,
			"resolution_status_id": statusID,
			"resolution_status":   status,
		}).Error
}

func (r *sqliteMRPRepository) DeleteShortageLog(ctx context.Context, orderID uuid.UUID, partID uuid.UUID) error {
	if orderID == uuid.Nil || partID == uuid.Nil {
		return fmt.Errorf("orderID and partID must not be Nil")
	}
	return r.db.WithContext(ctx).Table("shortage_logs").
		Where("production_order_id = ? AND part_id = ?", orderID.String(), partID.String()).
		Delete(nil).Error
}
