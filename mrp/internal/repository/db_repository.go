package repository

import (
	"context"

	"zeus-mrp-service/internal/models"

	"github.com/google/uuid"
)

type DbRepository interface {
	// Production Orders
	CreateProductionOrder(ctx context.Context, order *models.ProductionOrder) error
	GetProductionOrder(ctx context.Context, id uuid.UUID) (*models.ProductionOrder, error)
	GetOpenProductionOrders(ctx context.Context) ([]models.ProductionOrder, error)
	UpdateProductionOrderStatus(ctx context.Context, id uuid.UUID, status models.ProductionOrderStatus) error

	// BOM & Catalog
	CreateBOMEntries(ctx context.Context, entries []models.BomEntry) error
	DeleteBOMEntriesByModelCode(ctx context.Context, modelCode string) error
	GetBOMByModelCode(ctx context.Context, modelCode string) ([]models.BomEntry, error)
	GetAllBOMs(ctx context.Context) ([]models.BomEntry, error)
	GetWhereUsedByPartID(ctx context.Context, partID uuid.UUID) ([]models.BomEntry, error)

	// Shortages & Demand
	CreateShortageLog(ctx context.Context, log *models.ShortageLog) error
	GetShortagesByOrderID(ctx context.Context, orderID uuid.UUID) ([]models.ShortageLog, error)
	GetAggregatedShortages(ctx context.Context) ([]models.BOMExplosionResult, error)

	// External/Interop (Read-only proxy to Product/Audit services)
	GetPartInventory(ctx context.Context, partID uuid.UUID) (int, error)
	GetInventoryTransactions(ctx context.Context) ([]models.InventoryTransactionDTO, error)
	GetInventoryMetrics(ctx context.Context) (*models.InventoryMetrics, error)
}
