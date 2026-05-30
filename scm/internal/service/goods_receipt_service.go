package service

import (
	"context"
	"time"

	"zeus-scm-service/internal/infrastructure/observability"
	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/pagination"
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
	ListGRs(ctx context.Context, status string, params pagination.Params) ([]models.GoodsReceipt, *pagination.Meta, error)
	FindAllGRs(ctx context.Context) ([]models.GoodsReceipt, error)
	GetGR(ctx context.Context, grID string) (*models.GoodsReceipt, error)
	GetMetrics(ctx context.Context) (pending int64, completedToday int64, discrepancies int64, queue int64, err error)
}

type goodsReceiptService struct {
	db             *gorm.DB
	agingThreshold time.Duration
}

type goodsReceiptServiceRepo struct {
	repo           repository.IGoodsReceiptRepository
	stockRepo      repository.IStockRepository
	poRepo         repository.IPORepository
	agingThreshold time.Duration
}

func NewGoodsReceiptService(arg interface{}, args ...interface{}) IGoodsReceiptService {
	switch v := arg.(type) {
	case *gorm.DB:
		// support both NewGoodsReceiptService(db, years) and NewGoodsReceiptService(db, grRepo, stockRepo, poRepo, years)
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
				return &goodsReceiptServiceRepo{repo: grRepo, stockRepo: stock, poRepo: po, agingThreshold: time.Duration(years) * 365 * 24 * time.Hour}
			}
		}
		years := 0
		if len(args) > 0 {
			if y, ok := args[0].(int); ok {
				years = y
			}
		}
		return &goodsReceiptService{db: v, agingThreshold: time.Duration(years) * 365 * 24 * time.Hour}
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
		return &goodsReceiptServiceRepo{repo: v, stockRepo: stock, poRepo: po, agingThreshold: time.Duration(years) * 365 * 24 * time.Hour}
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
			observability.DefaultRegistry.Counter(observability.MetricLockContention).Inc()
			return ErrAlreadyLocked
		}
	}
	now := time.Now()
	expiresAt := now.Add(60 * time.Minute)
	err := s.db.WithContext(ctx).Model(&gr).Updates(map[string]interface{}{
		"locked_by":       operatorID,
		"lock_expires_at": expiresAt,
	}).Error
	if err == nil {
		observability.DefaultRegistry.Counter(observability.MetricLockAcquisitions).Inc()
	}
	return err
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
		goodQty := received - defective
		if goodQty < 0 {
			goodQty = 0
		}
		stock.StockQty += goodQty
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

	// Update PO line items' ReceivedQty from GR line items
	var poItems []models.POLineItem
	tx.Where("po_id = ?", po.ID).Find(&poItems)
	receivedBySKU := make(map[string]int)
	for _, item := range lineItems {
		if item.ReceivedQty != nil {
			receivedBySKU[item.SKU] += *item.ReceivedQty
		}
	}
	for i := range poItems {
		if qty, ok := receivedBySKU[poItems[i].SKU]; ok {
			poItems[i].ReceivedQty += qty
			tx.Save(&poItems[i])
		}
	}

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

	var incompleteGRs int64
	tx.Model(&models.GoodsReceipt{}).
		Where("po_ref = ? AND status != ?", po.ID, models.GRStatusComplete).
		Count(&incompleteGRs)
	if incompleteGRs == 0 {
		if allReceived {
			po.Status = models.POStatusReceived
		} else {
			po.Status = models.POStatusPartial
		}
		if err := tx.Save(&po).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	observability.DefaultRegistry.Counter(observability.MetricGRProcessed).Inc()
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
			observability.DefaultRegistry.Counter(observability.MetricLockContention).Inc()
			return ErrAlreadyLocked
		}
	}
	now := time.Now()
	expiresAt := now.Add(60 * time.Minute)
	err = s.repo.UpdateGRFields(ctx, grID, map[string]interface{}{"locked_by": operatorID, "lock_expires_at": expiresAt})
	if err == nil {
		observability.DefaultRegistry.Counter(observability.MetricLockAcquisitions).Inc()
	}
	return err
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
		goodQty := received - defective
		if goodQty < 0 {
			goodQty = 0
		}
		stock.StockQty += goodQty
		if err := s.stockRepo.SaveStock(ctx, stock); err != nil {
			return err
		}
	}

	po, err := s.poRepo.GetPOByID(ctx, gr.PORef)
	if err != nil || po == nil {
		return err
	}
	poItems, _ := s.poRepo.GetPOLineItemsByPOID(ctx, po.ID)

	// Update PO line items' ReceivedQty from GR line items
	receivedBySKU := make(map[string]int)
	for _, item := range items {
		if item.ReceivedQty != nil {
			receivedBySKU[item.SKU] += *item.ReceivedQty
		}
	}
	for i := range poItems {
		if qty, ok := receivedBySKU[poItems[i].SKU]; ok {
			poItems[i].ReceivedQty += qty
			_ = s.poRepo.SavePOLineItem(ctx, &poItems[i])
		}
	}

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

	incompleteGRs, _ := s.repo.CountByPOIDAndNotStatus(ctx, po.ID, models.GRStatusComplete)
	if incompleteGRs == 0 {
		if allReceived {
			po.Status = models.POStatusReceived
		} else {
			po.Status = models.POStatusPartial
		}
		if err := s.poRepo.SavePO(ctx, po); err != nil {
			return err
		}
	}

	observability.DefaultRegistry.Counter(observability.MetricGRProcessed).Inc()
	return nil
}

func (s *goodsReceiptServiceRepo) ReleaseLock(ctx context.Context, grID string) error {
	gr, err := s.repo.GetGRByID(ctx, grID)
	if err != nil || gr == nil {
		return ErrNotFound
	}
	return s.repo.UpdateGRFields(ctx, grID, map[string]interface{}{"locked_by": nil, "lock_expires_at": nil})
}

func (s *goodsReceiptService) ListGRs(ctx context.Context, status string, params pagination.Params) ([]models.GoodsReceipt, *pagination.Meta, error) {
	query := s.db.WithContext(ctx).Model(&models.GoodsReceipt{}).Preload("LineItems").Preload("Vendor")
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var grs []models.GoodsReceipt
	meta, err := pagination.Paginate(query, params, &grs, "created_at", "updated_at", "id", "status", "arrival_date")
	if err != nil {
		return nil, nil, err
	}
	return grs, meta, nil
}

func (s *goodsReceiptService) FindAllGRs(ctx context.Context) ([]models.GoodsReceipt, error) {
	var grs []models.GoodsReceipt
	if err := s.db.WithContext(ctx).Preload("LineItems").Preload("Vendor").Order("created_at DESC").Find(&grs).Error; err != nil {
		return nil, err
	}
	return grs, nil
}

func (s *goodsReceiptService) GetGR(ctx context.Context, grID string) (*models.GoodsReceipt, error) {
	var gr models.GoodsReceipt
	if err := s.db.WithContext(ctx).Preload("LineItems").Preload("Vendor").First(&gr, "id = ?", grID).Error; err != nil {
		return nil, ErrNotFound
	}
	return &gr, nil
}

func (s *goodsReceiptService) GetMetrics(ctx context.Context) (int64, int64, int64, int64, error) {
	var pending, completedToday, discrepancies, queue int64
	db := s.db.WithContext(ctx).Model(&models.GoodsReceipt{})
	if err := db.Where("status = ?", models.GRStatusPending).Count(&pending).Error; err != nil {
		return 0, 0, 0, 0, err
	}
	todayStart := time.Now().Truncate(24 * time.Hour)
	if err := db.Where("status = ? AND updated_at >= ?", models.GRStatusComplete, todayStart).Count(&completedToday).Error; err != nil {
		return 0, 0, 0, 0, err
	}
	if err := db.Where("status = ?", models.GRStatusDiscrepancy).Count(&discrepancies).Error; err != nil {
		return 0, 0, 0, 0, err
	}
	if err := db.Where("status = ?", models.GRStatusInspected).Count(&queue).Error; err != nil {
		return 0, 0, 0, 0, err
	}
	return pending, completedToday, discrepancies, queue, nil
}

func (s *goodsReceiptServiceRepo) ListGRs(ctx context.Context, status string, params pagination.Params) ([]models.GoodsReceipt, *pagination.Meta, error) {
	return s.repo.ListGRs(ctx, status, params)
}

func (s *goodsReceiptServiceRepo) FindAllGRs(ctx context.Context) ([]models.GoodsReceipt, error) {
	return s.repo.FindAllGRs(ctx)
}

func (s *goodsReceiptServiceRepo) GetGR(ctx context.Context, grID string) (*models.GoodsReceipt, error) {
	gr, err := s.repo.GetGRByID(ctx, grID)
	if err != nil || gr == nil {
		return nil, ErrNotFound
	}
	return gr, nil
}

func (s *goodsReceiptServiceRepo) GetMetrics(ctx context.Context) (int64, int64, int64, int64, error) {
	return s.repo.GetMetrics(ctx)
}
