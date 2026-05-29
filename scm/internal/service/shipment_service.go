package service

import (
	"context"
	"time"

	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/repository"

	"gorm.io/gorm"
)

type IShipmentService interface {
	AcquireDispatchLock(ctx context.Context, shipmentID string, operatorID string) error
	DispatchShipment(ctx context.Context, shipmentID string, operatorID string) error
}

type shipmentService struct {
	db        *gorm.DB
	ledgerSvc ILedgerService
}

type shipmentServiceRepo struct {
	repo      repository.IShipmentRepository
	stock     repository.IStockRepository
	ledgerSvc ILedgerService
}

func NewShipmentService(arg interface{}, args ...interface{}) IShipmentService {
	var ledgerSvc ILedgerService
	for _, a := range args {
		if ls, ok := a.(ILedgerService); ok {
			ledgerSvc = ls
		}
	}

	switch v := arg.(type) {
	case *gorm.DB:
		if len(args) > 0 {
			if repoArg, ok := args[0].(repository.IShipmentRepository); ok {
				var stock repository.IStockRepository
				if len(args) > 1 {
					if r, ok := args[1].(repository.IStockRepository); ok {
						stock = r
					}
				}
				return &shipmentServiceRepo{repo: repoArg, stock: stock, ledgerSvc: ledgerSvc}
			}
		}
		return &shipmentService{db: v, ledgerSvc: ledgerSvc}
	case repository.IShipmentRepository:
		var stock repository.IStockRepository
		if len(args) > 0 {
			if r, ok := args[0].(repository.IStockRepository); ok {
				stock = r
			}
		}
		return &shipmentServiceRepo{repo: v, stock: stock, ledgerSvc: ledgerSvc}
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

		if s.ledgerSvc != nil {
			if _, lerr := s.ledgerSvc.RecordEntry(ctx, item.SKU, models.LedgerTxnTypeOUT, -item.Qty, operatorID, shipment.ID, models.LedgerRefShipment, shipment.ID); lerr != nil {
				s.ledgerSvc.RecordEntry(context.Background(), item.SKU, models.LedgerTxnTypeOUT, -item.Qty, operatorID, shipment.ID, models.LedgerRefShipment, shipment.ID)
			}
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

	items, err := s.repo.GetShipmentItemsByShipmentID(ctx, shipmentID)
	if err != nil {
		return err
	}

	// emulate transaction by applying changes and failing early on errors
	for _, item := range items {
		stock, err := s.stock.GetStockBySKU(ctx, item.SKU)
		if err != nil {
			return err
		}
		if stock.StockQty < item.Qty {
			return ErrInsufficientDeficit
		}
		stock.StockQty -= item.Qty
		if err := s.stock.SaveStock(ctx, stock); err != nil {
			return err
		}

		if s.ledgerSvc != nil {
			if _, lerr := s.ledgerSvc.RecordEntry(ctx, item.SKU, models.LedgerTxnTypeOUT, -item.Qty, operatorID, shipmentID, models.LedgerRefShipment, shipmentID); lerr != nil {
				s.ledgerSvc.RecordEntry(context.Background(), item.SKU, models.LedgerTxnTypeOUT, -item.Qty, operatorID, shipmentID, models.LedgerRefShipment, shipmentID)
			}
		}
	}

	shipment.Status = models.ShipmentStatusInTransit
	shipment.ShipDate = time.Now()
	return s.repo.UpdateShipment(ctx, shipment)
}
