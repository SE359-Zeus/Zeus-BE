package consumer

import (
	"context"
	"testing"
	"time"

	"zeus-mrp-service/internal/models"
	"zeus-mrp-service/internal/service"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockMRPRepository matches internal/service's expected interface
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
	if args.Get(0) != nil {
		return args.Get(0).([]models.BomEntry), args.Int(1), args.Error(2)
	}
	return nil, args.Int(1), args.Error(2)
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

func (m *MockMRPRepository) GetPartInventory(ctx context.Context, partID uuid.UUID) (int, error) {
	args := m.Called(ctx, partID)
	return args.Int(0), args.Error(1)
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

func TestOrderConsumer_ProcessOrderPayload_Success(t *testing.T) {
	mockRepo := new(MockMRPRepository)
	mrpService := service.NewProductionService(mockRepo)

	orderID := uuid.New().String()
	requiredDate := time.Now().Add(48 * time.Hour).Truncate(time.Second)
	payload := OrderCreatedPayload{
		OrderID:      orderID,
		RequiredDate: requiredDate,
		Items: []OrderPayloadItem{
			{SKU: "ZEUS-PHONE", RequestedQty: 10},
		},
	}

	mockRepo.On("CreateProductionOrder", mock.Anything, mock.MatchedBy(func(order *models.ProductionOrder) bool {
		return order.ProductModelCode == "ZEUS-PHONE" && order.TargetQuantity == 10
	})).Return(nil)

	mockRepo.On("GetBOMByModelCode", mock.Anything, "ZEUS-PHONE").Return([]models.BomEntry{}, nil)

	mockRepo.On("GetProductionOrder", mock.Anything, mock.Anything).Return(&models.ProductionOrder{
		ID:               uuid.Nil,
		ProductModelCode: "ZEUS-PHONE",
		TargetQuantity:   10,
		Status:           models.StatusPlanned,
	}, nil)

	consumer := NewOrderConsumer("amqp://localhost", mrpService)
	err := consumer.processOrderPayload(context.Background(), payload)
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

func TestOrderConsumer_ProcessOrderPayload_NoItems(t *testing.T) {
	mockRepo := new(MockMRPRepository)
	mrpService := service.NewProductionService(mockRepo)

	payload := OrderCreatedPayload{OrderID: uuid.New().String(), Items: []OrderPayloadItem{}}

	consumer := NewOrderConsumer("amqp://localhost", mrpService)
	err := consumer.processOrderPayload(context.Background(), payload)
	assert.NoError(t, err)

	mockRepo.AssertNotCalled(t, "CreateProductionOrder", mock.Anything, mock.Anything)
}

func TestOrderConsumer_ProcessOrderPayload_InvalidItemsIgnored(t *testing.T) {
	mockRepo := new(MockMRPRepository)
	mrpService := service.NewProductionService(mockRepo)

	payload := OrderCreatedPayload{
		OrderID: uuid.New().String(),
		Items: []OrderPayloadItem{
			{SKU: "", RequestedQty: 10},
			{SKU: "MODEL-X", RequestedQty: 0},
		},
	}

	consumer := NewOrderConsumer("amqp://localhost", mrpService)
	err := consumer.processOrderPayload(context.Background(), payload)
	assert.NoError(t, err)

	mockRepo.AssertNotCalled(t, "CreateProductionOrder", mock.Anything, mock.Anything)
}

func TestOrderConsumer_ProcessOrderPayload_QtyAliasSupported(t *testing.T) {
	mockRepo := new(MockMRPRepository)
	mrpService := service.NewProductionService(mockRepo)

	payload := OrderCreatedPayload{
		OrderID: uuid.New().String(),
		Items: []OrderPayloadItem{
			{SKU: "ZEUS-PHONE", Qty: 7},
		},
	}

	mockRepo.On("CreateProductionOrder", mock.Anything, mock.MatchedBy(func(order *models.ProductionOrder) bool {
		return order.ProductModelCode == "ZEUS-PHONE" && order.TargetQuantity == 7
	})).Return(nil)
	mockRepo.On("GetBOMByModelCode", mock.Anything, "ZEUS-PHONE").Return([]models.BomEntry{}, nil)
	mockRepo.On("GetProductionOrder", mock.Anything, mock.Anything).Return(&models.ProductionOrder{
		ID:               uuid.Nil,
		ProductModelCode: "ZEUS-PHONE",
		TargetQuantity:   7,
		Status:           models.StatusPlanned,
	}, nil)

	consumer := NewOrderConsumer("amqp://localhost", mrpService)
	err := consumer.processOrderPayload(context.Background(), payload)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}
