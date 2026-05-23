package sqlite

import (
	"context"
	"time"

	"zeus-sales-service/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type salesOrderItemRecord struct {
	ID           string    `gorm:"primaryKey;column:id"`
	OrderID      string    `gorm:"column:order_id;index"`
	SKU          string    `gorm:"column:sku;index"`
	RequestedQty int       `gorm:"column:requested_qty"`
	AllocatedQty int       `gorm:"column:allocated_qty"`
	UnitPrice    float64   `gorm:"column:unit_price"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (salesOrderItemRecord) TableName() string { return "sales_order_items" }

func orderItemRecordFromModel(item *models.SalesOrderItem) *salesOrderItemRecord {
	return &salesOrderItemRecord{
		ID:           item.ID.String(),
		OrderID:      item.OrderID.String(),
		SKU:          item.SKU,
		RequestedQty: item.RequestedQty,
		AllocatedQty: item.AllocatedQty,
		UnitPrice:    item.UnitPrice,
		CreatedAt:    item.CreatedAt,
		UpdatedAt:    item.UpdatedAt,
	}
}

func (record salesOrderItemRecord) toModel() models.SalesOrderItem {
	parsedID, _ := uuid.Parse(record.ID)
	orderID, _ := uuid.Parse(record.OrderID)
	return models.SalesOrderItem{
		ID:           parsedID,
		OrderID:      orderID,
		SKU:          record.SKU,
		RequestedQty: record.RequestedQty,
		AllocatedQty: record.AllocatedQty,
		UnitPrice:    record.UnitPrice,
		CreatedAt:    record.CreatedAt,
		UpdatedAt:    record.UpdatedAt,
	}
}

func (repo *Repository) CreateOrderItem(ctx context.Context, item *models.SalesOrderItem) error {
	if item.ID == uuid.Nil {
		item.ID = uuid.New()
	}
	now := time.Now().UTC()
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	item.UpdatedAt = now
	return repo.db.WithContext(ctx).Create(orderItemRecordFromModel(item)).Error
}

func (repo *Repository) GetOrderItems(ctx context.Context, orderID uuid.UUID) ([]models.SalesOrderItem, error) {
	var records []salesOrderItemRecord
	if err := repo.db.WithContext(ctx).Where("order_id = ?", orderID.String()).Order("created_at ASC").Find(&records).Error; err != nil {
		return nil, err
	}
	items := make([]models.SalesOrderItem, 0, len(records))
	for _, record := range records {
		items = append(items, record.toModel())
	}
	return items, nil
}

func (repo *Repository) ReplaceOrderItems(ctx context.Context, orderID uuid.UUID, items []models.SalesOrderItem) error {
	return repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("order_id = ?", orderID.String()).Delete(&salesOrderItemRecord{}).Error; err != nil {
			return err
		}
		for _, item := range items {
			if item.ID == uuid.Nil {
				item.ID = uuid.New()
			}
			if item.CreatedAt.IsZero() {
				item.CreatedAt = time.Now().UTC()
			}
			item.UpdatedAt = item.CreatedAt
			record := orderItemRecordFromModel(&item)
			record.OrderID = orderID.String()
			if err := tx.Create(record).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
