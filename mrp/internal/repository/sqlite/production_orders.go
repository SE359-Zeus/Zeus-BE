package sqlite

import (
	"context"
	"fmt"
	"time"
	"zeus-mrp-service/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CreateProductionOrder inserts a production order record.
func (r *sqliteMRPRepository) CreateProductionOrder(ctx context.Context, order *models.ProductionOrder) error {
	if order == nil {
		return fmt.Errorf("order is nil")
	}

	type rec struct {
		ID               string  `gorm:"column:id"`
		ProductModelCode string  `gorm:"column:product_model_code"`
		TargetQuantity   int     `gorm:"column:target_quantity"`
		Status           string  `gorm:"column:status"`
		ScheduledAt      *string `gorm:"column:scheduled_at"`
		CreatedAt        string  `gorm:"column:created_at"`
	}

	var sched *string
	if !order.ScheduledAt.IsZero() {
		s := order.ScheduledAt.UTC().Format(time.RFC3339)
		sched = &s
	}

	recRow := rec{
		ID:               order.ID.String(),
		ProductModelCode: order.ProductModelCode,
		TargetQuantity:   order.TargetQuantity,
		Status:           string(order.Status),
		ScheduledAt:      sched,
		CreatedAt:        order.CreatedAt.UTC().Format(time.RFC3339),
	}

	return r.db.WithContext(ctx).Table("production_orders").Create(&recRow).Error
}

func (r *sqliteMRPRepository) GetProductionOrder(ctx context.Context, id uuid.UUID) (*models.ProductionOrder, error) {
	if id == uuid.Nil {
		return nil, nil
	}

	type row struct {
		ID               string
		ProductModelCode string
		TargetQuantity   int
		Status           string
		ScheduledAt      *string
		CreatedAt        *string
	}

	var dbRow row
	err := r.db.WithContext(ctx).
		Table("production_orders").
		Select("id, product_model_code, target_quantity, status, scheduled_at, created_at").
		Where("id = ?", id.String()).
		Take(&dbRow).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	parsedID, err := uuid.Parse(dbRow.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid production order id in database: %w", err)
	}

	var sched time.Time
	if dbRow.ScheduledAt != nil && *dbRow.ScheduledAt != "" {
		if t, err := time.Parse(time.RFC3339, *dbRow.ScheduledAt); err == nil {
			sched = t
		}
	}
	var created time.Time
	if dbRow.CreatedAt != nil && *dbRow.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339, *dbRow.CreatedAt); err == nil {
			created = t
		}
	}

	return &models.ProductionOrder{
		ID:               parsedID,
		ProductModelCode: dbRow.ProductModelCode,
		TargetQuantity:   dbRow.TargetQuantity,
		Status:           models.ProductionOrderStatus(dbRow.Status),
		ScheduledAt:      sched,
		CreatedAt:        created,
	}, nil
}

func (r *sqliteMRPRepository) GetOpenProductionOrders(ctx context.Context) ([]models.ProductionOrder, error) {
	type row struct {
		ID               string
		ProductModelCode string
		TargetQuantity   int
		Status           string
		ScheduledAt      *string
		CreatedAt        *string
	}

	var rows []row
	err := r.db.WithContext(ctx).
		Table("production_orders").
		Select("id, product_model_code, target_quantity, status, scheduled_at, created_at").
		Where("status IN ?", []string{string(models.StatusClearToBuild), string(models.StatusPartial), string(models.StatusShortage)}).
		Order("created_at DESC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	orders := make([]models.ProductionOrder, 0, len(rows))
	for _, row := range rows {
		id, err := uuid.Parse(row.ID)
		if err != nil {
			continue
		}

		var sched time.Time
		if row.ScheduledAt != nil && *row.ScheduledAt != "" {
			if t, err := time.Parse(time.RFC3339, *row.ScheduledAt); err == nil {
				sched = t
			}
		}
		var created time.Time
		if row.CreatedAt != nil && *row.CreatedAt != "" {
			if t, err := time.Parse(time.RFC3339, *row.CreatedAt); err == nil {
				created = t
			}
		}

		orders = append(orders, models.ProductionOrder{
			ID:               id,
			ProductModelCode: row.ProductModelCode,
			TargetQuantity:   row.TargetQuantity,
			Status:           models.ProductionOrderStatus(row.Status),
			ScheduledAt:      sched,
			CreatedAt:        created,
		})
	}

	return orders, nil
}

func (r *sqliteMRPRepository) UpdateProductionOrderStatus(ctx context.Context, id uuid.UUID, status models.ProductionOrderStatus) error {
	if id == uuid.Nil {
		return fmt.Errorf("id is required")
	}
	return r.db.WithContext(ctx).Table("production_orders").Where("id = ?", id.String()).Update("status", string(status)).Error
}

func (r *sqliteMRPRepository) DeleteProductionOrder(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return fmt.Errorf("id is required")
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("shortage_logs").Where("production_order_id = ?", id.String()).Delete(nil).Error; err != nil {
			return err
		}
		if err := tx.Table("production_orders").Where("id = ?", id.String()).Delete(nil).Error; err != nil {
			return err
		}
		return nil
	})
}
