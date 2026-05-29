package service

import (
	"context"
	"time"

	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/repository"

	"gorm.io/gorm"
)

type IGoodsReceiptService interface {
	AcquireLock(ctx context.Context, grID string, operatorID string) error
	ProcessBlindReceipt(ctx context.Context, grID string, operatorID string, counts map[string]struct {
		Received  int
		Defective int
	}) error
	ReleaseLock(ctx context.Context, grID string) error
}

type goodsReceiptService struct {
	db             *gorm.DB
	agingThreshold time.Duration
	ledgerSvc      ILedgerService
}

type goodsReceiptServiceRepo struct {
	repo           repository.IGoodsReceiptRepository
	stockRepo      repository.IStockRepository
	poRepo         repository.IPORepository
	agingThreshold time.Duration
	ledgerSvc      ILedgerService
}

func NewGoodsReceiptService(arg interface{}, args ...interface{}) IGoodsReceiptService {
	var ledgerSvc ILedgerService
	// scan all args for ILedgerService
	for _, a := range args {
		if ls, ok := a.(ILedgerService); ok {
			ledgerSvc = ls
		}
	}

	switch v := arg.(type) {
	case *gorm.DB:
		// support both NewGoodsReceiptService(db, years) and NewGoodsReceiptService(db, grRepo, stockRepo, poRepo, years, ledgerSvc)
		if len(args) > 0 {
			// if second arg is a repo, build repo-backed adapter
			if grRepo, ok := args[0].(repository.IGoodsReceiptRepository); ok {
				var stock repository.IStockRepository
				var po repository.IPORepository
				var years int
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
				if len(args) > 3 {
					if y, ok := args[3].(int); ok {
						years = y
					}
				}
				return &goodsReceiptServiceRepo{repo: grRepo, stockRepo: stock, poRepo: po, agingThreshold: time.Duration(years) * 365 * 24 * time.Hour, ledgerSvc: ledgerSvc}
			}
		}
		years := 0
		if len(args) > 0 {
			if y, ok := args[0].(int); ok {
				years = y
			}
		}
		return &goodsReceiptService{db: v, agingThreshold: time.Duration(years) * 365 * 24 * time.Hour, ledgerSvc: ledgerSvc}
	case repository.IGoodsReceiptRepository:
		var stock repository.IStockRepository
		var po repository.IPORepository
		var years int
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
		if len(args) > 2 {
			if y, ok := args[2].(int); ok {
				years = y
			}
		}
		return &goodsReceiptServiceRepo{repo: v, stockRepo: stock, poRepo: po, agingThreshold: time.Duration(years) * 365 * 24 * time.Hour, ledgerSvc: ledgerSvc}
	default:
		panic("invalid NewGoodsReceiptService usage")
	}
}

func (s *goodsReceiptService) AcquireLock(ctx context.Context, grID string, operatorID string) error {
	var gr models.GoodsReceipt
	if err := s.db.WithContext(ctx).First(&gr, "id = ?", grID).Error; err != nil {
		return ErrNotFound
	}
	if gr.LockedBy != nil && *gr.LockedBy != operatorID {
		if gr.LockExpiresAt != nil && gr.LockExpiresAt.After(time.Now()) {
			return ErrAlreadyLocked
		}
	}
	now := time.Now()
	expiresAt := now.Add(60 * time.Minute)
	return s.db.WithContext(ctx).Model(&gr).Updates(map[string]interface{}{
		"locked_by":       operatorID,
		"lock_expires_at": expiresAt,
	}).Error
}

func (s *goodsReceiptService) ProcessBlindReceipt(ctx context.Context, grID string, operatorID string, counts map[string]struct {
	Received  int
	Defective int
}) error {
	var gr models.GoodsReceipt
	if err := s.db.WithContext(ctx).First(&gr, "id = ?", grID).Error; err != nil {
		return ErrNotFound
	}
	if gr.LockedBy == nil || *gr.LockedBy != operatorID {
		return ErrAlreadyLocked
	}
	if gr.LockExpiresAt != nil && gr.LockExpiresAt.Before(time.Now()) {
		return ErrLockExpired
	}

	var lineItems []models.GRLineItem
	if err := s.db.WithContext(ctx).Where("gr_id = ?", grID).Find(&lineItems).Error; err != nil {
		return err
	}

	tx := s.db.WithContext(ctx).Begin()

	for i := range lineItems {
		item := &lineItems[i]
		count, ok := counts[item.SKU]
		if !ok {
			continue
		}
		received := count.Received
		defective := count.Defective
		item.ReceivedQty = &received
		item.DefectiveQty = &defective

		if item.AgingSensitive && item.ProductionDate != nil {
			if time.Since(*item.ProductionDate) > s.agingThreshold {
				item.AgingLabel = "Over-Age"
			}
		}
		if err := tx.Save(item).Error; err != nil {
			tx.Rollback()
			return err
		}

		var stock models.ComponentStock
		if err := tx.First(&stock, "sku = ?", item.SKU).Error; err != nil {
			tx.Rollback()
			return err
		}
		stock.StockQty += received
		if err := tx.Save(&stock).Error; err != nil {
			tx.Rollback()
			return err
		}

		if s.ledgerSvc != nil && received > 0 {
			if _, lerr := s.ledgerSvc.RecordEntry(ctx, item.SKU, models.LedgerTxnTypeIN, received, operatorID, gr.ID, models.LedgerRefGoodsReceipt, gr.ID); lerr != nil {
				s.ledgerSvc.RecordEntry(context.Background(), item.SKU, models.LedgerTxnTypeIN, received, operatorID, gr.ID, models.LedgerRefGoodsReceipt, gr.ID)
			}
		}
	}

	var po models.PurchaseOrder
	if err := tx.First(&po, "id = ?", gr.PORef).Error; err != nil {
		tx.Rollback()
		return err
	}

	var poItems []models.POLineItem
	tx.Where("po_id = ?", po.ID).Find(&poItems)
	allReceived := true
	for _, li := range poItems {
		if li.ReceivedQty < li.OrderedQty {
			allReceived = false
			break
		}
	}

	gr.Status = models.GRStatusComplete
	if err := tx.Save(&gr).Error; err != nil {
		tx.Rollback()
		return err
	}

	if allReceived {
		po.Status = models.POStatusReceived
	} else {
		po.Status = models.POStatusPartial
	}
	if err := tx.Save(&po).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (s *goodsReceiptService) ReleaseLock(ctx context.Context, grID string) error {
	var gr models.GoodsReceipt
	if err := s.db.WithContext(ctx).First(&gr, "id = ?", grID).Error; err != nil {
		return ErrNotFound
	}
	return s.db.WithContext(ctx).Model(&gr).Updates(map[string]interface{}{
		"locked_by":       nil,
		"lock_expires_at": nil,
	}).Error
}

// repo-backed implementation
func (s *goodsReceiptServiceRepo) AcquireLock(ctx context.Context, grID string, operatorID string) error {
	gr, err := s.repo.GetGRByID(ctx, grID)
	if err != nil || gr == nil {
		return ErrNotFound
	}
	if gr.LockedBy != nil && *gr.LockedBy != operatorID {
		if gr.LockExpiresAt != nil && gr.LockExpiresAt.After(time.Now()) {
			return ErrAlreadyLocked
		}
	}
	now := time.Now()
	expiresAt := now.Add(60 * time.Minute)
	return s.repo.UpdateGRFields(ctx, grID, map[string]interface{}{"locked_by": operatorID, "lock_expires_at": expiresAt})
}

func (s *goodsReceiptServiceRepo) ProcessBlindReceipt(ctx context.Context, grID string, operatorID string, counts map[string]struct {
	Received  int
	Defective int
}) error {
	gr, err := s.repo.GetGRByID(ctx, grID)
	if err != nil || gr == nil {
		return ErrNotFound
	}
	if gr.LockedBy == nil || *gr.LockedBy != operatorID {
		return ErrAlreadyLocked
	}
	if gr.LockExpiresAt != nil && gr.LockExpiresAt.Before(time.Now()) {
		return ErrLockExpired
	}

	items, err := s.repo.FindGRLineItemsByGRID(ctx, grID)
	if err != nil {
		return err
	}

	// emulate transaction semantics via sequence of operations; repo implementations/tests should simulate rollback behavior if needed
	for _, item := range items {
		count, ok := counts[item.SKU]
		if !ok {
			continue
		}
		received := count.Received
		defective := count.Defective
		item.ReceivedQty = &received
		item.DefectiveQty = &defective

		if item.AgingSensitive && item.ProductionDate != nil {
			if time.Since(*item.ProductionDate) > s.agingThreshold {
				item.AgingLabel = "Over-Age"
			}
		}
		if err := s.repo.SaveGRLineItem(ctx, &item); err != nil {
			return err
		}

		stock, err := s.stockRepo.GetStockBySKU(ctx, item.SKU)
		if err != nil {
			return err
		}
		stock.StockQty += received
		if err := s.stockRepo.SaveStock(ctx, stock); err != nil {
			return err
		}

		if s.ledgerSvc != nil && received > 0 {
			if _, lerr := s.ledgerSvc.RecordEntry(ctx, item.SKU, models.LedgerTxnTypeIN, received, operatorID, gr.ID, models.LedgerRefGoodsReceipt, gr.ID); lerr != nil {
				s.ledgerSvc.RecordEntry(context.Background(), item.SKU, models.LedgerTxnTypeIN, received, operatorID, gr.ID, models.LedgerRefGoodsReceipt, gr.ID)
			}
		}
	}

	po, err := s.poRepo.GetPOByID(ctx, gr.PORef)
	if err != nil || po == nil {
		return err
	}
	poItems, _ := s.poRepo.GetPOLineItemsByPOID(ctx, po.ID)
	allReceived := true
	for _, li := range poItems {
		if li.ReceivedQty < li.OrderedQty {
			allReceived = false
			break
		}
	}

	gr.Status = models.GRStatusComplete
	if err := s.repo.UpdateGR(ctx, gr); err != nil {
		return err
	}

	if allReceived {
		po.Status = models.POStatusReceived
	} else {
		po.Status = models.POStatusPartial
	}
	if err := s.poRepo.SavePO(ctx, po); err != nil {
		return err
	}
	return nil
}

func (s *goodsReceiptServiceRepo) ReleaseLock(ctx context.Context, grID string) error {
	gr, err := s.repo.GetGRByID(ctx, grID)
	if err != nil || gr == nil {
		return ErrNotFound
	}
	return s.repo.UpdateGRFields(ctx, grID, map[string]interface{}{"locked_by": nil, "lock_expires_at": nil})
}
