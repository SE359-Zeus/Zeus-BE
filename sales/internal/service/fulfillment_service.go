package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"zeus-sales-service/internal/middlewares"
	"zeus-sales-service/internal/models"
	"zeus-sales-service/internal/repository"

	"github.com/google/uuid"
)

type FulfillmentService struct {
	sqlite repository.DbRepository
	valkey repository.CacheRepository
}

func NewFulfillmentService(sqliteRepo repository.DbRepository, valkeyRepo repository.CacheRepository) *FulfillmentService {
	return &FulfillmentService{sqlite: sqliteRepo, valkey: valkeyRepo}
}

func (svc *FulfillmentService) BuildQueue(ctx context.Context) ([]models.AllocationQueueEntry, error) {
	return svc.rebuildQueue(ctx)
}

func (svc *FulfillmentService) rebuildQueue(ctx context.Context) ([]models.AllocationQueueEntry, error) {
	orders, err := svc.sqlite.ListPendingOrders(ctx)
	if err != nil {
		return nil, err
	}
	entries := make([]models.AllocationQueueEntry, 0, len(orders))
	for _, order := range orders {
		client, err := svc.sqlite.GetClient(ctx, order.ClientID)
		if err != nil {
			return nil, err
		}
		entries = append(entries, models.AllocationQueueEntry{
			OrderID:      order.ID,
			ClientID:     client.ID,
			ClientTier:   client.Tier,
			RequiredDate: order.RequiredDate,
			IngestedAt:   order.CreatedAt,
		})
	}
	if err := svc.valkey.ClearQueue(ctx); err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if err := svc.valkey.EnqueueOrder(ctx, entry); err != nil {
			return nil, err
		}
	}
	return entries, nil
}

func (svc *FulfillmentService) GetQueueStatus(ctx context.Context) (*models.QueueStatus, error) {
	entries, err := svc.valkey.ListQueue(ctx)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		entries, err = svc.rebuildQueue(ctx)
		if err != nil {
			return nil, err
		}
	}
	return &models.QueueStatus{Entries: entries}, nil
}

func (svc *FulfillmentService) ProcessQueue(ctx context.Context) ([]models.FulfillmentManifest, error) {
	entries, err := svc.valkey.ListQueue(ctx)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		entries, err = svc.rebuildQueue(ctx)
		if err != nil {
			return nil, err
		}
	}
	manifests := make([]models.FulfillmentManifest, 0, len(entries))
	for {
		entry, err := svc.valkey.DequeueOrder(ctx)
		if err != nil {
			return manifests, err
		}
		if entry == nil {
			break
		}
		manifest, err := svc.allocateOrder(ctx, entry.OrderID)
		if err != nil {
			if errors.Is(err, repository.ErrInsufficientInventory) {
				_ = svc.valkey.EnqueueOrder(ctx, *entry)
				return manifests, err
			}
			return manifests, err
		}
		manifests = append(manifests, *manifest)
	}
	return manifests, nil
}

func (svc *FulfillmentService) AllocateOrder(ctx context.Context, orderID uuid.UUID) (*models.FulfillmentManifest, error) {
	return svc.allocateOrder(ctx, orderID)
}

func (svc *FulfillmentService) allocateOrder(ctx context.Context, orderID uuid.UUID) (*models.FulfillmentManifest, error) {
	if orderID == uuid.Nil {
		return nil, fmt.Errorf("%w: order id is required", middlewares.ErrValidation)
	}
	order, err := svc.sqlite.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	items, err := svc.sqlite.GetOrderItems(ctx, orderID)
	if err != nil {
		return nil, err
	}
	processingStatus, err := svc.sqlite.GetOrderStatusByCode(ctx, models.SalesOrderStatusProcessingCode)
	if err != nil {
		return nil, err
	}
	reservationItems := make([]models.ReservationItem, 0, len(items))
	for _, item := range items {
		atp, err := svc.valkey.GetATP(ctx, item.SKU)
		if err != nil {
			return nil, err
		}
		if atp < item.RequestedQty {
			return nil, repository.ErrInsufficientInventory
		}
		reservationItems = append(reservationItems, models.ReservationItem{SKU: item.SKU, Quantity: item.RequestedQty})
	}
	if err := svc.valkey.ReserveInventory(ctx, orderID, reservationItems); err != nil {
		return nil, err
	}
	reservation := &models.InventoryReservation{
		ID:         uuid.New(),
		OrderID:    orderID,
		Items:      reservationItems,
		ReservedAt: time.Now().UTC(),
	}
	if err := svc.sqlite.CreateReservation(ctx, reservation); err != nil {
		_ = svc.valkey.ReleaseInventory(ctx, orderID)
		return nil, err
	}
	order.StatusID = processingStatus.ID
	order.Status = processingStatus
	order.Locked = true
	if err := svc.sqlite.UpdateOrder(ctx, order); err != nil {
		_ = svc.valkey.ReleaseInventory(ctx, orderID)
		_ = svc.sqlite.DeleteReservation(ctx, orderID)
		return nil, err
	}
	client, err := svc.sqlite.GetClient(ctx, order.ClientID)
	if err != nil {
		return nil, err
	}
	return &models.FulfillmentManifest{
		OrderID:            order.ID,
		ClientID:           client.ID,
		DestinationAddress: order.DestinationAddress,
		Items:              reservationItems,
		GeneratedAt:        time.Now().UTC(),
	}, nil
}
