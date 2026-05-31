package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"zeus-scm-service/internal/consumer"
	"zeus-scm-service/internal/infrastructure/observability"
	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/pagination"
	"zeus-scm-service/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IPOService interface {
	CreateDraft(ctx context.Context, vendorID uuid.UUID, targetBuild string) (*models.PurchaseOrder, error)
	AddLineItemWithLock(ctx context.Context, poID string, sku string, qty int) error
	ApprovePO(ctx context.Context, poID string) error
	TransitionState(ctx context.Context, poID string, newState models.POStatus) error
	ListPOs(ctx context.Context, params pagination.Params, q string) ([]models.PurchaseOrder, *pagination.Meta, error)
	GetPO(ctx context.Context, poID string) (*models.PurchaseOrder, error)
	FindAllPOs(ctx context.Context) ([]models.PurchaseOrder, error)
	CreatePO(ctx context.Context, po *models.PurchaseOrder) error
	GetMetrics(ctx context.Context) (total int64, draft int64, approved int64, inTransit int64, received int64, partial int64, void int64, err error)
}

type poService struct {
	db     *gorm.DB
	grRepo repository.IGoodsReceiptRepository
	mqURL  string
}

type poServiceRepo struct {
	poRepo    repository.IPORepository
	stockRepo repository.IStockRepository
	grRepo    repository.IGoodsReceiptRepository
	mqURL     string
}

func hydratePurchaseOrder(po *models.PurchaseOrder) {
	if po != nil && po.Vendor != nil {
		po.VendorName = po.Vendor.Name
	}
}

func hydratePurchaseOrders(pos []models.PurchaseOrder) {
	for i := range pos {
		hydratePurchaseOrder(&pos[i])
	}
}

func cloneAndHydratePurchaseOrders(pos []models.PurchaseOrder) []models.PurchaseOrder {
	out := make([]models.PurchaseOrder, len(pos))
	copy(out, pos)
	hydratePurchaseOrders(out)
	return out
}

func NewPOService(arg interface{}, args ...interface{}) IPOService {
	switch v := arg.(type) {
	case *gorm.DB:
		mqURL := ""
		var grRepo repository.IGoodsReceiptRepository
		for _, a := range args {
			switch typed := a.(type) {
			case string:
				mqURL = typed
			case repository.IGoodsReceiptRepository:
				grRepo = typed
			}
		}
		return &poService{db: v, grRepo: grRepo, mqURL: mqURL}
	case repository.IPORepository:
		var stock repository.IStockRepository
		var grRepo repository.IGoodsReceiptRepository
		var mqURL string
		for _, a := range args {
			switch typed := a.(type) {
			case repository.IStockRepository:
				stock = typed
			case string:
				mqURL = typed
			case repository.IGoodsReceiptRepository:
				grRepo = typed
			}
		}
		return &poServiceRepo{poRepo: v, stockRepo: stock, grRepo: grRepo, mqURL: mqURL}
	default:
		panic("invalid NewPOService usage")
	}
}

func (s *poService) CreateDraft(ctx context.Context, vendorID uuid.UUID, targetBuild string) (*models.PurchaseOrder, error) {
	var existingPO models.PurchaseOrder
	if err := s.db.WithContext(ctx).
		Where("vendor_id = ? AND status IN ?", vendorID, []models.POStatus{models.POStatusDraft, models.POStatusApproved, models.POStatusInTransit}).
		First(&existingPO).Error; err == nil {
		return nil, ErrMonoVendorViolation
	}

	var count int64
	year := time.Now().Year()
	s.db.WithContext(ctx).Model(&models.PurchaseOrder{}).
		Where("id LIKE ?", fmt.Sprintf("PO-%d-%%", year)).
		Count(&count)

	po := &models.PurchaseOrder{
		ID:          fmt.Sprintf("PO-%d-%d", year, count+1),
		VendorID:    vendorID,
		TargetBuild: targetBuild,
		Status:      models.POStatusDraft,
		TotalValue:  0,
	}
	if err := s.db.WithContext(ctx).Create(po).Error; err != nil {
		return nil, err
	}
	observability.DefaultRegistry.Counter(observability.MetricPOCreated).Inc()
	return po, nil
}

func (s *poService) AddLineItemWithLock(ctx context.Context, poID string, sku string, qty int) error {
	var po models.PurchaseOrder
	if err := s.db.WithContext(ctx).First(&po, "id = ?", poID).Error; err != nil {
		return ErrNotFound
	}
	if po.Status != models.POStatusDraft {
		return ErrInvalidTransition
	}

	lockMgr := consumer.NewDeficitLockManager(s.mqURL)
	if err := lockMgr.LockDeficit(ctx, sku, qty); err != nil {
		if errors.Is(err, consumer.ErrInsufficientDeficit) {
			return ErrInsufficientDeficit
		}
		slog.Warn("deficit locking failed or rabbitmq unavailable; proceeding in degraded mode",
			slog.String("service", "scm"),
			slog.String("component", "purchase_order"),
			slog.Any("error", err),
		)
	}


	var catalog models.ComponentStock
	if err := s.db.WithContext(ctx).First(&catalog, "sku = ?", sku).Error; err != nil {
		return err
	}

	lineItem := &models.POLineItem{
		ID:         uuid.New(),
		POID:       poID,
		SKU:        sku,
		OrderedQty: qty,
		UnitPrice:  catalog.UnitCost,
	}
	return s.db.WithContext(ctx).Create(lineItem).Error
}

// repo-backed implementation
func (s *poServiceRepo) CreateDraft(ctx context.Context, vendorID uuid.UUID, targetBuild string) (*models.PurchaseOrder, error) {
	existing, err := s.poRepo.FindPOByVendorAndStatuses(ctx, vendorID, []models.POStatus{models.POStatusDraft, models.POStatusApproved, models.POStatusInTransit})
	if err == nil && existing != nil {
		return nil, ErrMonoVendorViolation
	}
	year := time.Now().Year()
	count, err := s.poRepo.CountPOsByYearPattern(ctx, year, "PO-%d-%%")
	if err != nil {
		return nil, err
	}
	po := &models.PurchaseOrder{
		ID:          fmt.Sprintf("PO-%d-%d", year, count+1),
		VendorID:    vendorID,
		TargetBuild: targetBuild,
		Status:      models.POStatusDraft,
		TotalValue:  0,
	}
	if err := s.poRepo.CreatePO(ctx, po); err != nil {
		return nil, err
	}
	observability.DefaultRegistry.Counter(observability.MetricPOCreated).Inc()
	return po, nil
}

func (s *poServiceRepo) AddLineItemWithLock(ctx context.Context, poID string, sku string, qty int) error {
	po, err := s.poRepo.GetPOByID(ctx, poID)
	if err != nil || po == nil {
		return ErrNotFound
	}
	if po.Status != models.POStatusDraft {
		return ErrInvalidTransition
	}

	lockMgr := consumer.NewDeficitLockManager(s.mqURL)
	if err := lockMgr.LockDeficit(ctx, sku, qty); err != nil {
		if errors.Is(err, consumer.ErrInsufficientDeficit) {
			return ErrInsufficientDeficit
		}
		return err
	}

	catalog, err := s.stockRepo.GetStockBySKU(ctx, sku)
	if err != nil {
		return err
	}

	lineItem := &models.POLineItem{
		ID:         uuid.New(),
		POID:       poID,
		SKU:        sku,
		OrderedQty: qty,
		UnitPrice:  catalog.UnitCost,
	}
	return s.poRepo.CreatePOLineItem(ctx, lineItem)
}

func (s *poServiceRepo) ApprovePO(ctx context.Context, poID string) error {
	po, err := s.poRepo.GetPOByID(ctx, poID)
	if err != nil || po == nil {
		return ErrNotFound
	}
	if po.Status != models.POStatusDraft {
		return ErrInvalidTransition
	}
	items, _ := s.poRepo.GetPOLineItemsByPOID(ctx, poID)
	var totalValue float64
	for _, item := range items {
		totalValue += float64(item.OrderedQty) * item.UnitPrice
	}
	po.Status = models.POStatusApproved
	po.TotalValue = totalValue
	return s.poRepo.SavePO(ctx, po)
}

func (s *poServiceRepo) TransitionState(ctx context.Context, poID string, newState models.POStatus) error {
	po, err := s.poRepo.GetPOByID(ctx, poID)
	if err != nil || po == nil {
		return ErrNotFound
	}
	if !validTransition(po.Status, newState) {
		return ErrStateRegression
	}
	if newState != models.POStatusVoid && s.grRepo != nil {
		incomplete, err := s.grRepo.CountByPOIDAndNotStatus(ctx, poID, models.GRStatusComplete)
		if err != nil {
			return err
		}
		if incomplete > 0 {
			return ErrIncompleteGRs
		}
	}
	observability.DefaultRegistry.Counter(observability.MetricPOStateTransitions).Inc()
	return s.poRepo.UpdatePOStatus(ctx, poID, newState)
}

func (s *poService) ApprovePO(ctx context.Context, poID string) error {
	var po models.PurchaseOrder
	if err := s.db.WithContext(ctx).First(&po, "id = ?", poID).Error; err != nil {
		return ErrNotFound
	}
	if po.Status != models.POStatusDraft {
		return ErrInvalidTransition
	}

	po.Status = models.POStatusApproved
	var totalValue float64
	var lineItems []models.POLineItem
	s.db.WithContext(ctx).Where("po_id = ?", poID).Find(&lineItems)
	for _, item := range lineItems {
		totalValue += float64(item.OrderedQty) * item.UnitPrice
	}
	po.TotalValue = totalValue

	return s.db.WithContext(ctx).Save(&po).Error
}

func (s *poService) TransitionState(ctx context.Context, poID string, newState models.POStatus) error {
	var po models.PurchaseOrder
	if err := s.db.WithContext(ctx).First(&po, "id = ?", poID).Error; err != nil {
		return ErrNotFound
	}

	valid := validTransition(po.Status, newState)
	if !valid {
		return ErrStateRegression
	}

	if newState != models.POStatusVoid && s.grRepo != nil {
		incomplete, err := s.grRepo.CountByPOIDAndNotStatus(ctx, poID, models.GRStatusComplete)
		if err != nil {
			return err
		}
		if incomplete > 0 {
			return ErrIncompleteGRs
		}
	}

	observability.DefaultRegistry.Counter(observability.MetricPOStateTransitions).Inc()
	return s.db.WithContext(ctx).Model(&po).Update("status", newState).Error
}

func validTransition(current, new models.POStatus) bool {
	order := []models.POStatus{
		models.POStatusDraft,
		models.POStatusApproved,
		models.POStatusInTransit,
		models.POStatusReceived,
		models.POStatusPartial,
		models.POStatusVoid,
	}
	currentIdx := -1
	newIdx := -1
	for i, s := range order {
		if s == current {
			currentIdx = i
		}
		if s == new {
			newIdx = i
		}
	}
	if currentIdx == -1 || newIdx == -1 {
		return false
	}
	if new == models.POStatusVoid && current == models.POStatusDraft {
		return true
	}
	return newIdx > currentIdx
}

func (s *poService) ListPOs(ctx context.Context, params pagination.Params, q string) ([]models.PurchaseOrder, *pagination.Meta, error) {
	query := s.db.WithContext(ctx).Model(&models.PurchaseOrder{}).Preload("LineItems").Preload("Vendor")
	if q != "" {
		like := "%" + q + "%"
		query = query.Where("id LIKE ? OR target_build LIKE ? OR status LIKE ?", like, like, like)
	}
	var pos []models.PurchaseOrder
	meta, err := pagination.Paginate(query, params, &pos, "created_at", "updated_at", "id", "status")
	if err != nil {
		return nil, nil, err
	}
	hydratePurchaseOrders(pos)
	return pos, meta, nil
}

func (s *poService) GetPO(ctx context.Context, poID string) (*models.PurchaseOrder, error) {
	var po models.PurchaseOrder
	if err := s.db.WithContext(ctx).Preload("LineItems").Preload("Vendor").First(&po, "id = ?", poID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	hydratePurchaseOrder(&po)
	return &po, nil
}

func (s *poService) FindAllPOs(ctx context.Context) ([]models.PurchaseOrder, error) {
	var pos []models.PurchaseOrder
	if err := s.db.WithContext(ctx).Preload("LineItems").Preload("Vendor").Order("created_at DESC").Find(&pos).Error; err != nil {
		return nil, err
	}
	return cloneAndHydratePurchaseOrders(pos), nil
}

func (s *poService) CreatePO(ctx context.Context, po *models.PurchaseOrder) error {
	var existingPO models.PurchaseOrder
	if err := s.db.WithContext(ctx).
		Where("vendor_id = ? AND status IN ?", po.VendorID, []models.POStatus{models.POStatusDraft, models.POStatusApproved, models.POStatusInTransit}).
		First(&existingPO).Error; err == nil {
		return ErrMonoVendorViolation
	}

	if len(po.LineItems) == 0 {
		return fmt.Errorf("purchase order must have at least one line item")
	}

	var totalValue float64
	for i := range po.LineItems {
		item := &po.LineItems[i]
		var mapping models.SkuMapping
		if err := s.db.WithContext(ctx).
			Where("supplier_id = ? AND sku = ?", po.VendorID, item.SKU).
			First(&mapping).Error; err != nil {
			return fmt.Errorf("sku mapping not found for SKU: %s", item.SKU)
		}
		item.ID = uuid.New()
		item.POID = po.ID
		item.Description = mapping.Name
		item.UnitPrice = mapping.UnitPrice
		item.ReceivedQty = 0
		totalValue += float64(item.OrderedQty) * item.UnitPrice
	}

	po.Status = models.POStatusDraft
	po.TotalValue = totalValue

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(po).Error; err != nil {
			return err
		}
		for i := range po.LineItems {
			if err := tx.Create(&po.LineItems[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *poServiceRepo) ListPOs(ctx context.Context, params pagination.Params, q string) ([]models.PurchaseOrder, *pagination.Meta, error) {
	pos, meta, err := s.poRepo.ListPOs(ctx, params, q)
	if err != nil {
		return nil, nil, err
	}
	hydratePurchaseOrders(pos)
	return pos, meta, nil
}

func (s *poServiceRepo) GetPO(ctx context.Context, poID string) (*models.PurchaseOrder, error) {
	po, err := s.poRepo.GetPOByID(ctx, poID)
	if err != nil || po == nil {
		return nil, ErrNotFound
	}
	hydratePurchaseOrder(po)
	return po, nil
}

func (s *poServiceRepo) FindAllPOs(ctx context.Context) ([]models.PurchaseOrder, error) {
	pos, err := s.poRepo.FindAllPOs(ctx)
	if err != nil {
		return nil, err
	}
	return cloneAndHydratePurchaseOrders(pos), nil
}

func (s *poServiceRepo) CreatePO(ctx context.Context, po *models.PurchaseOrder) error {
	existing, err := s.poRepo.FindPOByVendorAndStatuses(ctx, po.VendorID, []models.POStatus{models.POStatusDraft, models.POStatusApproved, models.POStatusInTransit})
	if err == nil && existing != nil {
		return ErrMonoVendorViolation
	}

	if len(po.LineItems) == 0 {
		return fmt.Errorf("purchase order must have at least one line item")
	}

	var totalValue float64
	for i := range po.LineItems {
		item := &po.LineItems[i]
		mapping, err := s.poRepo.FindSkuMapping(ctx, po.VendorID, item.SKU)
		if err != nil || mapping == nil {
			return fmt.Errorf("sku mapping not found for SKU: %s", item.SKU)
		}
		item.ID = uuid.New()
		item.POID = po.ID
		item.Description = mapping.Name
		item.UnitPrice = mapping.UnitPrice
		item.ReceivedQty = 0
		totalValue += float64(item.OrderedQty) * item.UnitPrice
	}

	po.Status = models.POStatusDraft
	po.TotalValue = totalValue

	if err := s.poRepo.CreatePO(ctx, po); err != nil {
		return err
	}

	for i := range po.LineItems {
		if err := s.poRepo.CreatePOLineItem(ctx, &po.LineItems[i]); err != nil {
			return err
		}
	}
	return nil
}

func (s *poService) GetMetrics(ctx context.Context) (int64, int64, int64, int64, int64, int64, int64, error) {
	var total, draft, approved, inTransit, received, partial, void int64
	db := s.db.WithContext(ctx).Model(&models.PurchaseOrder{})
	if err := db.Count(&total).Error; err != nil {
		return 0, 0, 0, 0, 0, 0, 0, err
	}
	db.Where("status = ?", models.POStatusDraft).Count(&draft)
	db.Where("status = ?", models.POStatusApproved).Count(&approved)
	db.Where("status = ?", models.POStatusInTransit).Count(&inTransit)
	db.Where("status = ?", models.POStatusReceived).Count(&received)
	db.Where("status = ?", models.POStatusPartial).Count(&partial)
	db.Where("status = ?", models.POStatusVoid).Count(&void)
	return total, draft, approved, inTransit, received, partial, void, nil
}

func (s *poServiceRepo) GetMetrics(ctx context.Context) (int64, int64, int64, int64, int64, int64, int64, error) {
	return s.poRepo.GetPOMetrics(ctx)
}
