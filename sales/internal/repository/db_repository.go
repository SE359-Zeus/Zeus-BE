package repository

import (
	"context"

	"zeus-sales-service/internal/models"

	"github.com/google/uuid"
)

type DbRepository interface {
	CreateClient(ctx context.Context, client *models.Client) error
	GetClient(ctx context.Context, id uuid.UUID) (*models.Client, error)
	GetClientByName(ctx context.Context, name string) (*models.Client, error)
	ListClients(ctx context.Context) ([]models.Client, error)
	UpdateClient(ctx context.Context, client *models.Client) error

	ListOrderStatuses(ctx context.Context) ([]models.SalesOrderStatusLUT, error)
	GetOrderStatusByID(ctx context.Context, id uuid.UUID) (*models.SalesOrderStatusLUT, error)
	GetOrderStatusByCode(ctx context.Context, code string) (*models.SalesOrderStatusLUT, error)

	CreateOrder(ctx context.Context, order *models.SalesOrder) error
	GetOrder(ctx context.Context, id uuid.UUID) (*models.SalesOrder, error)
	ListOrders(ctx context.Context) ([]models.SalesOrder, error)
	ListPendingOrders(ctx context.Context) ([]models.SalesOrder, error)
	UpdateOrder(ctx context.Context, order *models.SalesOrder) error

	CreateOrderItem(ctx context.Context, item *models.SalesOrderItem) error
	GetOrderItems(ctx context.Context, orderID uuid.UUID) ([]models.SalesOrderItem, error)
	ReplaceOrderItems(ctx context.Context, orderID uuid.UUID, items []models.SalesOrderItem) error

	CreateReservation(ctx context.Context, reservation *models.InventoryReservation) error
	GetReservation(ctx context.Context, orderID uuid.UUID) (*models.InventoryReservation, error)
	DeleteReservation(ctx context.Context, orderID uuid.UUID) error
}
