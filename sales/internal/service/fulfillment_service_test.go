package service

import (
	"context"
	"testing"
	"time"

	"zeus-sales-service/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestFulfillmentService_BuildQueue_SortsByTimestamp(t *testing.T) {
	db := setupMockDbRepo()
	cache := setupMockCacheRepo()
	svc := newTestServicesWithMocks(db, cache).Fulfillment

	earliest := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	middle := time.Date(2026, 5, 2, 10, 0, 0, 0, time.UTC)
	latest := time.Date(2026, 5, 3, 10, 0, 0, 0, time.UTC)

	clientB2B := &models.Client{ID: uuid.New(), Name: "B2B Client", Tier: models.ClientTierB2B}
	clientB2C := &models.Client{ID: uuid.New(), Name: "B2C Client", Tier: models.ClientTierB2C}
	order1 := models.SalesOrder{ID: uuid.New(), ClientID: clientB2C.ID, ClientName: clientB2C.Name, RequiredDate: latest, CreatedAt: earliest}
	order2 := models.SalesOrder{ID: uuid.New(), ClientID: clientB2B.ID, ClientName: clientB2B.Name, RequiredDate: latest, CreatedAt: middle}
	order3 := models.SalesOrder{ID: uuid.New(), ClientID: clientB2B.ID, ClientName: clientB2B.Name, RequiredDate: latest, CreatedAt: latest}

	db.On("ListPendingOrders", mock.Anything).Return([]models.SalesOrder{order1, order2, order3}, nil)
	db.On("GetClient", mock.Anything, clientB2B.ID).Return(clientB2B, nil).Maybe()
	db.On("GetClient", mock.Anything, clientB2C.ID).Return(clientB2C, nil).Maybe()
	cache.On("ClearQueue", mock.Anything).Return(nil)
	cache.On("EnqueueOrder", mock.Anything, mock.AnythingOfType("models.AllocationQueueEntry")).Return(nil).Times(3)

	queue, err := svc.BuildQueue(context.Background())
	require.NoError(t, err)
	require.Len(t, queue, 3)
	assert.Equal(t, order1.ID, queue[0].OrderID)
	assert.Equal(t, order2.ID, queue[1].OrderID)
	assert.Equal(t, order3.ID, queue[2].OrderID)
	db.AssertExpectations(t)
	cache.AssertExpectations(t)
}

func TestFulfillmentService_AllocateOrder_ReservesInventoryAndLocksOrders(t *testing.T) {
	db := setupMockDbRepo()
	cache := setupMockCacheRepo()
	svc := newTestServicesWithMocks(db, cache).Fulfillment

	orderID := uuid.New()
	clientID := uuid.New()
	processingStatus := defaultProcessingStatus()
	client := &models.Client{ID: clientID, Name: "Priority Corp", Tier: models.ClientTierB2B}
	order := &models.SalesOrder{
		ID:                 orderID,
		ClientID:           clientID,
		ClientName:         client.Name,
		DestinationAddress: "Dock 7",
		RequiredDate:       time.Now().Add(12 * time.Hour).UTC(),
		StatusID:           defaultPendingStatus().ID,
		Status:             defaultPendingStatus(),
	}
	items := []models.SalesOrderItem{{ID: uuid.New(), OrderID: orderID, SKU: "SKU-A", RequestedQty: 2, UnitPrice: 5}, {ID: uuid.New(), OrderID: orderID, SKU: "SKU-B", RequestedQty: 3, UnitPrice: 7}}

	db.On("GetClient", mock.Anything, clientID).Return(client, nil)
	db.On("GetOrder", mock.Anything, orderID).Return(order, nil)
	db.On("GetOrderItems", mock.Anything, orderID).Return(items, nil)
	db.On("GetOrderStatusByCode", mock.Anything, models.SalesOrderStatusProcessingCode).Return(processingStatus, nil)
	db.On("CreateReservation", mock.Anything, mock.AnythingOfType("*models.InventoryReservation")).Return(nil)
	db.On("UpdateOrder", mock.Anything, mock.MatchedBy(func(updated *models.SalesOrder) bool {
		return updated.ID == orderID && updated.Locked && updated.Status != nil && updated.Status.Code == models.SalesOrderStatusProcessingCode
	})).Return(nil)
	cache.On("GetATP", mock.Anything, "SKU-A").Return(10, nil)
	cache.On("GetATP", mock.Anything, "SKU-B").Return(10, nil)
	cache.On("ReserveInventory", mock.Anything, orderID, mock.AnythingOfType("[]models.ReservationItem")).Return(nil)

	manifest, err := svc.AllocateOrder(context.Background(), orderID)
	require.NoError(t, err)
	require.NotNil(t, manifest)
	assert.Equal(t, "Dock 7", manifest.DestinationAddress)
	db.AssertExpectations(t)
	cache.AssertExpectations(t)
}

func TestFulfillmentService_ProcessQueue_StopsOnInventoryDeficit(t *testing.T) {
	db := setupMockDbRepo()
	cache := setupMockCacheRepo()
	svc := newTestServicesWithMocks(db, cache).Fulfillment

	orderID := uuid.New()
	clientID := uuid.New()
	order := &models.SalesOrder{
		ID:                 orderID,
		ClientID:           clientID,
		ClientName:         "Blocked One",
		DestinationAddress: "Bay 1",
		RequiredDate:       time.Now().Add(1 * time.Hour).UTC(),
		StatusID:           defaultPendingStatus().ID,
		Status:             defaultPendingStatus(),
	}
	entry := models.AllocationQueueEntry{OrderID: orderID, ClientID: clientID, ClientTier: models.ClientTierB2C, RequiredDate: order.RequiredDate, IngestedAt: time.Now().UTC()}
	items := []models.SalesOrderItem{{ID: uuid.New(), OrderID: orderID, SKU: "SKU-X", RequestedQty: 2, UnitPrice: 1}}

	db.On("GetClient", mock.Anything, clientID).Return(&models.Client{ID: clientID, Name: "Blocked One", Tier: models.ClientTierB2C}, nil).Maybe()
	db.On("GetOrder", mock.Anything, orderID).Return(order, nil)
	db.On("GetOrderItems", mock.Anything, orderID).Return(items, nil)
	db.On("GetOrderStatusByCode", mock.Anything, models.SalesOrderStatusProcessingCode).Return(defaultProcessingStatus(), nil)
	cache.On("ListQueue", mock.Anything).Return([]models.AllocationQueueEntry{entry}, nil)
	cache.On("DequeueOrder", mock.Anything).Return(&entry, nil).Once()
	cache.On("GetATP", mock.Anything, "SKU-X").Return(1, nil)
	cache.On("EnqueueOrder", mock.Anything, entry).Return(nil)

	manifests, err := svc.ProcessQueue(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient inventory")
	assert.Len(t, manifests, 0)
	db.AssertExpectations(t)
	cache.AssertExpectations(t)
}
