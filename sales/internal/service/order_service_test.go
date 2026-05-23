package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"zeus-sales-service/internal/models"
	rootrepo "zeus-sales-service/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestOrderService_CreateOrder_ValidatesHardCases(t *testing.T) {
	db := setupMockDbRepo()
	cache := setupMockCacheRepo()
	svc := newTestServicesWithMocks(db, cache).Orders
	baseReq := models.CreateOrderRequest{
		ClientName:         "Acme Manufacturing",
		DestinationAddress: "Dock 9",
		ClientTier:         models.ClientTierB2B,
		RequiredDate:       time.Now().Add(24 * time.Hour).UTC(),
		Items:              []models.OrderItemRequest{{SKU: "SKU-1", RequestedQty: 2, UnitPrice: 10}},
	}

	tests := []struct {
		name    string
		req     models.CreateOrderRequest
		wantErr string
	}{
		{name: "missing client name", req: func() models.CreateOrderRequest { req := baseReq; req.ClientName = ""; return req }(), wantErr: "validation error"},
		{name: "missing required date", req: func() models.CreateOrderRequest { req := baseReq; req.RequiredDate = time.Time{}; return req }(), wantErr: "validation error"},
		{name: "missing items", req: func() models.CreateOrderRequest { req := baseReq; req.Items = nil; return req }(), wantErr: "validation error"},
		{name: "duplicate sku", req: func() models.CreateOrderRequest {
			req := baseReq
			req.Items = []models.OrderItemRequest{{SKU: "SKU-1", RequestedQty: 2, UnitPrice: 10}, {SKU: "sku-1", RequestedQty: 1, UnitPrice: 5}}
			return req
		}(), wantErr: "validation error"},
		{name: "zero quantity", req: func() models.CreateOrderRequest {
			req := baseReq
			req.Items = []models.OrderItemRequest{{SKU: "SKU-2", RequestedQty: 0, UnitPrice: 10}}
			return req
		}(), wantErr: "validation error"},
		{name: "negative price", req: func() models.CreateOrderRequest {
			req := baseReq
			req.Items = []models.OrderItemRequest{{SKU: "SKU-2", RequestedQty: 1, UnitPrice: -1}}
			return req
		}(), wantErr: "validation error"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			response, err := svc.CreateOrder(context.Background(), testCase.req)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), testCase.wantErr)
			assert.Nil(t, response)
		})
	}
}

func TestOrderService_CreateOrder_SetsDerivedFields(t *testing.T) {
	db := setupMockDbRepo()
	cache := setupMockCacheRepo()
	svc := newTestServicesWithMocks(db, cache).Orders
	pendingStatus := defaultPendingStatus()
	clientID := uuid.New()

	db.On("GetClientByName", mock.Anything, "Orbit Co").Return(nil, rootrepo.ErrNotFound)
	db.On("CreateClient", mock.Anything, mock.AnythingOfType("*models.Client")).Return(nil)
	db.On("GetOrderStatusByCode", mock.Anything, models.SalesOrderStatusPendingCode).Return(pendingStatus, nil)
	db.On("CreateOrder", mock.Anything, mock.AnythingOfType("*models.SalesOrder")).Return(nil)
	db.On("CreateOrderItem", mock.Anything, mock.AnythingOfType("*models.SalesOrderItem")).Return(nil).Times(2)
	db.On("UpdateClient", mock.Anything, mock.MatchedBy(func(client *models.Client) bool {
		return client.Name == "Orbit Co" && client.TotalLifetimeOrders == 1
	})).Return(nil)
	cache.On("EnqueueOrder", mock.Anything, mock.AnythingOfType("models.AllocationQueueEntry")).Return(nil)
	db.On("GetClient", mock.Anything, mock.Anything).Return(&models.Client{
		ID:                        clientID,
		Name:                      "Orbit Co",
		Tier:                      models.ClientTierB2C,
		DefaultDestinationAddress: "",
		TotalLifetimeOrders:       1,
	}, nil)

	response, err := svc.CreateOrder(context.Background(), models.CreateOrderRequest{
		ClientName:         "Orbit Co",
		DestinationAddress: "",
		RequiredDate:       time.Now().Add(72 * time.Hour).UTC(),
		Items: []models.OrderItemRequest{
			{SKU: "SKU-100", RequestedQty: 3, UnitPrice: 15},
			{SKU: "SKU-200", RequestedQty: 2, UnitPrice: 8},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, response)
	require.NotNil(t, response.Order.Status)
	assert.Equal(t, models.SalesOrderStatusPendingCode, response.Order.Status.Code)
	assert.Equal(t, models.ClientTierB2C, response.Client.Tier)
	assert.Len(t, response.Items, 2)
	assert.InDelta(t, 61.0, response.Order.TotalValue, 0.0001)
	assert.Equal(t, response.Client.DefaultDestinationAddress, response.Order.DestinationAddress)
	db.AssertExpectations(t)
	cache.AssertExpectations(t)
}

func TestOrderService_UpdateAndCancel_RespectLockAndStatus(t *testing.T) {
	db := setupMockDbRepo()
	cache := setupMockCacheRepo()
	svc := newTestServicesWithMocks(db, cache).Orders
	orderID := uuid.New()
	clientID := uuid.New()
	processingStatus := defaultProcessingStatus()
	lockedOrder := &models.SalesOrder{
		ID:                 orderID,
		ClientID:           clientID,
		ClientName:         "Lockstep Ltd",
		DestinationAddress: "Bay 2",
		RequiredDate:       time.Now().Add(24 * time.Hour).UTC(),
		StatusID:           processingStatus.ID,
		Status:             processingStatus,
		Locked:             true,
	}

	db.On("GetOrder", mock.Anything, orderID).Return(lockedOrder, nil).Twice()
	cache.On("ClearQueue", mock.Anything).Return(nil).Maybe()

	updated, err := svc.UpdateOrder(context.Background(), orderID, models.UpdateOrderRequest{DestinationAddress: ptrString("New Bay")})
	assert.Error(t, err)
	assert.Nil(t, updated)
	assert.Contains(t, err.Error(), "conflict")

	err = svc.CancelOrder(context.Background(), orderID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "conflict")
	db.AssertExpectations(t)
}

func TestOrderService_CancelOrder_MarksPendingOrderCancelled(t *testing.T) {
	db := setupMockDbRepo()
	cache := setupMockCacheRepo()
	svc := newTestServicesWithMocks(db, cache).Orders
	orderID := uuid.New()
	clientID := uuid.New()
	pendingStatus := defaultPendingStatus()
	cancelledStatus := defaultCancelledStatus()
	order := &models.SalesOrder{
		ID:                 orderID,
		ClientID:           clientID,
		ClientName:         "Cancel Me Co",
		DestinationAddress: "Dock 7",
		RequiredDate:       time.Now().Add(48 * time.Hour).UTC(),
		StatusID:           pendingStatus.ID,
		Locked:             false,
	}

	db.On("GetOrder", mock.Anything, orderID).Return(order, nil)
	db.On("GetOrderStatusByID", mock.Anything, pendingStatus.ID).Return(pendingStatus, nil)
	db.On("GetOrderStatusByCode", mock.Anything, models.SalesOrderStatusCancelledCode).Return(cancelledStatus, nil)
	db.On("UpdateOrder", mock.Anything, mock.MatchedBy(func(updated *models.SalesOrder) bool {
		return updated.ID == orderID && updated.StatusID == cancelledStatus.ID && updated.Status != nil && updated.Status.Code == models.SalesOrderStatusCancelledCode
	})).Return(nil)
	cache.On("ClearQueue", mock.Anything).Return(nil)

	err := svc.CancelOrder(context.Background(), orderID)
	require.NoError(t, err)
	db.AssertExpectations(t)
	cache.AssertExpectations(t)
}

func TestOrderService_CancelOrder_IgnoresCacheCleanupFailure(t *testing.T) {
	db := setupMockDbRepo()
	cache := setupMockCacheRepo()
	svc := newTestServicesWithMocks(db, cache).Orders
	orderID := uuid.New()
	clientID := uuid.New()
	pendingStatus := defaultPendingStatus()
	cancelledStatus := defaultCancelledStatus()
	order := &models.SalesOrder{
		ID:                 orderID,
		ClientID:           clientID,
		ClientName:         "Cancel Me Co",
		DestinationAddress: "Dock 7",
		RequiredDate:       time.Now().Add(48 * time.Hour).UTC(),
		StatusID:           pendingStatus.ID,
		Locked:             false,
	}

	db.On("GetOrder", mock.Anything, orderID).Return(order, nil)
	db.On("GetOrderStatusByID", mock.Anything, pendingStatus.ID).Return(pendingStatus, nil)
	db.On("GetOrderStatusByCode", mock.Anything, models.SalesOrderStatusCancelledCode).Return(cancelledStatus, nil)
	db.On("UpdateOrder", mock.Anything, mock.MatchedBy(func(updated *models.SalesOrder) bool {
		return updated.ID == orderID && updated.StatusID == cancelledStatus.ID && updated.Status != nil && updated.Status.Code == models.SalesOrderStatusCancelledCode
	})).Return(nil)
	cache.On("ClearQueue", mock.Anything).Return(fmt.Errorf("redis down"))

	err := svc.CancelOrder(context.Background(), orderID)
	require.NoError(t, err)
	db.AssertExpectations(t)
	cache.AssertExpectations(t)
}

func TestOrderService_ListOrders_ReturnsSummaryRows(t *testing.T) {
	db := setupMockDbRepo()
	cache := setupMockCacheRepo()
	svc := newTestServicesWithMocks(db, cache).Orders
	orderID := uuid.New()
	clientID := uuid.New()
	status := defaultPendingStatus()
	orders := []models.SalesOrder{{
		ID:           orderID,
		ClientID:     clientID,
		ClientName:   "Acme",
		RequiredDate: time.Date(2026, time.January, 2, 10, 0, 0, 0, time.UTC),
		StatusID:     status.ID,
		Status:       status,
		TotalValue:   99.5,
	}}

	db.On("ListOrders", mock.Anything).Return(orders, nil)
	db.On("GetClient", mock.Anything, clientID).Return(&models.Client{ID: clientID, Name: "Acme"}, nil)

	rows, err := svc.ListOrders(context.Background())
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, orderID, rows[0].OrderID)
	assert.Equal(t, "Acme", rows[0].ClientName)
	assert.Equal(t, status.Code, rows[0].Status)
	assert.InDelta(t, 99.5, rows[0].TotalValue, 0.0001)
	db.AssertExpectations(t)
}

func TestOrderService_CreateOrder_AcceptsCamelCaseDatePayload(t *testing.T) {
	db := setupMockDbRepo()
	cache := setupMockCacheRepo()
	svc := newTestServicesWithMocks(db, cache).Orders
	pendingStatus := defaultPendingStatus()

	db.On("GetClientByName", mock.Anything, "Hung").Return(nil, rootrepo.ErrNotFound)
	db.On("CreateClient", mock.Anything, mock.AnythingOfType("*models.Client")).Return(nil)
	db.On("GetOrderStatusByCode", mock.Anything, models.SalesOrderStatusPendingCode).Return(pendingStatus, nil)
	db.On("CreateOrder", mock.Anything, mock.AnythingOfType("*models.SalesOrder")).Return(nil)
	db.On("CreateOrderItem", mock.Anything, mock.AnythingOfType("*models.SalesOrderItem")).Return(nil)
	db.On("UpdateClient", mock.Anything, mock.AnythingOfType("*models.Client")).Return(nil)
	cache.On("EnqueueOrder", mock.Anything, mock.AnythingOfType("models.AllocationQueueEntry")).Return(nil)
	db.On("GetClient", mock.Anything, mock.Anything).Return(&models.Client{ID: uuid.New(), Name: "Hung", Tier: models.ClientTierB2B}, nil)

	response, err := svc.CreateOrder(context.Background(), models.CreateOrderRequest{
		ClientName:         "Hung",
		DestinationAddress: "123 Nguyen Van Troi, TPHCM",
		ClientTier:         models.ClientTierB2B,
		RequiredDate:       time.Date(2026, time.January, 1, 10, 0, 0, 0, time.UTC),
		Items: []models.OrderItemRequest{{
			RequestedQty: 1,
			SKU:          "sadfawefdf",
			UnitPrice:    1,
		}},
	})
	require.NoError(t, err)
	require.NotNil(t, response)
	require.NotNil(t, response.Order.Status)
	assert.Equal(t, models.SalesOrderStatusPendingCode, response.Order.Status.Code)
}

func ptrString(value string) *string {
	return &value
}
