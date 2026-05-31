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
	DeleteProductionOrder(ctx context.Context, id uuid.UUID) error

	// BOM & Catalog
	CreateBOMEntries(ctx context.Context, entries []models.BomEntry) error
	DeleteBOMEntriesByModelCode(ctx context.Context, modelCode string) error
	HardDeleteBOMEntriesByModelCode(ctx context.Context, modelCode string) error
	GetBOMByModelCode(ctx context.Context, modelCode string) ([]models.BomEntry, error)
	GetAllBOMs(ctx context.Context) ([]models.BomEntry, error)
	GetPagedBOMsByAssembly(ctx context.Context, page, per int) ([]models.BomEntry, int, error)
	GetWhereUsedByPartID(ctx context.Context, partID uuid.UUID) ([]models.BomEntry, error)

	// Shortages & Demand
	CreateShortageLog(ctx context.Context, log *models.ShortageLog) error
	GetShortagesByOrderID(ctx context.Context, orderID uuid.UUID) ([]models.ShortageLog, error)
	GetShortagesByOrderIDs(ctx context.Context, orderIDs []uuid.UUID) (map[uuid.UUID][]models.ShortageLog, error)
	GetAggregatedShortages(ctx context.Context) ([]models.BOMExplosionResult, error)
	UpdateShortageLog(ctx context.Context, log *models.ShortageLog) error
	DeleteShortageLog(ctx context.Context, orderID uuid.UUID, partID uuid.UUID) error

	// External/Interop (Read-only proxy to Product/Audit services)
	GetInventoryTransactions(ctx context.Context) ([]models.InventoryTransactionDTO, error)
	GetInventoryMetrics(ctx context.Context) (*models.InventoryMetrics, error)
}
