package consumer

import (
	"context"
	"encoding/json"
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

func (m *MockMRPRepository) HardDeleteBOMEntriesByModelCode(ctx context.Context, modelCode string) error {
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

func (m *MockMRPRepository) GetShortagesByOrderIDs(ctx context.Context, orderIDs []uuid.UUID) (map[uuid.UUID][]models.ShortageLog, error) {
	args := m.Called(ctx, orderIDs)
	if args.Get(0) != nil {
		return args.Get(0).(map[uuid.UUID][]models.ShortageLog), args.Error(1)
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

func (m *MockMRPRepository) DeleteProductionOrder(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockMRPRepository) UpdateShortageLog(ctx context.Context, log *models.ShortageLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func (m *MockMRPRepository) DeleteShortageLog(ctx context.Context, orderID uuid.UUID, partID uuid.UUID) error {
	args := m.Called(ctx, orderID, partID)
	return args.Error(0)
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
	mockRepo.On("GetShortagesByOrderID", mock.Anything, mock.Anything).Return([]models.ShortageLog{}, nil).Maybe()

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
	mockRepo.On("GetShortagesByOrderID", mock.Anything, mock.Anything).Return([]models.ShortageLog{}, nil).Maybe()
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

func TestOrderConsumer_UnmarshalSalesPayload(t *testing.T) {
	jsonStr := `{
		"order_id": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
		"client_id": "9f8e7d6c-5b4a-3f2e-1d0c-9b8a7f6e5d4c",
		"total": 999.99,
		"required_date": "2026-06-05T15:04:05Z",
		"items": [
			{"sku": "ZEUS-PHONE", "qty": 5}
		]
	}`

	var payload OrderCreatedPayload
	err := json.Unmarshal([]byte(jsonStr), &payload)
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d", payload.OrderID)
	assert.Equal(t, "9f8e7d6c-5b4a-3f2e-1d0c-9b8a7f6e5d4c", payload.ClientID)
	assert.Equal(t, 999.99, payload.Total)
	assert.Equal(t, "2026-06-05T15:04:05Z", payload.RequiredDate.Format(time.RFC3339))
	if assert.Len(t, payload.Items, 1) {
		assert.Equal(t, "ZEUS-PHONE", payload.Items[0].SKU)
		assert.Equal(t, 5, payload.Items[0].requestedQuantity())
	}
}
