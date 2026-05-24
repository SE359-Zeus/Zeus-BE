package service

import (
	"context"
	"time"

	"zeus-scm-service/internal/models"

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
}

func NewGoodsReceiptService(db *gorm.DB, agingThresholdYears int) IGoodsReceiptService {
	return &goodsReceiptService{
		db:             db,
		agingThreshold: time.Duration(agingThresholdYears) * 365 * 24 * time.Hour,
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
