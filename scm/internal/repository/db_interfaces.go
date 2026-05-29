package repository

import (
	"context"

	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/pagination"

	"github.com/google/uuid"
)

type IVendorRepository interface {
	GetSupplierByID(ctx context.Context, id uuid.UUID) (*models.Supplier, error)
	FindSkuMappingsBySKU(ctx context.Context, sku string) ([]models.SkuMapping, error)
	FindSkuMappingsBySupplierID(ctx context.Context, supplierID uuid.UUID) ([]models.SkuMapping, error)
	UpdateSupplier(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error
	FindGoodsReceiptsByVendor(ctx context.Context, vendorID uuid.UUID) ([]models.GoodsReceipt, error)
	CountGoodsReceiptsByVendor(ctx context.Context, vendorID uuid.UUID) (int64, error)
	FindGRLineItemsByGRID(ctx context.Context, grID string) ([]models.GRLineItem, error)
}

type IPORepository interface {
	GetPOByID(ctx context.Context, id string) (*models.PurchaseOrder, error)
	CreatePO(ctx context.Context, po *models.PurchaseOrder) error
	SavePO(ctx context.Context, po *models.PurchaseOrder) error
	UpdatePOStatus(ctx context.Context, id string, status models.POStatus) error
	CreatePOLineItem(ctx context.Context, item *models.POLineItem) error
	GetPOLineItemsByPOID(ctx context.Context, poID string) ([]models.POLineItem, error)
	CountPOsByYearPattern(ctx context.Context, year int, pattern string) (int64, error)
	FindPOByVendorAndStatuses(ctx context.Context, vendorID uuid.UUID, statuses []models.POStatus) (*models.PurchaseOrder, error)
}

type IInventoryRepository interface {
	GetProductByID(ctx context.Context, id uuid.UUID) (*models.Product, error)
	ListProducts(ctx context.Context, params pagination.Params, q string) ([]models.Product, *pagination.Meta, error)
	CreateProduct(ctx context.Context, p *models.Product) error
	UpdateProduct(ctx context.Context, id uuid.UUID, fields map[string]any) (int64, error)

	GetProductModelByCode(ctx context.Context, code string) (*models.ProductModel, error)
	CreateProductModel(ctx context.Context, m *models.ProductModel) error

	GetPartByID(ctx context.Context, id uuid.UUID) (*models.Part, error)
	ListParts(ctx context.Context, catalogID *uuid.UUID, productID *uuid.UUID, conditionID *int32, params pagination.Params, q string) ([]models.Part, *pagination.Meta, error)
	CreatePart(ctx context.Context, p *models.Part) error
	UpdatePart(ctx context.Context, id uuid.UUID, fields map[string]any) (int64, error)
	UpdatePartFields(ctx context.Context, id uuid.UUID, updates map[string]interface{}) (int64, error)

	GetPartCatalogByID(ctx context.Context, id uuid.UUID) (*models.PartCatalog, error)
	ListPartCatalog(ctx context.Context, typeID *int32, params pagination.Params, q string) ([]models.PartCatalog, *pagination.Meta, error)
	CreatePartCatalog(ctx context.Context, pc *models.PartCatalog) error
	UpdatePartCatalogFieldsBySKU(ctx context.Context, sku string, updates map[string]interface{}) (int64, error)
	DeletePartCatalogBySKU(ctx context.Context, sku string) (int64, error)
	GetPartCatalogBySKU(ctx context.Context, sku string) (*models.PartCatalog, error)
	GetComponentStockBySKU(ctx context.Context, sku string) (*models.ComponentStock, error)
	ListComponentStocks(ctx context.Context, params pagination.Params, q string) ([]models.ComponentStock, *pagination.Meta, error)
	CreateComponentStock(ctx context.Context, stock *models.ComponentStock) error
	UpdateComponentStockFieldsBySKU(ctx context.Context, sku string, updates map[string]interface{}) (int64, error)
	DeleteComponentStockBySKU(ctx context.Context, sku string) (int64, error)
}

type IShipmentRepository interface {
	GetShipmentByID(ctx context.Context, id string) (*models.Shipment, error)
	UpdateShipment(ctx context.Context, shipment *models.Shipment) error
	UpdateShipmentFields(ctx context.Context, id string, fields map[string]interface{}) error
	GetShipmentItemsByShipmentID(ctx context.Context, shipmentID string) ([]models.ShipmentItem, error)
}

type IStockRepository interface {
	GetStockBySKU(ctx context.Context, sku string) (*models.ComponentStock, error)
	SaveStock(ctx context.Context, stock *models.ComponentStock) error
	UpsertStock(ctx context.Context, stock *models.ComponentStock) error
}

type IGoodsReceiptRepository interface {
	GetGRByID(ctx context.Context, id string) (*models.GoodsReceipt, error)
	UpdateGR(ctx context.Context, gr *models.GoodsReceipt) error
	UpdateGRFields(ctx context.Context, id string, fields map[string]interface{}) error
	FindGRLineItemsByGRID(ctx context.Context, grID string) ([]models.GRLineItem, error)
	SaveGRLineItem(ctx context.Context, item *models.GRLineItem) error
}
