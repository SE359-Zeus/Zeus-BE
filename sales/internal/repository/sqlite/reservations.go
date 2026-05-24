package sqlite

import (
	"context"
	"time"

	"zeus-sales-service/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type inventoryReservationItemRecord struct {
	ID            string    `gorm:"primaryKey;column:id"`
	ReservationID string    `gorm:"column:reservation_id;index"`
	SKU           string    `gorm:"column:sku;index"`
	Quantity      int       `gorm:"column:quantity"`
	CreatedAt     time.Time `gorm:"column:created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at"`
}

type inventoryReservationRecord struct {
	ID         string                           `gorm:"primaryKey;column:id"`
	OrderID    string                           `gorm:"column:order_id;uniqueIndex"`
	ReservedAt time.Time                        `gorm:"column:reserved_at"`
	Items      []inventoryReservationItemRecord `gorm:"foreignKey:ReservationID;references:ID"`
}

func (inventoryReservationItemRecord) TableName() string { return "inventory_reservation_items" }
func (inventoryReservationRecord) TableName() string     { return "inventory_reservations" }

func reservationRecordFromModel(res *models.InventoryReservation) *inventoryReservationRecord {
	record := &inventoryReservationRecord{
		ID:         res.ID.String(),
		OrderID:    res.OrderID.String(),
		ReservedAt: res.ReservedAt,
	}
	items := make([]inventoryReservationItemRecord, 0, len(res.Items))
	for _, it := range res.Items {
		items = append(items, inventoryReservationItemRecord{
			ID:            uuid.NewSHA1(uuid.NameSpaceURL, []byte(res.ID.String()+":"+it.SKU)).String(),
			ReservationID: res.ID.String(),
			SKU:           it.SKU,
			Quantity:      it.Quantity,
			CreatedAt:     res.ReservedAt,
			UpdatedAt:     res.ReservedAt,
		})
	}
	record.Items = items
	return record
}

func (r inventoryReservationRecord) toModel() models.InventoryReservation {
	parsedID, _ := uuid.Parse(r.ID)
	orderID, _ := uuid.Parse(r.OrderID)
	items := make([]models.ReservationItem, 0, len(r.Items))
	for _, it := range r.Items {
		items = append(items, models.ReservationItem{SKU: it.SKU, Quantity: it.Quantity})
	}
	return models.InventoryReservation{
		ID:         parsedID,
		OrderID:    orderID,
		Items:      items,
		ReservedAt: r.ReservedAt,
	}
}

func (repo *Repository) CreateReservation(ctx context.Context, reservation *models.InventoryReservation) error {
	if reservation.ID == uuid.Nil {
		reservation.ID = uuid.New()
	}
	if reservation.ReservedAt.IsZero() {
		reservation.ReservedAt = time.Now().UTC()
	}
	rec := reservationRecordFromModel(reservation)
	return repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(rec).Error; err != nil {
			return err
		}
		return nil
	})
}

func (repo *Repository) GetReservation(ctx context.Context, orderID uuid.UUID) (*models.InventoryReservation, error) {
	var record inventoryReservationRecord
	if err := repo.db.WithContext(ctx).Preload("Items").First(&record, "order_id = ?", orderID.String()).Error; err != nil {
		return nil, mapRecordError(err)
	}
	model := record.toModel()
	return &model, nil
}

func (repo *Repository) DeleteReservation(ctx context.Context, orderID uuid.UUID) error {
	return repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("order_id = ?", orderID.String()).Delete(&inventoryReservationRecord{}).Error; err != nil {
			return err
		}
		if err := tx.Where("reservation_id IN (SELECT id FROM inventory_reservations WHERE order_id = ?)", orderID.String()).Delete(&inventoryReservationItemRecord{}).Error; err != nil {
			return err
		}
		return nil
	})
}
