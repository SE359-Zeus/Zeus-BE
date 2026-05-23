package service

import (
	"context"

	"zeus-sales-service/internal/models"
	rootrepo "zeus-sales-service/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type MockDbRepository struct {
	mock.Mock
}

type MockCacheRepository struct {
	mock.Mock
}

func (m *MockDbRepository) CreateClient(ctx context.Context, client *models.Client) error {
	return m.Called(ctx, client).Error(0)
}

func (m *MockDbRepository) GetClient(ctx context.Context, id uuid.UUID) (*models.Client, error) {
	args := m.Called(ctx, id)
	if args.Get(0) != nil {
		return args.Get(0).(*models.Client), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockDbRepository) GetClientByName(ctx context.Context, name string) (*models.Client, error) {
	args := m.Called(ctx, name)
	if args.Get(0) != nil {
		return args.Get(0).(*models.Client), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockDbRepository) ListClients(ctx context.Context) ([]models.Client, error) {
	args := m.Called(ctx)
	if args.Get(0) != nil {
		return args.Get(0).([]models.Client), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockDbRepository) UpdateClient(ctx context.Context, client *models.Client) error {
	return m.Called(ctx, client).Error(0)
}

func (m *MockDbRepository) ListOrderStatuses(ctx context.Context) ([]models.SalesOrderStatusLUT, error) {
	args := m.Called(ctx)
	if args.Get(0) != nil {
		return args.Get(0).([]models.SalesOrderStatusLUT), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockDbRepository) GetOrderStatusByID(ctx context.Context, id uuid.UUID) (*models.SalesOrderStatusLUT, error) {
	args := m.Called(ctx, id)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SalesOrderStatusLUT), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockDbRepository) GetOrderStatusByCode(ctx context.Context, code string) (*models.SalesOrderStatusLUT, error) {
	args := m.Called(ctx, code)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SalesOrderStatusLUT), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockDbRepository) CreateOrder(ctx context.Context, order *models.SalesOrder) error {
	return m.Called(ctx, order).Error(0)
}

func (m *MockDbRepository) GetOrder(ctx context.Context, id uuid.UUID) (*models.SalesOrder, error) {
	args := m.Called(ctx, id)
	if args.Get(0) != nil {
		return args.Get(0).(*models.SalesOrder), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockDbRepository) ListOrders(ctx context.Context) ([]models.SalesOrder, error) {
	args := m.Called(ctx)
	if args.Get(0) != nil {
		return args.Get(0).([]models.SalesOrder), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockDbRepository) ListPendingOrders(ctx context.Context) ([]models.SalesOrder, error) {
	args := m.Called(ctx)
	if args.Get(0) != nil {
		return args.Get(0).([]models.SalesOrder), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockDbRepository) UpdateOrder(ctx context.Context, order *models.SalesOrder) error {
	return m.Called(ctx, order).Error(0)
}

func (m *MockDbRepository) CreateOrderItem(ctx context.Context, item *models.SalesOrderItem) error {
	return m.Called(ctx, item).Error(0)
}

func (m *MockDbRepository) GetOrderItems(ctx context.Context, orderID uuid.UUID) ([]models.SalesOrderItem, error) {
	args := m.Called(ctx, orderID)
	if args.Get(0) != nil {
		return args.Get(0).([]models.SalesOrderItem), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockDbRepository) ReplaceOrderItems(ctx context.Context, orderID uuid.UUID, items []models.SalesOrderItem) error {
	return m.Called(ctx, orderID, items).Error(0)
}

func (m *MockDbRepository) CreateReservation(ctx context.Context, reservation *models.InventoryReservation) error {
	return m.Called(ctx, reservation).Error(0)
}

func (m *MockDbRepository) GetReservation(ctx context.Context, orderID uuid.UUID) (*models.InventoryReservation, error) {
	args := m.Called(ctx, orderID)
	if args.Get(0) != nil {
		return args.Get(0).(*models.InventoryReservation), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockDbRepository) DeleteReservation(ctx context.Context, orderID uuid.UUID) error {
	return m.Called(ctx, orderID).Error(0)
}

func (m *MockCacheRepository) EnqueueOrder(ctx context.Context, entry models.AllocationQueueEntry) error {
	return m.Called(ctx, entry).Error(0)
}

func (m *MockCacheRepository) DequeueOrder(ctx context.Context) (*models.AllocationQueueEntry, error) {
	args := m.Called(ctx)
	if args.Get(0) != nil {
		return args.Get(0).(*models.AllocationQueueEntry), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockCacheRepository) ListQueue(ctx context.Context) ([]models.AllocationQueueEntry, error) {
	args := m.Called(ctx)
	if args.Get(0) != nil {
		return args.Get(0).([]models.AllocationQueueEntry), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockCacheRepository) ClearQueue(ctx context.Context) error {
	return m.Called(ctx).Error(0)
}

func (m *MockCacheRepository) GetATP(ctx context.Context, sku string) (int, error) {
	args := m.Called(ctx, sku)
	return args.Int(0), args.Error(1)
}

func (m *MockCacheRepository) SetATP(ctx context.Context, sku string, quantity int) error {
	return m.Called(ctx, sku, quantity).Error(0)
}

func (m *MockCacheRepository) ReserveInventory(ctx context.Context, orderID uuid.UUID, items []models.ReservationItem) error {
	return m.Called(ctx, orderID, items).Error(0)
}

func (m *MockCacheRepository) ReleaseInventory(ctx context.Context, orderID uuid.UUID) error {
	return m.Called(ctx, orderID).Error(0)
}

func setupMockDbRepo() *MockDbRepository {
	return &MockDbRepository{}
}

func setupMockCacheRepo() *MockCacheRepository {
	return &MockCacheRepository{}
}

func defaultPendingStatus() *models.SalesOrderStatusLUT {
	return &models.SalesOrderStatusLUT{ID: uuid.New(), Code: models.SalesOrderStatusPendingCode, Label: "Pending", SortOrder: 1}
}

func defaultProcessingStatus() *models.SalesOrderStatusLUT {
	return &models.SalesOrderStatusLUT{ID: uuid.New(), Code: models.SalesOrderStatusProcessingCode, Label: "Processing", SortOrder: 2}
}

func defaultCancelledStatus() *models.SalesOrderStatusLUT {
	return &models.SalesOrderStatusLUT{ID: uuid.New(), Code: models.SalesOrderStatusCancelledCode, Label: "Cancelled", SortOrder: 5}
}

var _ rootrepo.DbRepository = (*MockDbRepository)(nil)
var _ rootrepo.CacheRepository = (*MockCacheRepository)(nil)
