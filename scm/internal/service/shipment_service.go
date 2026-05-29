package service

import (
	"context"
	"fmt"
	"time"

	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/pagination"
	"zeus-scm-service/internal/repository"

	"gorm.io/gorm"
)

type IShipmentService interface {
	AcquireDispatchLock(ctx context.Context, shipmentID string, operatorID string) error
	DispatchShipment(ctx context.Context, shipmentID string, operatorID string) error
	ListShipments(ctx context.Context, status string, params pagination.Params) ([]models.Shipment, *pagination.Meta, error)
	GetShipment(ctx context.Context, shipmentID string) (*models.Shipment, error)
	CreateShipment(ctx context.Context, shipment *models.Shipment) error
	GetMetrics(ctx context.Context) (total int64, inTransit int64, delayed int64, onTimeRate float64, err error)
	ListCarriers(ctx context.Context) ([]models.Carrier, error)
}

type shipmentService struct {
	db          *gorm.DB
	ledgerSvc   ILedgerService
	carrierRepo repository.ICarrierRepository
}

type shipmentServiceRepo struct {
	repo        repository.IShipmentRepository
	poRepo      repository.IPORepository
	stock       repository.IStockRepository
	carrierRepo repository.ICarrierRepository
	ledgerSvc   ILedgerService
}

func NewShipmentService(arg interface{}, args ...interface{}) IShipmentService {
	var ledgerSvc ILedgerService
	var carrierRepo repository.ICarrierRepository
	for _, a := range args {
		if ls, ok := a.(ILedgerService); ok {
			ledgerSvc = ls
		}
		if cr, ok := a.(repository.ICarrierRepository); ok {
			carrierRepo = cr
		}
	}

	switch v := arg.(type) {
	case *gorm.DB:
		if len(args) > 0 {
			if repoArg, ok := args[0].(repository.IShipmentRepository); ok {
				var stock repository.IStockRepository
				var po repository.IPORepository
				if len(args) > 1 {
					if r, ok := args[1].(repository.IStockRepository); ok {
						stock = r
					}
				}
				if len(args) > 2 {
					if r, ok := args[2].(repository.IPORepository); ok {
						po = r
					}
				}
				return &shipmentServiceRepo{repo: repoArg, stock: stock, poRepo: po, carrierRepo: carrierRepo, ledgerSvc: ledgerSvc}
			}
		}
		return &shipmentService{db: v, ledgerSvc: ledgerSvc, carrierRepo: carrierRepo}
	case repository.IShipmentRepository:
		var stock repository.IStockRepository
		var po repository.IPORepository
		if len(args) > 0 {
			if r, ok := args[0].(repository.IStockRepository); ok {
				stock = r
			}
		}
		if len(args) > 1 {
			if r, ok := args[1].(repository.IPORepository); ok {
				po = r
			}
		}
		return &shipmentServiceRepo{repo: v, stock: stock, poRepo: po, carrierRepo: carrierRepo, ledgerSvc: ledgerSvc}
	default:
		panic("invalid NewShipmentService usage")
	}
}

// ── db-backed (direct gorm.DB) implementation ────────────────────────────────

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

func (s *shipmentService) ListShipments(ctx context.Context, status string, params pagination.Params) ([]models.Shipment, *pagination.Meta, error) {
	query := s.db.WithContext(ctx).Model(&models.Shipment{}).Preload("Items").Preload("Supplier")
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

func (s *shipmentService) GetShipment(ctx context.Context, shipmentID string) (*models.Shipment, error) {
	var shipment models.Shipment
	if err := s.db.WithContext(ctx).Preload("Items").Preload("Supplier").First(&shipment, "id = ?", shipmentID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &shipment, nil
}

func (s *shipmentService) CreateShipment(ctx context.Context, shipment *models.Shipment) error {
	// Validate PO exists
	var po models.PurchaseOrder
	if err := s.db.WithContext(ctx).First(&po, "id = ?", shipment.PORef).Error; err != nil {
		return ErrNotFound
	}
	return s.db.WithContext(ctx).Create(shipment).Error
}

func (s *shipmentService) GetMetrics(ctx context.Context) (total int64, inTransit int64, delayed int64, onTimeRate float64, err error) {
	err = s.db.WithContext(ctx).Model(&models.Shipment{}).Where("deleted_at IS NULL").Count(&total).Error
	if err != nil {
		return
	}
	err = s.db.WithContext(ctx).Model(&models.Shipment{}).Where("status = ?", models.ShipmentStatusInTransit).Count(&inTransit).Error
	if err != nil {
		return
	}
	err = s.db.WithContext(ctx).Model(&models.Shipment{}).Where("status = ?", models.ShipmentStatusDelayed).Count(&delayed).Error
	if err != nil {
		return
	}

	var totalDelivered int64
	err = s.db.WithContext(ctx).Model(&models.Shipment{}).Where("status = ?", models.ShipmentStatusDelivered).Count(&totalDelivered).Error
	if err != nil {
		return
	}
	if totalDelivered == 0 {
		onTimeRate = 100.0
		return
	}
	var onTime int64
	err = s.db.WithContext(ctx).Model(&models.Shipment{}).
		Where("status = ? AND updated_at <= eta", models.ShipmentStatusDelivered).
		Count(&onTime).Error
	if err != nil {
		return
	}
	onTimeRate = float64(onTime) / float64(totalDelivered) * 100.0
	return
}

func (s *shipmentService) ListCarriers(ctx context.Context) ([]models.Carrier, error) {
	if s.carrierRepo == nil {
		return nil, nil
	}
	return s.carrierRepo.ListCarriers(ctx)
}

// ── repo-backed implementation ────────────────────────────────────────────────

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

func (s *shipmentServiceRepo) ListShipments(ctx context.Context, status string, params pagination.Params) ([]models.Shipment, *pagination.Meta, error) {
	return s.repo.ListShipments(ctx, status, params)
}

func (s *shipmentServiceRepo) GetShipment(ctx context.Context, shipmentID string) (*models.Shipment, error) {
	shipment, err := s.repo.GetShipmentByID(ctx, shipmentID)
	if err != nil || shipment == nil {
		return nil, ErrNotFound
	}
	return shipment, nil
}

func (s *shipmentServiceRepo) CreateShipment(ctx context.Context, shipment *models.Shipment) error {
	// Validate PO exists
	if s.poRepo != nil {
		if _, err := s.poRepo.GetPOByID(ctx, shipment.PORef); err != nil {
			return ErrNotFound
		}
	}
	return s.repo.CreateShipment(ctx, shipment)
}

func (s *shipmentServiceRepo) GetMetrics(ctx context.Context) (total int64, inTransit int64, delayed int64, onTimeRate float64, err error) {
	return s.repo.GetShipmentMetrics(ctx)
}

func (s *shipmentServiceRepo) ListCarriers(ctx context.Context) ([]models.Carrier, error) {
	if s.carrierRepo == nil {
		return nil, nil
	}
	return s.carrierRepo.ListCarriers(ctx)
}

// generateShipmentID creates a readable shipment ID like SHP-2026-001
func generateShipmentID(year int, count int64) string {
	return fmt.Sprintf("SHP-%d-%03d", year, count+1)
}


