package repository

import (
	"context"

	"zeus-sales-service/internal/models"

	"github.com/google/uuid"
)

type CacheRepository interface {
	EnqueueOrder(ctx context.Context, entry models.AllocationQueueEntry) error
	DequeueOrder(ctx context.Context) (*models.AllocationQueueEntry, error)
	ListQueue(ctx context.Context) ([]models.AllocationQueueEntry, error)
	ClearQueue(ctx context.Context) error

	GetATP(ctx context.Context, sku string) (int, error)
	SetATP(ctx context.Context, sku string, quantity int) error
	ReserveInventory(ctx context.Context, orderID uuid.UUID, items []models.ReservationItem) error
	ReleaseInventory(ctx context.Context, orderID uuid.UUID) error
}
