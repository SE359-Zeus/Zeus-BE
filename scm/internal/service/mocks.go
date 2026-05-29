package service

import (
	"context"

	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/pagination"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type MockVendorService struct {
	mock.Mock
}

func (m *MockVendorService) GetOptimalSupplier(ctx context.Context, sku string) (*models.Supplier, *models.SkuMapping, error) {
	args := m.Called(ctx, sku)
	if args.Get(0) != nil {
		return args.Get(0).(*models.Supplier), args.Get(1).(*models.SkuMapping), args.Error(2)
	}
	return nil, nil, args.Error(2)
}

func (m *MockVendorService) UpdateSupplierMetrics(ctx context.Context, supplierID uuid.UUID) error {
	args := m.Called(ctx, supplierID)
	return args.Error(0)
}

func (m *MockVendorService) ListSuppliers(ctx context.Context, tier string, params pagination.Params, q string) ([]models.Supplier, *pagination.Meta, error) {
	args := m.Called(ctx, tier, params, q)
	var suppliers []models.Supplier
	if args.Get(0) != nil {
		suppliers = args.Get(0).([]models.Supplier)
	}
	var meta *pagination.Meta
	if args.Get(1) != nil {
		meta = args.Get(1).(*pagination.Meta)
	}
	return suppliers, meta, args.Error(2)
}

func (m *MockVendorService) CreateSupplier(ctx context.Context, supplier *models.Supplier) error {
	args := m.Called(ctx, supplier)
	return args.Error(0)
}

func (m *MockVendorService) CreateSkuMapping(ctx context.Context, mapping *models.SkuMapping) error {
	args := m.Called(ctx, mapping)
	return args.Error(0)
}

func (m *MockVendorService) GetSupplierMetrics(ctx context.Context) (int64, float64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Get(1).(float64), args.Error(2)
}

func (m *MockVendorService) FindAllSuppliersWithMappings(ctx context.Context) ([]models.Supplier, error) {
	args := m.Called(ctx)
	var suppliers []models.Supplier
	if args.Get(0) != nil {
		suppliers = args.Get(0).([]models.Supplier)
	}
	return suppliers, args.Error(1)
}

type MockPOService struct {
	mock.Mock
}

func (m *MockPOService) CreateDraft(ctx context.Context, vendorID uuid.UUID, targetBuild string) (*models.PurchaseOrder, error) {
	args := m.Called(ctx, vendorID, targetBuild)
	if args.Get(0) != nil {
		return args.Get(0).(*models.PurchaseOrder), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPOService) AddLineItemWithLock(ctx context.Context, poID string, sku string, qty int) error {
	args := m.Called(ctx, poID, sku, qty)
	return args.Error(0)
}

func (m *MockPOService) ApprovePO(ctx context.Context, poID string) error {
	args := m.Called(ctx, poID)
	return args.Error(0)
}

func (m *MockPOService) TransitionState(ctx context.Context, poID string, newState models.POStatus) error {
	args := m.Called(ctx, poID, newState)
	return args.Error(0)
}

func (m *MockPOService) ListPOs(ctx context.Context, params pagination.Params, q string) ([]models.PurchaseOrder, *pagination.Meta, error) {
	args := m.Called(ctx, params, q)
	var pos []models.PurchaseOrder
	if args.Get(0) != nil {
		pos = args.Get(0).([]models.PurchaseOrder)
	}
	var meta *pagination.Meta
	if args.Get(1) != nil {
		meta = args.Get(1).(*pagination.Meta)
	}
	return pos, meta, args.Error(2)
}

func (m *MockPOService) GetPO(ctx context.Context, poID string) (*models.PurchaseOrder, error) {
	args := m.Called(ctx, poID)
	if args.Get(0) != nil {
		return args.Get(0).(*models.PurchaseOrder), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPOService) CreatePO(ctx context.Context, po *models.PurchaseOrder) error {
	args := m.Called(ctx, po)
	return args.Error(0)
}

type MockInventoryService struct {
	mock.Mock
}

func (m *MockInventoryService) GetProduct(ctx context.Context, id uuid.UUID) (*models.Product, error) {
	args := m.Called(ctx, id)
	if args.Get(0) != nil {
		return args.Get(0).(*models.Product), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockInventoryService) ListProducts(ctx context.Context, params pagination.Params, q string) ([]models.Product, *pagination.Meta, error) {
	args := m.Called(ctx, params, q)
	if args.Get(0) != nil {
		return args.Get(0).([]models.Product), args.Get(1).(*pagination.Meta), args.Error(2)
	}
	return nil, args.Get(1).(*pagination.Meta), args.Error(2)
}

func (m *MockInventoryService) CreateProduct(ctx context.Context, p *models.Product) error {
	args := m.Called(ctx, p)
	return args.Error(0)
}

func (m *MockInventoryService) UpdateProduct(ctx context.Context, id uuid.UUID, fields map[string]any) (*models.Product, error) {
	args := m.Called(ctx, id, fields)
	if args.Get(0) != nil {
		return args.Get(0).(*models.Product), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockInventoryService) GetProductModel(ctx context.Context, code string) (*models.ProductModel, error) {
	args := m.Called(ctx, code)
	if args.Get(0) != nil {
		return args.Get(0).(*models.ProductModel), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockInventoryService) CreateProductModel(ctx context.Context, model *models.ProductModel) error {
	args := m.Called(ctx, model)
	return args.Error(0)
}

func (m *MockInventoryService) GetPart(ctx context.Context, id uuid.UUID) (*models.Part, error) {
	args := m.Called(ctx, id)
	if args.Get(0) != nil {
		return args.Get(0).(*models.Part), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockInventoryService) ListParts(ctx context.Context, catalogID *uuid.UUID, productID *uuid.UUID, conditionID *int32, params pagination.Params, q string) ([]models.Part, *pagination.Meta, error) {
	args := m.Called(ctx, catalogID, productID, conditionID, params, q)
	if args.Get(0) != nil {
		return args.Get(0).([]models.Part), args.Get(1).(*pagination.Meta), args.Error(2)
	}
	return nil, args.Get(1).(*pagination.Meta), args.Error(2)
}

func (m *MockInventoryService) CreatePart(ctx context.Context, p *models.Part) error {
	args := m.Called(ctx, p)
	return args.Error(0)
}

func (m *MockInventoryService) UpdatePart(ctx context.Context, id uuid.UUID, fields map[string]any) (*models.Part, error) {
	args := m.Called(ctx, id, fields)
	if args.Get(0) != nil {
		return args.Get(0).(*models.Part), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockInventoryService) UpdatePartCondition(ctx context.Context, partID uuid.UUID, conditionID int32) error {
	args := m.Called(ctx, partID, conditionID)
	return args.Error(0)
}

func (m *MockInventoryService) MarkPartScrapped(ctx context.Context, partID uuid.UUID) error {
	args := m.Called(ctx, partID)
	return args.Error(0)
}

func (m *MockInventoryService) InstallPart(ctx context.Context, partID uuid.UUID, productID uuid.UUID) error {
	args := m.Called(ctx, partID, productID)
	return args.Error(0)
}

func (m *MockInventoryService) RemovePart(ctx context.Context, partID uuid.UUID) error {
	args := m.Called(ctx, partID)
	return args.Error(0)
}

func (m *MockInventoryService) GetPartCatalog(ctx context.Context, id uuid.UUID) (*models.PartCatalog, error) {
	args := m.Called(ctx, id)
	if args.Get(0) != nil {
		return args.Get(0).(*models.PartCatalog), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockInventoryService) ListPartCatalog(ctx context.Context, typeID *int32, params pagination.Params, q string) ([]models.PartCatalog, *pagination.Meta, error) {
	args := m.Called(ctx, typeID, params, q)
	if args.Get(0) != nil {
		return args.Get(0).([]models.PartCatalog), args.Get(1).(*pagination.Meta), args.Error(2)
	}
	return nil, args.Get(1).(*pagination.Meta), args.Error(2)
}

func (m *MockInventoryService) CreatePartCatalog(ctx context.Context, pc *models.PartCatalog, price float64) error {
	args := m.Called(ctx, pc, price)
	return args.Error(0)
}

func (m *MockInventoryService) UpdatePartCatalogBySKU(ctx context.Context, sku string, fields map[string]any) (*models.PartCatalog, error) {
	args := m.Called(ctx, sku, fields)
	if args.Get(0) != nil {
		return args.Get(0).(*models.PartCatalog), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockInventoryService) DeletePartCatalogBySKU(ctx context.Context, sku string) error {
	args := m.Called(ctx, sku)
	return args.Error(0)
}

func (m *MockInventoryService) GetPartCatalogBySKU(ctx context.Context, sku string) (*models.PartCatalog, float64, int, error) {
	args := m.Called(ctx, sku)
	if args.Get(0) != nil {
		return args.Get(0).(*models.PartCatalog), args.Get(1).(float64), args.Get(2).(int), args.Error(3)
	}
	return nil, 0, 0, args.Error(3)
}

func (m *MockInventoryService) ListStocks(ctx context.Context, params pagination.Params, status, q string) ([]models.ComponentStock, *pagination.Meta, error) {
	args := m.Called(ctx, params, status, q)
	if args.Get(0) != nil {
		return args.Get(0).([]models.ComponentStock), args.Get(1).(*pagination.Meta), args.Error(2)
	}
	return nil, args.Get(1).(*pagination.Meta), args.Error(2)
}

func (m *MockInventoryService) CreateComponentStock(ctx context.Context, stock *models.ComponentStock) error {
	args := m.Called(ctx, stock)
	return args.Error(0)
}

func (m *MockInventoryService) GetStockBySKU(ctx context.Context, sku string) (*models.ComponentStock, error) {
	args := m.Called(ctx, sku)
	if args.Get(0) != nil {
		return args.Get(0).(*models.ComponentStock), args.Error(1)
	}
	return nil, args.Error(1)
}

type MockShipmentService struct {
	mock.Mock
}

func (m *MockShipmentService) AcquireDispatchLock(ctx context.Context, shipmentID string, operatorID string) error {
	args := m.Called(ctx, shipmentID, operatorID)
	return args.Error(0)
}

func (m *MockShipmentService) DispatchShipment(ctx context.Context, shipmentID string, operatorID string) error {
	args := m.Called(ctx, shipmentID, operatorID)
	return args.Error(0)
}

func (m *MockShipmentService) ListShipments(ctx context.Context, status string, params pagination.Params) ([]models.Shipment, *pagination.Meta, error) {
	args := m.Called(ctx, status, params)
	var shipments []models.Shipment
	if args.Get(0) != nil {
		shipments = args.Get(0).([]models.Shipment)
	}
	var meta *pagination.Meta
	if args.Get(1) != nil {
		meta = args.Get(1).(*pagination.Meta)
	}
	return shipments, meta, args.Error(2)
}

func (m *MockShipmentService) GetShipment(ctx context.Context, shipmentID string) (*models.Shipment, error) {
	args := m.Called(ctx, shipmentID)
	if args.Get(0) != nil {
		return args.Get(0).(*models.Shipment), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockShipmentService) CreateShipment(ctx context.Context, shipment *models.Shipment) error {
	args := m.Called(ctx, shipment)
	return args.Error(0)
}

func (m *MockShipmentService) GetMetrics(ctx context.Context) (total int64, inTransit int64, delayed int64, onTimeRate float64, err error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Get(1).(int64), args.Get(2).(int64), args.Get(3).(float64), args.Error(4)
}

func (m *MockShipmentService) ListCarriers(ctx context.Context) ([]models.Carrier, error) {
	args := m.Called(ctx)
	if args.Get(0) != nil {
		return args.Get(0).([]models.Carrier), args.Error(1)
	}
	return nil, args.Error(1)
}

type MockGoodsReceiptService struct {
	mock.Mock
}

func (m *MockGoodsReceiptService) AcquireLock(ctx context.Context, grID string, operatorID string) error {
	args := m.Called(ctx, grID, operatorID)
	return args.Error(0)
}

func (m *MockGoodsReceiptService) ProcessBlindReceipt(ctx context.Context, grID string, operatorID string, counts map[string]struct {
	Received  int
	Defective int
}) error {
	args := m.Called(ctx, grID, operatorID, counts)
	return args.Error(0)
}

func (m *MockGoodsReceiptService) ReleaseLock(ctx context.Context, grID string) error {
	args := m.Called(ctx, grID)
	return args.Error(0)
}

func (m *MockGoodsReceiptService) ListGRs(ctx context.Context, status string, params pagination.Params) ([]models.GoodsReceipt, *pagination.Meta, error) {
	args := m.Called(ctx, status, params)
	var grs []models.GoodsReceipt
	if args.Get(0) != nil {
		grs = args.Get(0).([]models.GoodsReceipt)
	}
	var meta *pagination.Meta
	if args.Get(1) != nil {
		meta = args.Get(1).(*pagination.Meta)
	}
	return grs, meta, args.Error(2)
}

func (m *MockGoodsReceiptService) GetGR(ctx context.Context, grID string) (*models.GoodsReceipt, error) {
	args := m.Called(ctx, grID)
	if args.Get(0) != nil {
		return args.Get(0).(*models.GoodsReceipt), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockGoodsReceiptService) GetMetrics(ctx context.Context) (pending int64, completedToday int64, discrepancies int64, queue int64, err error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Get(1).(int64), args.Get(2).(int64), args.Get(3).(int64), args.Error(4)
}
