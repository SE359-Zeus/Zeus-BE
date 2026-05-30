package sqlite

import (
	"context"

	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/pagination"
	"zeus-scm-service/internal/repository"

	"gorm.io/gorm"
)

type shipmentRepository struct {
	db *gorm.DB
}

func NewShipmentRepository(db *gorm.DB) repository.IShipmentRepository {
	return &shipmentRepository{db: db}
}

func (r *shipmentRepository) GetShipmentByID(ctx context.Context, id string) (*models.Shipment, error) {
	var s models.Shipment
	if err := r.db.WithContext(ctx).Preload("Items").Preload("Supplier").First(&s, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *shipmentRepository) CreateShipmentItem(ctx context.Context, item *models.ShipmentItem) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *shipmentRepository) UpdateShipment(ctx context.Context, shipment *models.Shipment) error {
	return r.db.WithContext(ctx).Save(shipment).Error
}

func (r *shipmentRepository) UpdateShipmentFields(ctx context.Context, id string, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.Shipment{}).Where("id = ?", id).Updates(fields).Error
}

func (r *shipmentRepository) GetShipmentItemsByShipmentID(ctx context.Context, shipmentID string) ([]models.ShipmentItem, error) {
	var items []models.ShipmentItem
	if err := r.db.WithContext(ctx).Where("shipment_id = ?", shipmentID).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *shipmentRepository) ListShipments(ctx context.Context, status string, params pagination.Params) ([]models.Shipment, *pagination.Meta, error) {
	query := r.db.WithContext(ctx).Model(&models.Shipment{}).Preload("Items").Preload("Supplier")
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var shipments []models.Shipment
	meta, err := pagination.Paginate(query, params, &shipments, "created_at", "updated_at", "id", "status", "ship_date")
	if err != nil {
		return nil, nil, err
	}
	return shipments, meta, nil
}

func (r *shipmentRepository) FindAllShipments(ctx context.Context) ([]models.Shipment, error) {
	var shipments []models.Shipment
	if err := r.db.WithContext(ctx).Preload("Items").Preload("Supplier").Order("created_at DESC").Find(&shipments).Error; err != nil {
		return nil, err
	}
	return shipments, nil
}

func (r *shipmentRepository) CreateShipment(ctx context.Context, shipment *models.Shipment) error {
	return r.db.WithContext(ctx).Create(shipment).Error
}

func (r *shipmentRepository) GetShipmentMetrics(ctx context.Context) (total int64, inTransit int64, delayed int64, onTimeRate float64, err error) {
	err = r.db.WithContext(ctx).Model(&models.Shipment{}).Where("deleted_at IS NULL").Count(&total).Error
	if err != nil {
		return
	}

	err = r.db.WithContext(ctx).Model(&models.Shipment{}).Where("status = ?", models.ShipmentStatusInTransit).Count(&inTransit).Error
	if err != nil {
		return
	}

	err = r.db.WithContext(ctx).Model(&models.Shipment{}).Where("status = ?", models.ShipmentStatusDelayed).Count(&delayed).Error
	if err != nil {
		return
	}

	// On-time rate = (Delivered on-time / total delivered) * 100
	// We approximate: delivered before ETA = on-time
	var totalDelivered int64
	err = r.db.WithContext(ctx).Model(&models.Shipment{}).Where("status = ?", models.ShipmentStatusDelivered).Count(&totalDelivered).Error
	if err != nil {
		return
	}

	if totalDelivered == 0 {
		onTimeRate = 100.0
		return
	}

	var onTime int64
	err = r.db.WithContext(ctx).Model(&models.Shipment{}).
		Where("status = ? AND updated_at <= eta", models.ShipmentStatusDelivered).
		Count(&onTime).Error
	if err != nil {
		return
	}

	onTimeRate = float64(onTime) / float64(totalDelivered) * 100.0
	return
}

