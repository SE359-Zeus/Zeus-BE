package service

import (
	"context"
	"fmt"
	"time"

	"zeus-scm-service/internal/infrastructure/observability"
	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/pagination"
	"zeus-scm-service/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IShipmentService interface {
	AcquireDispatchLock(ctx context.Context, shipmentID string, operatorID string) error
	DispatchShipment(ctx context.Context, shipmentID string, operatorID string) error
	MarkDelivered(ctx context.Context, shipmentID string, operatorID string) error
	TransitionState(ctx context.Context, shipmentID string, newState models.ShipmentStatus) error
	ListShipments(ctx context.Context, status string, params pagination.Params) ([]models.Shipment, *pagination.Meta, error)
	FindAllShipments(ctx context.Context) ([]models.Shipment, error)
	GetShipment(ctx context.Context, shipmentID string) (*models.Shipment, error)
	CreateShipment(ctx context.Context, shipment *models.Shipment) error
	GetMetrics(ctx context.Context) (total int64, inTransit int64, delayed int64, onTimeRate float64, err error)
	ListCarriers(ctx context.Context) ([]models.Carrier, error)
}

type shipmentService struct {
	db          *gorm.DB
	ledgerSvc   ILedgerService
	carrierRepo repository.ICarrierRepository
	grRepo      repository.IGoodsReceiptRepository
}

type shipmentServiceRepo struct {
	repo        repository.IShipmentRepository
	poRepo      repository.IPORepository
	stock       repository.IStockRepository
	vendorRepo  repository.IVendorRepository
	carrierRepo repository.ICarrierRepository
	grRepo      repository.IGoodsReceiptRepository
	ledgerSvc   ILedgerService
}

func NewShipmentService(arg interface{}, args ...interface{}) IShipmentService {
	var ledgerSvc ILedgerService
	var carrierRepo repository.ICarrierRepository
	var grRepo repository.IGoodsReceiptRepository
	for _, a := range args {
		if ls, ok := a.(ILedgerService); ok {
			ledgerSvc = ls
		}
		if cr, ok := a.(repository.ICarrierRepository); ok {
			carrierRepo = cr
		}
		if gr, ok := a.(repository.IGoodsReceiptRepository); ok {
			grRepo = gr
		}
	}

	switch v := arg.(type) {
	case *gorm.DB:
		if len(args) > 0 {
			if repoArg, ok := args[0].(repository.IShipmentRepository); ok {
				var stock repository.IStockRepository
				var po repository.IPORepository
				var vendor repository.IVendorRepository
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
					if r, ok := args[3].(repository.IVendorRepository); ok {
						vendor = r
					}
				}
				return &shipmentServiceRepo{repo: repoArg, stock: stock, poRepo: po, vendorRepo: vendor, carrierRepo: carrierRepo, grRepo: grRepo, ledgerSvc: ledgerSvc}
			}
		}
		return &shipmentService{db: v, ledgerSvc: ledgerSvc, carrierRepo: carrierRepo, grRepo: grRepo}
	case repository.IShipmentRepository:
		var stock repository.IStockRepository
		var po repository.IPORepository
		var vendor repository.IVendorRepository
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
			if r, ok := args[2].(repository.IVendorRepository); ok {
				vendor = r
			}
		}
		return &shipmentServiceRepo{repo: v, stock: stock, poRepo: po, vendorRepo: vendor, carrierRepo: carrierRepo, grRepo: grRepo, ledgerSvc: ledgerSvc}
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

func (s *shipmentService) MarkDelivered(ctx context.Context, shipmentID string, operatorID string) error {
	var shipment models.Shipment
	if err := s.db.WithContext(ctx).First(&shipment, "id = ?", shipmentID).Error; err != nil {
		return ErrNotFound
	}
	if shipment.Status != models.ShipmentStatusInTransit {
		return ErrInvalidTransition
	}

	tx := s.db.WithContext(ctx).Begin()

	shipment.Status = models.ShipmentStatusDelivered
	if err := tx.Save(&shipment).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Auto-create GR from shipment
	var poItems []models.POLineItem
	if err := tx.Where("po_id = ?", shipment.PORef).Find(&poItems).Error; err != nil {
		tx.Rollback()
		return err
	}

	var existingGRs []models.GoodsReceipt
	tx.Where("po_ref = ?", shipment.PORef).Find(&existingGRs)
	grIdx := len(existingGRs) + 1
	grID := fmt.Sprintf("%s-GR-%03d", shipment.PORef, grIdx)

	operatorName := operatorNameFromContext(ctx)

	gr := models.GoodsReceipt{
		ID:           grID,
		PORef:        shipment.PORef,
		VendorID:     shipment.SupplierID,
		Status:       models.GRStatusPending,
		ArrivalDate:  time.Now(),
		OperatorID:   operatorID,
		OperatorName: operatorName,
	}
	if err := tx.Create(&gr).Error; err != nil {
		tx.Rollback()
		return err
	}

	for _, item := range poItems {
		grLine := models.GRLineItem{
			ID:         uuid.New(),
			GRID:       grID,
			SKU:        item.SKU,
			Name:       item.Description,
			OrderedQty: item.OrderedQty,
		}
		if err := tx.Create(&grLine).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	observability.DefaultRegistry.Counter(observability.MetricShipmentDelivered).Inc()
	observability.DefaultRegistry.Counter(observability.MetricGRCreated).Inc()
	return tx.Commit().Error
}

func (s *shipmentService) TransitionState(ctx context.Context, shipmentID string, newState models.ShipmentStatus) error {
	var shipment models.Shipment
	if err := s.db.WithContext(ctx).First(&shipment, "id = ?", shipmentID).Error; err != nil {
		return ErrNotFound
	}
	if !validShipmentTransition(shipment.Status, newState) {
		return ErrStateRegression
	}
	return s.db.WithContext(ctx).Model(&shipment).Update("status", newState).Error
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
	hydrateShipmentSupplierNames(shipments)
	return shipments, meta, nil
}

func (s *shipmentService) FindAllShipments(ctx context.Context) ([]models.Shipment, error) {
	var shipments []models.Shipment
	if err := s.db.WithContext(ctx).Preload("Items").Preload("Supplier").Order("created_at DESC").Find(&shipments).Error; err != nil {
		return nil, err
	}
	hydrateShipmentSupplierNames(shipments)
	return shipments, nil
}

func (s *shipmentService) GetShipment(ctx context.Context, shipmentID string) (*models.Shipment, error) {
	var shipment models.Shipment
	if err := s.db.WithContext(ctx).Preload("Items").Preload("Supplier").First(&shipment, "id = ?", shipmentID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	hydrateShipmentSupplierName(&shipment)
	return &shipment, nil
}

func (s *shipmentService) CreateShipment(ctx context.Context, shipment *models.Shipment) error {
	var po models.PurchaseOrder
	if err := s.db.WithContext(ctx).First(&po, "id = ?", shipment.PORef).Error; err != nil {
		return ErrNotFound
	}
	if shipment != nil && shipment.SupplierID != uuid.Nil {
		var supplier models.Supplier
		if err := s.db.WithContext(ctx).First(&supplier, "id = ?", shipment.SupplierID).Error; err == nil {
			shipment.SupplierName = supplier.Name
		}
	}

	tx := s.db.WithContext(ctx).Begin()
	if err := tx.Create(shipment).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Clone PO line items as shipment items
	var poItems []models.POLineItem
	tx.Where("po_id = ?", po.ID).Find(&poItems)
	for _, item := range poItems {
		shipmentItem := models.ShipmentItem{
			ID:          uuid.New(),
			ShipmentID:  shipment.ID,
			SKU:         item.SKU,
			Description: item.Description,
			Qty:         item.OrderedQty,
		}
		if err := tx.Create(&shipmentItem).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
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

func (s *shipmentServiceRepo) MarkDelivered(ctx context.Context, shipmentID string, operatorID string) error {
	shipment, err := s.repo.GetShipmentByID(ctx, shipmentID)
	if err != nil || shipment == nil {
		return ErrNotFound
	}
	if shipment.Status != models.ShipmentStatusInTransit {
		return ErrInvalidTransition
	}

	shipment.Status = models.ShipmentStatusDelivered
	if err := s.repo.UpdateShipment(ctx, shipment); err != nil {
		return err
	}

	// Auto-create GR from shipment
	if s.grRepo != nil && s.poRepo != nil {
		poItems, _ := s.poRepo.GetPOLineItemsByPOID(ctx, shipment.PORef)
		existingGRs, _ := s.grRepo.FindGRsByPOID(ctx, shipment.PORef)
		grIdx := len(existingGRs) + 1
		grID := fmt.Sprintf("%s-GR-%03d", shipment.PORef, grIdx)

		operatorName := operatorNameFromContext(ctx)

		gr := &models.GoodsReceipt{
			ID:           grID,
			PORef:        shipment.PORef,
			VendorID:     shipment.SupplierID,
			Status:       models.GRStatusPending,
			ArrivalDate:  time.Now(),
			OperatorID:   operatorID,
			OperatorName: operatorName,
		}
		if err := s.grRepo.CreateGR(ctx, gr); err != nil {
			return err
		}

		for _, item := range poItems {
			grLine := &models.GRLineItem{
				ID:         uuid.New(),
				GRID:       grID,
				SKU:        item.SKU,
				Name:       item.Description,
				OrderedQty: item.OrderedQty,
			}
			_ = s.grRepo.SaveGRLineItem(ctx, grLine)
		}
	}

	observability.DefaultRegistry.Counter(observability.MetricShipmentDelivered).Inc()
	observability.DefaultRegistry.Counter(observability.MetricGRCreated).Inc()
	return nil
}

func (s *shipmentServiceRepo) TransitionState(ctx context.Context, shipmentID string, newState models.ShipmentStatus) error {
	shipment, err := s.repo.GetShipmentByID(ctx, shipmentID)
	if err != nil || shipment == nil {
		return ErrNotFound
	}
	if !validShipmentTransition(shipment.Status, newState) {
		return ErrStateRegression
	}
	shipment.Status = newState
	return s.repo.UpdateShipment(ctx, shipment)
}

func (s *shipmentServiceRepo) ListShipments(ctx context.Context, status string, params pagination.Params) ([]models.Shipment, *pagination.Meta, error) {
	shipments, meta, err := s.repo.ListShipments(ctx, status, params)
	if err != nil {
		return nil, nil, err
	}
	hydrateShipmentSupplierNames(shipments)
	return shipments, meta, nil
}

func (s *shipmentServiceRepo) FindAllShipments(ctx context.Context) ([]models.Shipment, error) {
	return s.repo.FindAllShipments(ctx)
}

func (s *shipmentServiceRepo) GetShipment(ctx context.Context, shipmentID string) (*models.Shipment, error) {
	shipment, err := s.repo.GetShipmentByID(ctx, shipmentID)
	if err != nil || shipment == nil {
		return nil, ErrNotFound
	}
	hydrateShipmentSupplierName(shipment)
	return shipment, nil
}

func (s *shipmentServiceRepo) CreateShipment(ctx context.Context, shipment *models.Shipment) error {
	if s.poRepo != nil {
		if _, err := s.poRepo.GetPOByID(ctx, shipment.PORef); err != nil {
			return ErrNotFound
		}
	}
	if s.vendorRepo != nil && shipment != nil && shipment.SupplierID != uuid.Nil {
		if supplier, err := s.vendorRepo.GetSupplierByID(ctx, shipment.SupplierID); err == nil && supplier != nil {
			shipment.SupplierName = supplier.Name
		}
	}
	if err := s.repo.CreateShipment(ctx, shipment); err != nil {
		return err
	}

	// Clone PO line items as shipment items
	if s.poRepo != nil {
		poItems, _ := s.poRepo.GetPOLineItemsByPOID(ctx, shipment.PORef)
		for _, item := range poItems {
			shipmentItem := &models.ShipmentItem{
				ID:          uuid.New(),
				ShipmentID:  shipment.ID,
				SKU:         item.SKU,
				Description: item.Description,
				Qty:         item.OrderedQty,
			}
			_ = s.repo.CreateShipmentItem(ctx, shipmentItem)
		}
	}

	return nil
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

func hydrateShipmentSupplierNames(shipments []models.Shipment) {
	for i := range shipments {
		hydrateShipmentSupplierName(&shipments[i])
	}
}

func hydrateShipmentSupplierName(shipment *models.Shipment) {
	if shipment != nil && shipment.Supplier != nil {
		shipment.SupplierName = shipment.Supplier.Name
	}
}

func generateShipmentID(year int, count int64) string {
	return fmt.Sprintf("SHP-%d-%03d", year, count+1)
}

func validShipmentTransition(current, new models.ShipmentStatus) bool {
	order := []models.ShipmentStatus{
		models.ShipmentStatusScheduled,
		models.ShipmentStatusInTransit,
		models.ShipmentStatusDelivered,
		models.ShipmentStatusDelayed,
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
	if new == models.ShipmentStatusDelayed {
		return current == models.ShipmentStatusInTransit
	}
	if current == models.ShipmentStatusDelayed {
		return new == models.ShipmentStatusInTransit
	}
	return newIdx > currentIdx
}
