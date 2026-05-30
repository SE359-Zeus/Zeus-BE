package service

import (
	"context"
	"time"

	"zeus-scm-service/internal/infrastructure/observability"
	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/repository"

	"gorm.io/gorm"
)

type IShipmentService interface {
	AcquireDispatchLock(ctx context.Context, shipmentID string, operatorID string) error
	DispatchShipment(ctx context.Context, shipmentID string, operatorID string) error
}

type shipmentService struct {
	db *gorm.DB
}

type shipmentServiceRepo struct {
	repo  repository.IShipmentRepository
	stock repository.IStockRepository
}

func NewShipmentService(arg interface{}, args ...interface{}) IShipmentService {
	switch v := arg.(type) {
	case *gorm.DB:
		// support NewShipmentService(db) and NewShipmentService(db, shipmentRepo, stockRepo)
		if len(args) > 0 {
			if repoArg, ok := args[0].(repository.IShipmentRepository); ok {
				var stock repository.IStockRepository
				if len(args) > 1 {
					if r, ok := args[1].(repository.IStockRepository); ok {
						stock = r
					}
				}
				return &shipmentServiceRepo{repo: repoArg, stock: stock}
			}
		}
		return &shipmentService{db: v}
	case repository.IShipmentRepository:
		var stock repository.IStockRepository
		if len(args) > 0 {
			if r, ok := args[0].(repository.IStockRepository); ok {
				stock = r
			}
		}
		return &shipmentServiceRepo{repo: v, stock: stock}
	default:
		panic("invalid NewShipmentService usage")
	}
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

	tx := s.db.WithContext(ctx).Begin()

	shipment.Status = models.ShipmentStatusInTransit
	shipment.ShipDate = time.Now()
	if err := tx.Save(&shipment).Error; err != nil {
		tx.Rollback()
		return err
	}

	observability.DefaultRegistry.Counter(observability.MetricShipmentDispatched).Inc()
	return tx.Commit().Error
}

// repo-backed implementation
func (s *shipmentServiceRepo) AcquireDispatchLock(ctx context.Context, shipmentID string, operatorID string) error {
	shipment, err := s.repo.GetShipmentByID(ctx, shipmentID)
	if err != nil || shipment == nil {
		return ErrNotFound
	}
	if shipment.Status == models.ShipmentStatusInTransit || shipment.Status == models.ShipmentStatusDelivered {
		return ErrAlreadyLocked
	}
	return s.repo.UpdateShipmentFields(ctx, shipmentID, map[string]interface{}{"ship_date": time.Now()})
}

func (s *shipmentServiceRepo) DispatchShipment(ctx context.Context, shipmentID string, operatorID string) error {
	shipment, err := s.repo.GetShipmentByID(ctx, shipmentID)
	if err != nil || shipment == nil {
		return ErrNotFound
	}
	if shipment.Status != models.ShipmentStatusScheduled {
		return ErrInvalidTransition
	}

	shipment.Status = models.ShipmentStatusInTransit
	shipment.ShipDate = time.Now()
	err = s.repo.UpdateShipment(ctx, shipment)
	if err == nil {
		observability.DefaultRegistry.Counter(observability.MetricShipmentDispatched).Inc()
	}
	return err
}
