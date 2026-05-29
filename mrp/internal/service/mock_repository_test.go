package service

import (
	"context"
	"time"
	"zeus-mrp-service/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type MockMRPRepository struct {
	mock.Mock
}

func (m *MockMRPRepository) CreateProductionOrder(ctx context.Context, order *models.ProductionOrder) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

func (m *MockMRPRepository) GetProductionOrder(ctx context.Context, id uuid.UUID) (*models.ProductionOrder, error) {
	args := m.Called(ctx, id)
	if args.Get(0) != nil {
		return args.Get(0).(*models.ProductionOrder), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockMRPRepository) GetOpenProductionOrders(ctx context.Context) ([]models.ProductionOrder, error) {
	args := m.Called(ctx)
	if args.Get(0) != nil {
		return args.Get(0).([]models.ProductionOrder), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockMRPRepository) UpdateProductionOrderStatus(ctx context.Context, id uuid.UUID, status models.ProductionOrderStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockMRPRepository) CreateBOMEntries(ctx context.Context, entries []models.BomEntry) error {
	args := m.Called(ctx, entries)
	return args.Error(0)
}

func (m *MockMRPRepository) DeleteBOMEntriesByModelCode(ctx context.Context, modelCode string) error {
	args := m.Called(ctx, modelCode)
	return args.Error(0)
}

func (m *MockMRPRepository) GetBOMByModelCode(ctx context.Context, modelCode string) ([]models.BomEntry, error) {
	args := m.Called(ctx, modelCode)
	if args.Get(0) != nil {
		return args.Get(0).([]models.BomEntry), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockMRPRepository) GetAllBOMs(ctx context.Context) ([]models.BomEntry, error) {
	args := m.Called(ctx)
	if args.Get(0) != nil {
		return args.Get(0).([]models.BomEntry), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockMRPRepository) GetPagedBOMsByAssembly(ctx context.Context, page, per int) ([]models.BomEntry, int, error) {
	args := m.Called(ctx, page, per)
	var total int
	if t := args.Get(1); t != nil {
		total = t.(int)
	}
	if args.Get(0) != nil {
		return args.Get(0).([]models.BomEntry), total, args.Error(2)
	}
	return nil, total, args.Error(2)
}

func (m *MockMRPRepository) GetWhereUsedByPartID(ctx context.Context, partID uuid.UUID) ([]models.BomEntry, error) {
	args := m.Called(ctx, partID)
	if args.Get(0) != nil {
		return args.Get(0).([]models.BomEntry), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockMRPRepository) CreateShortageLog(ctx context.Context, log *models.ShortageLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func (m *MockMRPRepository) GetShortagesByOrderID(ctx context.Context, orderID uuid.UUID) ([]models.ShortageLog, error) {
	args := m.Called(ctx, orderID)
	if args.Get(0) != nil {
		return args.Get(0).([]models.ShortageLog), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockMRPRepository) GetAggregatedShortages(ctx context.Context) ([]models.BOMExplosionResult, error) {
	args := m.Called(ctx)
	if args.Get(0) != nil {
		return args.Get(0).([]models.BOMExplosionResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockMRPRepository) GetInventoryTransactions(ctx context.Context) ([]models.InventoryTransactionDTO, error) {
	args := m.Called(ctx)
	if args.Get(0) != nil {
		return args.Get(0).([]models.InventoryTransactionDTO), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockMRPRepository) GetInventoryMetrics(ctx context.Context) (*models.InventoryMetrics, error) {
	args := m.Called(ctx)
	if args.Get(0) != nil {
		return args.Get(0).(*models.InventoryMetrics), args.Error(1)
	}
	return nil, args.Error(1)
}

func setupMockRepo() *MockMRPRepository {
	m := new(MockMRPRepository)

	// Seed default happy-path data
	m.On("GetProductionOrder", mock.Anything, mock.Anything).Return(&models.ProductionOrder{ProductModelCode: "MODEL-A", TargetQuantity: 10}, nil)
	m.On("GetOpenProductionOrders", mock.Anything).Return([]models.ProductionOrder{}, nil)
	m.On("UpdateProductionOrderStatus", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	m.On("CreateProductionOrder", mock.Anything, mock.Anything).Return(nil)
	m.On("GetBOMByModelCode", mock.Anything, mock.Anything).Return([]models.BomEntry{}, nil)
	m.On("GetAllBOMs", mock.Anything).Return([]models.BomEntry{}, nil)
	m.On("GetPagedBOMsByAssembly", mock.Anything, mock.Anything, mock.Anything).Return([]models.BomEntry{}, 0, nil)
	m.On("GetWhereUsedByPartID", mock.Anything, mock.Anything).Return([]models.BomEntry{}, nil)
	m.On("CreateBOMEntries", mock.Anything, mock.Anything).Return(nil)
	m.On("DeleteBOMEntriesByModelCode", mock.Anything, mock.Anything).Return(nil)
	m.On("CreateShortageLog", mock.Anything, mock.Anything).Return(nil)
	m.On("GetShortagesByOrderID", mock.Anything, mock.Anything).Return([]models.ShortageLog{}, nil)
	m.On("GetAggregatedShortages", mock.Anything).Return([]models.BOMExplosionResult{}, nil)
	m.On("GetInventoryTransactions", mock.Anything).Return([]models.InventoryTransactionDTO{{ID: "TXN-1", SKU: "PART-1", QtyChange: 10, RunningBalance: 10, Timestamp: time.Now()}}, nil)
	m.On("GetInventoryMetrics", mock.Anything).Return(&models.InventoryMetrics{ActiveSKUs: 154}, nil)

	return m
}

type MockSCMClient struct {
	mock.Mock
}

func (m *MockSCMClient) GetPartCatalogBySKU(ctx context.Context, sku string) (*models.Part, error) {
	args := m.Called(ctx, sku)
	if args.Get(0) != nil {
		return args.Get(0).(*models.Part), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockSCMClient) GetPartCatalogByID(ctx context.Context, id uuid.UUID) (*models.Part, error) {
	args := m.Called(ctx, id)
	if args.Get(0) != nil {
		return args.Get(0).(*models.Part), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockSCMClient) GetStockBySKU(ctx context.Context, sku string) (*models.ComponentStock, error) {
	args := m.Called(ctx, sku)
	if args.Get(0) != nil {
		return args.Get(0).(*models.ComponentStock), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockSCMClient) GetProductModelByCode(ctx context.Context, code string) (*models.ProductModel, error) {
	args := m.Called(ctx, code)
	if args.Get(0) != nil {
		return args.Get(0).(*models.ProductModel), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockSCMClient) ListStocks(ctx context.Context, page, limit int, sortBy, sortDir, q string) ([]models.ComponentStock, bool, error) {
	args := m.Called(ctx, page, limit, sortBy, sortDir, q)
	if args.Get(0) != nil {
		return args.Get(0).([]models.ComponentStock), args.Bool(1), args.Error(2)
	}
	return nil, args.Bool(1), args.Error(2)
}

func (m *MockSCMClient) CreateCatalogPart(ctx context.Context, sku, description string, price float64) (*models.Part, error) {
	args := m.Called(ctx, sku, description, price)
	if args.Get(0) != nil {
		return args.Get(0).(*models.Part), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockSCMClient) UpdateCatalogPart(ctx context.Context, sku, description string, price float64) (*models.Part, error) {
	args := m.Called(ctx, sku, description, price)
	if args.Get(0) != nil {
		return args.Get(0).(*models.Part), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockSCMClient) DeleteCatalogPart(ctx context.Context, sku string) error {
	args := m.Called(ctx, sku)
	return args.Error(0)
}

func (m *MockSCMClient) GetInventoryLedger(ctx context.Context, page, limit int, sortBy, sortDir, txnType, sku string) ([]models.InventoryLedgerEntry, bool, error) {
	args := m.Called(ctx, page, limit, sortBy, sortDir, txnType, sku)
	if args.Get(0) != nil {
		return args.Get(0).([]models.InventoryLedgerEntry), args.Bool(1), args.Error(2)
	}
	return nil, args.Bool(1), args.Error(2)
}

func (m *MockSCMClient) GetLUTs(ctx context.Context) (*models.LUTCollection, error) {
	args := m.Called(ctx)
	if args.Get(0) != nil {
		return args.Get(0).(*models.LUTCollection), args.Error(1)
	}
	return nil, args.Error(1)
}

func setupMockSCMClient() *MockSCMClient {
	m := new(MockSCMClient)
	return m
}

type MockAuditPublisher struct {
	mock.Mock
}

func (m *MockAuditPublisher) PublishJSON(ctx context.Context, queue string, payload any) error {
	args := m.Called(ctx, queue, payload)
	return args.Error(0)
}

