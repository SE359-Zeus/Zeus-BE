package service

import (
	"context"
	"time"

	"zeus-scm-service/internal/models"

	"gorm.io/gorm"
)

type IShipmentService interface {
	AcquireDispatchLock(ctx context.Context, shipmentID string, operatorID string) error
	DispatchShipment(ctx context.Context, shipmentID string, operatorID string) error
}

type shipmentService struct {
	db *gorm.DB
}

func NewShipmentService(db *gorm.DB) IShipmentService {
	return &shipmentService{db: db}
}

func (s *shipmentService) AcquireDispatchLock(ctx context.Context, shipmentID string, operatorID string) error {
	var shipment models.Shipment
	if err := s.db.WithContext(ctx).First(&shipment, "id = ?", shipmentID).Error; err != nil {
		return ErrNotFound
	}
	if shipment.Status == models.ShipmentStatusInTransit || shipment.Status == models.ShipmentStatusDelivered {
		return ErrAlreadyLocked
	}
	return s.db.WithContext(ctx).Model(&shipment).Updates(map[string]interface{}{
		"ship_date": time.Now(),
	}).Error
}

func (s *shipmentService) DispatchShipment(ctx context.Context, shipmentID string, operatorID string) error {
	var shipment models.Shipment
	if err := s.db.WithContext(ctx).First(&shipment, "id = ?", shipmentID).Error; err != nil {
		return ErrNotFound
	}
	if shipment.Status != models.ShipmentStatusScheduled {
		return ErrInvalidTransition
	}

	var items []models.ShipmentItem
	if err := s.db.WithContext(ctx).Where("shipment_id = ?", shipmentID).Find(&items).Error; err != nil {
		return err
	}

	tx := s.db.WithContext(ctx).Begin()

	for _, item := range items {
		var stock models.ComponentStock
		if err := tx.First(&stock, "sku = ?", item.SKU).Error; err != nil {
			tx.Rollback()
			return err
		}
		if stock.StockQty < item.Qty {
			tx.Rollback()
			return ErrInsufficientDeficit
		}
		stock.StockQty -= item.Qty
		if err := tx.Save(&stock).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	shipment.Status = models.ShipmentStatusInTransit
	now := time.Now()
	shipment.ShipDate = now
	if err := tx.Save(&shipment).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}
