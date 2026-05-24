package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"zeus-scm-service/internal/messaging"
	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IPOService interface {
	CreateDraft(ctx context.Context, vendorID uuid.UUID, targetBuild string) (*models.PurchaseOrder, error)
	AddLineItemWithLock(ctx context.Context, poID string, sku string, qty int) error
	ApprovePO(ctx context.Context, poID string) error
	TransitionState(ctx context.Context, poID string, newState models.POStatus) error
}

type poService struct {
	db    *gorm.DB
	mqURL string
}

type poServiceRepo struct {
	poRepo    repository.IPORepository
	stockRepo repository.IStockRepository
	mqURL     string
}

func NewPOService(arg interface{}, args ...interface{}) IPOService {
	switch v := arg.(type) {
	case *gorm.DB:
		mqURL := ""
		if len(args) > 0 {
			if s, ok := args[0].(string); ok {
				mqURL = s
			}
		}
		return &poService{db: v, mqURL: mqURL}
	case repository.IPORepository:
		var stock repository.IStockRepository
		var mqURL string
		if len(args) > 0 {
			if r, ok := args[0].(repository.IStockRepository); ok {
				stock = r
			}
		}
		if len(args) > 1 {
			if s, ok := args[1].(string); ok {
				mqURL = s
			}
		}
		return &poServiceRepo{poRepo: v, stockRepo: stock, mqURL: mqURL}
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

	conn, err := messaging.Dial(s.mqURL)
	if err != nil {
		return err
	}
	defer conn.Close()

	done := make(chan struct{}, 1)
	var consumeErr error

	go func() {
		defer func() { close(done) }()
		msg, ok, err := conn.GetFromPool(false)
		if err != nil {
			consumeErr = err
			return
		}
		if !ok {
			consumeErr = ErrInsufficientDeficit
			return
		}
		var d messaging.DeficitMessage
		if err := json.Unmarshal(msg.Body, &d); err != nil {
			_ = conn.Nack(msg.DeliveryTag, true)
			consumeErr = err
			return
		}
		if d.SKU != sku || d.Qty < qty {
			_ = conn.Nack(msg.DeliveryTag, true)
			consumeErr = ErrInsufficientDeficit
			return
		}
		reservedMsg := messaging.DeficitMessage{
			SKU: sku,
			Qty: qty,
		}
		if err := conn.PublishToReserved(ctx, reservedMsg); err != nil {
			_ = conn.Nack(msg.DeliveryTag, true)
			consumeErr = err
			return
		}
		if err := conn.Ack(msg.DeliveryTag); err != nil {
			consumeErr = err
			return
		}
	}()

	select {
	case <-done:
		if consumeErr != nil {
			return consumeErr
		}
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second):
		return ErrInsufficientDeficit
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

	conn, err := messaging.Dial(s.mqURL)
	if err != nil {
		return err
	}
	defer conn.Close()

	done := make(chan struct{}, 1)
	var consumeErr error

	go func() {
		defer func() { close(done) }()
		msg, ok, err := conn.GetFromPool(false)
		if err != nil {
			consumeErr = err
			return
		}
		if !ok {
			consumeErr = ErrInsufficientDeficit
			return
		}
		var d messaging.DeficitMessage
		if err := json.Unmarshal(msg.Body, &d); err != nil {
			_ = conn.Nack(msg.DeliveryTag, true)
			consumeErr = err
			return
		}
		if d.SKU != sku || d.Qty < qty {
			_ = conn.Nack(msg.DeliveryTag, true)
			consumeErr = ErrInsufficientDeficit
			return
		}
		reservedMsg := messaging.DeficitMessage{SKU: sku, Qty: qty}
		if err := conn.PublishToReserved(ctx, reservedMsg); err != nil {
			_ = conn.Nack(msg.DeliveryTag, true)
			consumeErr = err
			return
		}
		if err := conn.Ack(msg.DeliveryTag); err != nil {
			consumeErr = err
			return
		}
	}()

	select {
	case <-done:
		if consumeErr != nil {
			return consumeErr
		}
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second):
		return ErrInsufficientDeficit
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
