package sqlite

import (
	"context"

	"zeus-scm-service/internal/models"
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
	if err := r.db.WithContext(ctx).First(&s, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &s, nil
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
