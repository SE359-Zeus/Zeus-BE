package repository

import (
	"context"

	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/pagination"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type MockVendorRepository struct {
	mock.Mock
}

func (m *MockVendorRepository) GetSupplierByID(ctx context.Context, id uuid.UUID) (*models.Supplier, error) {
	args := m.Called(ctx, id)
	if args.Get(0) != nil {
		return args.Get(0).(*models.Supplier), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockVendorRepository) FindSkuMappingsBySKU(ctx context.Context, sku string) ([]models.SkuMapping, error) {
	args := m.Called(ctx, sku)
	if args.Get(0) != nil {
		return args.Get(0).([]models.SkuMapping), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockVendorRepository) FindSkuMappingsBySupplierID(ctx context.Context, supplierID uuid.UUID) ([]models.SkuMapping, error) {
	args := m.Called(ctx, supplierID)
	if args.Get(0) != nil {
		return args.Get(0).([]models.SkuMapping), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockVendorRepository) UpdateSupplier(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	args := m.Called(ctx, id, updates)
	return args.Error(0)
}

func (m *MockVendorRepository) FindGoodsReceiptsByVendor(ctx context.Context, vendorID uuid.UUID) ([]models.GoodsReceipt, error) {
	args := m.Called(ctx, vendorID)
	if args.Get(0) != nil {
		return args.Get(0).([]models.GoodsReceipt), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockVendorRepository) CountGoodsReceiptsByVendor(ctx context.Context, vendorID uuid.UUID) (int64, error) {
	args := m.Called(ctx, vendorID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockVendorRepository) FindGRLineItemsByGRID(ctx context.Context, grID string) ([]models.GRLineItem, error) {
	args := m.Called(ctx, grID)
	if args.Get(0) != nil {
		return args.Get(0).([]models.GRLineItem), args.Error(1)
	}
	return nil, args.Error(1)
}

type MockPORepository struct {
	mock.Mock
}

func (m *MockPORepository) GetPOByID(ctx context.Context, id string) (*models.PurchaseOrder, error) {
	args := m.Called(ctx, id)
	if args.Get(0) != nil {
		return args.Get(0).(*models.PurchaseOrder), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPORepository) CreatePO(ctx context.Context, po *models.PurchaseOrder) error {
	args := m.Called(ctx, po)
	return args.Error(0)
}

func (m *MockPORepository) SavePO(ctx context.Context, po *models.PurchaseOrder) error {
	args := m.Called(ctx, po)
	return args.Error(0)
}

func (m *MockPORepository) UpdatePOStatus(ctx context.Context, id string, status models.POStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockPORepository) CreatePOLineItem(ctx context.Context, item *models.POLineItem) error {
	args := m.Called(ctx, item)
	return args.Error(0)
}

func (m *MockPORepository) GetPOLineItemsByPOID(ctx context.Context, poID string) ([]models.POLineItem, error) {
	args := m.Called(ctx, poID)
	if args.Get(0) != nil {
		return args.Get(0).([]models.POLineItem), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPORepository) CountPOsByYearPattern(ctx context.Context, year int, pattern string) (int64, error) {
	args := m.Called(ctx, year, pattern)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockPORepository) FindPOByVendorAndStatuses(ctx context.Context, vendorID uuid.UUID, statuses []models.POStatus) (*models.PurchaseOrder, error) {
	args := m.Called(ctx, vendorID, statuses)
	if args.Get(0) != nil {
		return args.Get(0).(*models.PurchaseOrder), args.Error(1)
	}
	return nil, args.Error(1)
}

type MockInventoryRepository struct {
	mock.Mock
}

func (m *MockInventoryRepository) GetProductByID(ctx context.Context, id uuid.UUID) (*models.Product, error) {
	args := m.Called(ctx, id)
	if args.Get(0) != nil {
		return args.Get(0).(*models.Product), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockInventoryRepository) ListProducts(ctx context.Context, params pagination.Params, q string) ([]models.Product, *pagination.Meta, error) {
	args := m.Called(ctx, params, q)
	if args.Get(0) != nil {
		return args.Get(0).([]models.Product), args.Get(1).(*pagination.Meta), args.Error(2)
	}
	return nil, args.Get(1).(*pagination.Meta), args.Error(2)
}

func (m *MockInventoryRepository) CreateProduct(ctx context.Context, p *models.Product) error {
	args := m.Called(ctx, p)
	return args.Error(0)
}

func (m *MockInventoryRepository) UpdateProduct(ctx context.Context, id uuid.UUID, fields map[string]any) (int64, error) {
	args := m.Called(ctx, id, fields)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockInventoryRepository) GetProductModelByCode(ctx context.Context, code string) (*models.ProductModel, error) {
	args := m.Called(ctx, code)
	if args.Get(0) != nil {
		return args.Get(0).(*models.ProductModel), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockInventoryRepository) CreateProductModel(ctx context.Context, m2 *models.ProductModel) error {
	args := m.Called(ctx, m2)
	return args.Error(0)
}

func (m *MockInventoryRepository) GetPartByID(ctx context.Context, id uuid.UUID) (*models.Part, error) {
	args := m.Called(ctx, id)
	if args.Get(0) != nil {
		return args.Get(0).(*models.Part), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockInventoryRepository) ListParts(ctx context.Context, catalogID *uuid.UUID, productID *uuid.UUID, conditionID *int32, params pagination.Params, q string) ([]models.Part, *pagination.Meta, error) {
	args := m.Called(ctx, catalogID, productID, conditionID, params, q)
	if args.Get(0) != nil {
		return args.Get(0).([]models.Part), args.Get(1).(*pagination.Meta), args.Error(2)
	}
	return nil, args.Get(1).(*pagination.Meta), args.Error(2)
}

func (m *MockInventoryRepository) CreatePart(ctx context.Context, p *models.Part) error {
	args := m.Called(ctx, p)
	return args.Error(0)
}

func (m *MockInventoryRepository) UpdatePart(ctx context.Context, id uuid.UUID, fields map[string]any) (int64, error) {
	args := m.Called(ctx, id, fields)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockInventoryRepository) UpdatePartFields(ctx context.Context, id uuid.UUID, updates map[string]interface{}) (int64, error) {
	args := m.Called(ctx, id, updates)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockInventoryRepository) GetPartCatalogByID(ctx context.Context, id uuid.UUID) (*models.PartCatalog, error) {
	args := m.Called(ctx, id)
	if args.Get(0) != nil {
		return args.Get(0).(*models.PartCatalog), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockInventoryRepository) ListPartCatalog(ctx context.Context, typeID *int32, params pagination.Params, q string) ([]models.PartCatalog, *pagination.Meta, error) {
	args := m.Called(ctx, typeID, params, q)
	if args.Get(0) != nil {
		return args.Get(0).([]models.PartCatalog), args.Get(1).(*pagination.Meta), args.Error(2)
	}
	return nil, args.Get(1).(*pagination.Meta), args.Error(2)
}

func (m *MockInventoryRepository) CreatePartCatalog(ctx context.Context, pc *models.PartCatalog) error {
	args := m.Called(ctx, pc)
	return args.Error(0)
}

func (m *MockInventoryRepository) UpdatePartCatalogFieldsBySKU(ctx context.Context, sku string, updates map[string]interface{}) (int64, error) {
	args := m.Called(ctx, sku, updates)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockInventoryRepository) DeletePartCatalogBySKU(ctx context.Context, sku string) (int64, error) {
	args := m.Called(ctx, sku)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockInventoryRepository) GetPartCatalogBySKU(ctx context.Context, sku string) (*models.PartCatalog, error) {
	args := m.Called(ctx, sku)
	if args.Get(0) != nil {
		return args.Get(0).(*models.PartCatalog), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockInventoryRepository) GetComponentStockBySKU(ctx context.Context, sku string) (*models.ComponentStock, error) {
	args := m.Called(ctx, sku)
	if args.Get(0) != nil {
		return args.Get(0).(*models.ComponentStock), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockInventoryRepository) CreateComponentStock(ctx context.Context, stock *models.ComponentStock) error {
	args := m.Called(ctx, stock)
	return args.Error(0)
}

func (m *MockInventoryRepository) UpdateComponentStockFieldsBySKU(ctx context.Context, sku string, updates map[string]interface{}) (int64, error) {
	args := m.Called(ctx, sku, updates)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockInventoryRepository) DeleteComponentStockBySKU(ctx context.Context, sku string) (int64, error) {
	args := m.Called(ctx, sku)
	return args.Get(0).(int64), args.Error(1)
}

type MockShipmentRepository struct {
	mock.Mock
}

func (m *MockShipmentRepository) GetShipmentByID(ctx context.Context, id string) (*models.Shipment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) != nil {
		return args.Get(0).(*models.Shipment), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockShipmentRepository) UpdateShipment(ctx context.Context, shipment *models.Shipment) error {
	args := m.Called(ctx, shipment)
	return args.Error(0)
}

func (m *MockShipmentRepository) UpdateShipmentFields(ctx context.Context, id string, fields map[string]interface{}) error {
	args := m.Called(ctx, id, fields)
	return args.Error(0)
}

func (m *MockShipmentRepository) GetShipmentItemsByShipmentID(ctx context.Context, shipmentID string) ([]models.ShipmentItem, error) {
	args := m.Called(ctx, shipmentID)
	if args.Get(0) != nil {
		return args.Get(0).([]models.ShipmentItem), args.Error(1)
	}
	return nil, args.Error(1)
}

type MockStockRepository struct {
	mock.Mock
}

func (m *MockStockRepository) GetStockBySKU(ctx context.Context, sku string) (*models.ComponentStock, error) {
	args := m.Called(ctx, sku)
	if args.Get(0) != nil {
		return args.Get(0).(*models.ComponentStock), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockStockRepository) SaveStock(ctx context.Context, stock *models.ComponentStock) error {
	args := m.Called(ctx, stock)
	return args.Error(0)
}

func (m *MockStockRepository) UpsertStock(ctx context.Context, stock *models.ComponentStock) error {
	args := m.Called(ctx, stock)
	return args.Error(0)
}

type MockGoodsReceiptRepository struct {
	mock.Mock
}

func (m *MockGoodsReceiptRepository) GetGRByID(ctx context.Context, id string) (*models.GoodsReceipt, error) {
	args := m.Called(ctx, id)
	if args.Get(0) != nil {
		return args.Get(0).(*models.GoodsReceipt), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockGoodsReceiptRepository) UpdateGR(ctx context.Context, gr *models.GoodsReceipt) error {
	args := m.Called(ctx, gr)
	return args.Error(0)
}

func (m *MockGoodsReceiptRepository) UpdateGRFields(ctx context.Context, id string, fields map[string]interface{}) error {
	args := m.Called(ctx, id, fields)
	return args.Error(0)
}

func (m *MockGoodsReceiptRepository) FindGRLineItemsByGRID(ctx context.Context, grID string) ([]models.GRLineItem, error) {
	args := m.Called(ctx, grID)
	if args.Get(0) != nil {
		return args.Get(0).([]models.GRLineItem), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockGoodsReceiptRepository) SaveGRLineItem(ctx context.Context, item *models.GRLineItem) error {
	args := m.Called(ctx, item)
	return args.Error(0)
}
