package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"zeus-scm-service/internal/infrastructure/cache"
	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/pagination"
	"zeus-scm-service/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IInventoryService interface {
	GetProduct(ctx context.Context, id uuid.UUID) (*models.Product, error)
	ListProducts(ctx context.Context, params pagination.Params, q string) ([]models.Product, *pagination.Meta, error)
	CreateProduct(ctx context.Context, p *models.Product) error
	UpdateProduct(ctx context.Context, id uuid.UUID, fields map[string]any) (*models.Product, error)

	GetProductModel(ctx context.Context, code string) (*models.ProductModel, error)
	CreateProductModel(ctx context.Context, m *models.ProductModel) error

	GetPart(ctx context.Context, id uuid.UUID) (*models.Part, error)
	ListParts(ctx context.Context, catalogID *uuid.UUID, productID *uuid.UUID, conditionID *int32, params pagination.Params, q string) ([]models.Part, *pagination.Meta, error)
	CreatePart(ctx context.Context, p *models.Part) error
	UpdatePart(ctx context.Context, id uuid.UUID, fields map[string]any) (*models.Part, error)
	UpdatePartCondition(ctx context.Context, partID uuid.UUID, conditionID int32) error
	MarkPartScrapped(ctx context.Context, partID uuid.UUID) error
	InstallPart(ctx context.Context, partID uuid.UUID, productID uuid.UUID) error
	RemovePart(ctx context.Context, partID uuid.UUID) error

	GetPartCatalog(ctx context.Context, id uuid.UUID) (*models.PartCatalog, error)
	ListPartCatalog(ctx context.Context, typeID *int32, params pagination.Params, q string) ([]models.PartCatalog, *pagination.Meta, error)
	CreatePartCatalog(ctx context.Context, pc *models.PartCatalog, price float64) error
	UpdatePartCatalogBySKU(ctx context.Context, sku string, fields map[string]any) (*models.PartCatalog, error)
	DeletePartCatalogBySKU(ctx context.Context, sku string) error
	GetPartCatalogBySKU(ctx context.Context, sku string) (*models.PartCatalog, float64, int, error)
	ListStocks(ctx context.Context, params pagination.Params, status, q string) ([]models.ComponentStock, *pagination.Meta, error)
	CreateComponentStock(ctx context.Context, stock *models.ComponentStock) error
	GetStockBySKU(ctx context.Context, sku string) (*models.ComponentStock, error)
}

type inventoryService struct {
	db    *gorm.DB
	cache cache.Cache
}

type inventoryServiceRepo struct {
	repo repository.IInventoryRepository
}

func NewInventoryService(arg interface{}) IInventoryService {
	switch v := arg.(type) {
	case *gorm.DB:
		return &inventoryService{db: v}
	case repository.IInventoryRepository:
		return &inventoryServiceRepo{repo: v}
	default:
		panic("invalid NewInventoryService usage")
	}
}

// NewInventoryServiceWithCache constructs an inventory service backed by a DB and a cache.
func NewInventoryServiceWithCache(db *gorm.DB, c cache.Cache) IInventoryService {
	return &inventoryService{db: db, cache: c}
}

func (s *inventoryService) GetProduct(ctx context.Context, id uuid.UUID) (*models.Product, error) {
	// cache-aside: try cache first
	if s.cache != nil {
		key := fmt.Sprintf("product:%s", id.String())
		if data, err := s.cache.Get(ctx, key); err == nil && data != nil {
			var p models.Product
			if err := json.Unmarshal(data, &p); err == nil {
				return &p, nil
			}
			// fallthrough to DB on unmarshal error
		}
	}

	var p models.Product
	if err := s.db.WithContext(ctx).First(&p, "id = ?", id).Error; err != nil {
		return nil, ErrNotFound
	}

	// populate cache
	if s.cache != nil {
		key := fmt.Sprintf("product:%s", id.String())
		if b, err := json.Marshal(&p); err == nil {
			_ = s.cache.Set(ctx, key, b)
		}
	}
	return &p, nil
}

func (s *inventoryService) ListProducts(ctx context.Context, params pagination.Params, q string) ([]models.Product, *pagination.Meta, error) {
	query := s.db.WithContext(ctx).Model(&models.Product{})
	if q != "" {
		like := "%" + q + "%"
		query = query.Where(
			"product_name LIKE ? OR serial_number LIKE ?",
			like, like,
		)
	}
	var products []models.Product
	meta, err := pagination.Paginate(query, params, &products, "created_at", "updated_at", "product_name", "serial_number")
	if err != nil {
		return nil, nil, err
	}
	return products, meta, nil
}

// repo-backed implementation (used by unit tests with mocks)
func (s *inventoryServiceRepo) GetProduct(ctx context.Context, id uuid.UUID) (*models.Product, error) {
	p, err := s.repo.GetProductByID(ctx, id)
	if err != nil || p == nil {
		return nil, ErrNotFound
	}
	return p, nil
}

func (s *inventoryServiceRepo) ListProducts(ctx context.Context, params pagination.Params, q string) ([]models.Product, *pagination.Meta, error) {
	return s.repo.ListProducts(ctx, params, q)
}

func (s *inventoryServiceRepo) UpdateProduct(ctx context.Context, id uuid.UUID, fields map[string]any) (*models.Product, error) {
	rows, err := s.repo.UpdateProduct(ctx, id, fields)
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		return nil, ErrNotFound
	}
	return s.repo.GetProductByID(ctx, id)
}

func (s *inventoryServiceRepo) CreateProduct(ctx context.Context, p *models.Product) error {
	return s.repo.CreateProduct(ctx, p)
}

func (s *inventoryServiceRepo) GetProductModel(ctx context.Context, code string) (*models.ProductModel, error) {
	m, err := s.repo.GetProductModelByCode(ctx, code)
	if err != nil || m == nil {
		return nil, ErrNotFound
	}
	return m, nil
}

func (s *inventoryServiceRepo) CreateProductModel(ctx context.Context, m *models.ProductModel) error {
	return s.repo.CreateProductModel(ctx, m)
}

func (s *inventoryServiceRepo) GetPart(ctx context.Context, id uuid.UUID) (*models.Part, error) {
	p, err := s.repo.GetPartByID(ctx, id)
	if err != nil || p == nil {
		return nil, ErrNotFound
	}
	return p, nil
}

func (s *inventoryServiceRepo) ListParts(ctx context.Context, catalogID *uuid.UUID, productID *uuid.UUID, conditionID *int32, params pagination.Params, q string) ([]models.Part, *pagination.Meta, error) {
	return s.repo.ListParts(ctx, catalogID, productID, conditionID, params, q)
}

func (s *inventoryServiceRepo) CreatePart(ctx context.Context, p *models.Part) error {
	return s.repo.CreatePart(ctx, p)
}

func (s *inventoryServiceRepo) UpdatePart(ctx context.Context, id uuid.UUID, fields map[string]any) (*models.Part, error) {
	rows, err := s.repo.UpdatePart(ctx, id, fields)
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		return nil, ErrNotFound
	}
	return s.repo.GetPartByID(ctx, id)
}

func (s *inventoryServiceRepo) UpdatePartCondition(ctx context.Context, partID uuid.UUID, conditionID int32) error {
	rows, err := s.repo.UpdatePartFields(ctx, partID, map[string]interface{}{"part_condition_id": conditionID})
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *inventoryServiceRepo) MarkPartScrapped(ctx context.Context, partID uuid.UUID) error {
	rows, err := s.repo.UpdatePartFields(ctx, partID, map[string]interface{}{"scrapped_date": time.Now()})
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *inventoryServiceRepo) InstallPart(ctx context.Context, partID uuid.UUID, productID uuid.UUID) error {
	rows, err := s.repo.UpdatePartFields(ctx, partID, map[string]interface{}{"product_id": productID, "installation_date": time.Now()})
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *inventoryServiceRepo) RemovePart(ctx context.Context, partID uuid.UUID) error {
	rows, err := s.repo.UpdatePartFields(ctx, partID, map[string]interface{}{"product_id": nil, "removal_date": time.Now()})
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *inventoryServiceRepo) GetPartCatalog(ctx context.Context, id uuid.UUID) (*models.PartCatalog, error) {
	c, err := s.repo.GetPartCatalogByID(ctx, id)
	if err != nil || c == nil {
		return nil, ErrNotFound
	}
	return c, nil
}

func (s *inventoryServiceRepo) ListPartCatalog(ctx context.Context, typeID *int32, params pagination.Params, q string) ([]models.PartCatalog, *pagination.Meta, error) {
	return s.repo.ListPartCatalog(ctx, typeID, params, q)
}

func (s *inventoryServiceRepo) CreatePartCatalog(ctx context.Context, pc *models.PartCatalog, price float64) error {
	if err := s.repo.CreatePartCatalog(ctx, pc); err != nil {
		return err
	}
	desc := ""
	if pc.Description != nil {
		desc = *pc.Description
	}
	return s.CreateComponentStock(ctx, &models.ComponentStock{
		SKU:          pc.PartNumber,
		Name:         desc,
		Category:     "Components",
		StockQty:     0,
		ReorderPoint: 0,
		UnitCost:     price,
		Status:       models.ComponentStatusInStock,
		LeadTimeDays: 0,
	})
}

func deriveComponentStatus(stockQty, reorderPoint int) models.ComponentStatus {
	switch {
	case stockQty <= 0:
		return models.ComponentStatusOutOfStock
	case stockQty <= reorderPoint:
		return models.ComponentStatusLowStock
	default:
		return models.ComponentStatusInStock
	}
}

func normalizeComponentStatusFilter(status string) string {
	normalized := strings.ToLower(strings.TrimSpace(status))
	normalized = strings.NewReplacer(" ", "", "_", "", "-", "").Replace(normalized)
	switch normalized {
	case "", "all":
		return ""
	case "instock":
		return string(models.ComponentStatusInStock)
	case "lowstock":
		return string(models.ComponentStatusLowStock)
	case "outofstock":
		return string(models.ComponentStatusOutOfStock)
	case "discontinued":
		return string(models.ComponentStatusDiscontinued)
	default:
		return status
	}
}

func pickRandomIndex(count int) int {
	if count <= 1 {
		return 0
	}
	return rand.New(rand.NewSource(time.Now().UnixNano())).Intn(count)
}

func (s *inventoryServiceRepo) resolvePrimarySupplier(ctx context.Context, sku string) (*models.Supplier, error) {
	mappings, err := s.repo.FindSkuMappingsBySKU(ctx, sku)
	if err != nil {
		return nil, err
	}
	if len(mappings) == 0 {
		return nil, nil
	}
	selected := mappings[pickRandomIndex(len(mappings))]
	supplier, err := s.repo.GetSupplierByID(ctx, selected.SupplierID)
	if err != nil {
		return nil, err
	}
	return supplier, nil
}

func (s *inventoryServiceRepo) hydrateStockSuppliers(ctx context.Context, stocks []models.ComponentStock) ([]models.ComponentStock, error) {
	supplierNames := make(map[uuid.UUID]string)
	for i := range stocks {
		if stocks[i].PrimarySupplierID == uuid.Nil {
			continue
		}
		if name, ok := supplierNames[stocks[i].PrimarySupplierID]; ok {
			stocks[i].PrimarySupplier = name
			continue
		}
		supplier, err := s.repo.GetSupplierByID(ctx, stocks[i].PrimarySupplierID)
		if err != nil {
			return nil, err
		}
		supplierNames[stocks[i].PrimarySupplierID] = supplier.Name
		stocks[i].PrimarySupplier = supplier.Name
	}
	return stocks, nil
}

func (s *inventoryServiceRepo) CreateComponentStock(ctx context.Context, stock *models.ComponentStock) error {
	if stock == nil {
		return fmt.Errorf("component stock is required")
	}
	if stock.Category == "" {
		stock.Category = "Components"
	}
	if stock.Location == "" {
		stock.Location = "WH-A / Zone-C1"
	}
	if stock.Status == "" {
		stock.Status = deriveComponentStatus(stock.StockQty, stock.ReorderPoint)
	}
	if supplier, err := s.resolvePrimarySupplier(ctx, stock.SKU); err == nil && supplier != nil {
		stock.PrimarySupplierID = supplier.ID
		stock.PrimarySupplier = supplier.Name
		if stock.LeadTimeDays == 0 {
			stock.LeadTimeDays = supplier.LeadTimeDays
		}
	}
	return s.repo.CreateComponentStock(ctx, stock)
}

func (s *inventoryServiceRepo) UpdatePartCatalogBySKU(ctx context.Context, sku string, fields map[string]any) (*models.PartCatalog, error) {
	pc, err := s.repo.GetPartCatalogBySKU(ctx, sku)
	if err != nil || pc == nil {
		return nil, ErrNotFound
	}
	updates := make(map[string]interface{})
	if desc, ok := fields["description"]; ok {
		updates["description"] = desc
		if rows, err := s.repo.UpdateComponentStockFieldsBySKU(ctx, sku, map[string]interface{}{"name": desc}); err != nil {
			return nil, err
		} else if rows == 0 {
			return nil, ErrNotFound
		}
	}
	if status, ok := fields["part_mfg_status"]; ok {
		updates["part_mfg_status"] = status
	}
	if len(updates) > 0 {
		if rows, err := s.repo.UpdatePartCatalogFieldsBySKU(ctx, sku, updates); err != nil {
			return nil, err
		} else if rows == 0 {
			return nil, ErrNotFound
		}
	}
	if price, ok := fields["price"].(float64); ok {
		if rows, err := s.repo.UpdateComponentStockFieldsBySKU(ctx, sku, map[string]interface{}{"unit_cost": price}); err != nil {
			return nil, err
		} else if rows == 0 {
			return nil, ErrNotFound
		}
	}
	return pc, nil
}

func (s *inventoryServiceRepo) DeletePartCatalogBySKU(ctx context.Context, sku string) error {
	if rows, err := s.repo.DeleteComponentStockBySKU(ctx, sku); err != nil {
		return err
	} else if rows == 0 {
		return ErrNotFound
	}
	if rows, err := s.repo.DeletePartCatalogBySKU(ctx, sku); err != nil {
		return err
	} else if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *inventoryServiceRepo) GetPartCatalogBySKU(ctx context.Context, sku string) (*models.PartCatalog, float64, int, error) {
	pc, err := s.repo.GetPartCatalogBySKU(ctx, sku)
	if err != nil || pc == nil {
		return nil, 0, 0, ErrNotFound
	}
	stock, err := s.repo.GetComponentStockBySKU(ctx, sku)
	if err != nil || stock == nil {
		return pc, 0, 0, ErrNotFound
	}
	return pc, stock.UnitCost, stock.StockQty, nil
}

func (s *inventoryServiceRepo) ListStocks(ctx context.Context, params pagination.Params, status, q string) ([]models.ComponentStock, *pagination.Meta, error) {
	status = normalizeComponentStatusFilter(status)
	stocks, meta, err := s.repo.ListComponentStocks(ctx, params, status, q)
	if err != nil {
		return nil, nil, err
	}
	hydrated, err := s.hydrateStockSuppliers(ctx, stocks)
	if err != nil {
		return nil, nil, err
	}
	return hydrated, meta, nil
}

func (s *inventoryServiceRepo) GetStockBySKU(ctx context.Context, sku string) (*models.ComponentStock, error) {
	stk, err := s.repo.GetComponentStockBySKU(ctx, sku)
	if err != nil || stk == nil {
		return nil, ErrNotFound
	}
	if stk.PrimarySupplierID != uuid.Nil {
		if supplier, err := s.repo.GetSupplierByID(ctx, stk.PrimarySupplierID); err == nil && supplier != nil {
			stk.PrimarySupplier = supplier.Name
		}
	}
	return stk, nil
}

func (s *inventoryService) UpdateProduct(ctx context.Context, id uuid.UUID, fields map[string]any) (*models.Product, error) {
	result := s.db.WithContext(ctx).Model(&models.Product{}).Where("id = ?", id).Updates(fields)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, ErrNotFound
	}
	var p models.Product
	if err := s.db.WithContext(ctx).First(&p, "id = ?", id).Error; err != nil {
		return nil, ErrNotFound
	}

	// write-through: update cache with new value
	if s.cache != nil {
		key := fmt.Sprintf("product:%s", id.String())
		if b, err := json.Marshal(&p); err == nil {
			_ = s.cache.Set(ctx, key, b)
		}
	}
	return &p, nil
}

func (s *inventoryService) CreateProduct(ctx context.Context, p *models.Product) error {
	if err := s.db.WithContext(ctx).Create(p).Error; err != nil {
		return err
	}
	if s.cache != nil {
		key := fmt.Sprintf("product:%s", p.ID.String())
		if b, err := json.Marshal(p); err == nil {
			_ = s.cache.Set(ctx, key, b)
		}
	}
	return nil
}

func (s *inventoryService) GetProductModel(ctx context.Context, code string) (*models.ProductModel, error) {
	// cache-aside
	if s.cache != nil {
		key := fmt.Sprintf("product_model:%s", code)
		if data, err := s.cache.Get(ctx, key); err == nil && data != nil {
			var m models.ProductModel
			if err := json.Unmarshal(data, &m); err == nil {
				return &m, nil
			}
		}
	}

	var m models.ProductModel
	if err := s.db.WithContext(ctx).First(&m, "model_code = ?", code).Error; err != nil {
		return nil, ErrNotFound
	}

	// Calculate unit price dynamically based on BOM components
	var boms []models.PartsByModel
	if err := s.db.WithContext(ctx).Where("product_model_code = ?", code).Find(&boms).Error; err == nil {
		var totalPrice float64
		for _, bom := range boms {
			var pc models.PartCatalog
			if err := s.db.WithContext(ctx).First(&pc, "id = ?", bom.PartCatalogID).Error; err == nil {
				var stock models.ComponentStock
				if err := s.db.WithContext(ctx).First(&stock, "sku = ?", pc.PartNumber).Error; err == nil {
					totalPrice += float64(bom.Quantity) * stock.UnitCost
				}
			}
		}
		m.UnitPrice = totalPrice
	}

	if s.cache != nil {
		key := fmt.Sprintf("product_model:%s", code)
		if b, err := json.Marshal(&m); err == nil {
			_ = s.cache.Set(ctx, key, b)
		}
	}
	return &m, nil
}

func (s *inventoryService) CreateProductModel(ctx context.Context, m *models.ProductModel) error {
	if err := s.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	if s.cache != nil {
		key := fmt.Sprintf("product_model:%s", m.ModelCode)
		if b, err := json.Marshal(m); err == nil {
			_ = s.cache.Set(ctx, key, b)
		}
	}
	return nil
}

func (s *inventoryService) GetPart(ctx context.Context, id uuid.UUID) (*models.Part, error) {
	if s.cache != nil {
		key := fmt.Sprintf("part:%s", id.String())
		if data, err := s.cache.Get(ctx, key); err == nil && data != nil {
			var p models.Part
			if err := json.Unmarshal(data, &p); err == nil {
				return &p, nil
			}
		}
	}

	var p models.Part
	if err := s.db.WithContext(ctx).First(&p, "id = ?", id).Error; err != nil {
		return nil, ErrNotFound
	}
	if s.cache != nil {
		key := fmt.Sprintf("part:%s", id.String())
		if b, err := json.Marshal(&p); err == nil {
			_ = s.cache.Set(ctx, key, b)
		}
	}
	return &p, nil
}

func (s *inventoryService) ListParts(ctx context.Context, catalogID *uuid.UUID, productID *uuid.UUID, conditionID *int32, params pagination.Params, q string) ([]models.Part, *pagination.Meta, error) {
	query := s.db.WithContext(ctx).Model(&models.Part{})
	if catalogID != nil {
		query = query.Where("part_catalog_id = ?", *catalogID)
	}
	if productID != nil {
		query = query.Where("product_id = ?", *productID)
	}
	if conditionID != nil {
		query = query.Where("part_condition_id = ?", *conditionID)
	}
	if q != "" {
		like := "%" + q + "%"
		query = query.Where("serial_number LIKE ?", like)
	}
	var parts []models.Part
	meta, err := pagination.Paginate(query, params, &parts, "created_at", "updated_at", "serial_number", "part_condition_id")
	if err != nil {
		return nil, nil, err
	}
	return parts, meta, nil
}

func (s *inventoryService) CreatePart(ctx context.Context, p *models.Part) error {
	if err := s.db.WithContext(ctx).Create(p).Error; err != nil {
		return err
	}
	if s.cache != nil {
		key := fmt.Sprintf("part:%s", p.ID.String())
		if b, err := json.Marshal(p); err == nil {
			_ = s.cache.Set(ctx, key, b)
		}
	}
	return nil
}

func (s *inventoryService) UpdatePart(ctx context.Context, id uuid.UUID, fields map[string]any) (*models.Part, error) {
	result := s.db.WithContext(ctx).Model(&models.Part{}).Where("id = ?", id).Updates(fields)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, ErrNotFound
	}
	var p models.Part
	if err := s.db.WithContext(ctx).First(&p, "id = ?", id).Error; err != nil {
		return nil, ErrNotFound
	}

	if s.cache != nil {
		key := fmt.Sprintf("part:%s", id.String())
		if b, err := json.Marshal(&p); err == nil {
			_ = s.cache.Set(ctx, key, b)
		}
	}
	return &p, nil
}

func (s *inventoryService) UpdatePartCondition(ctx context.Context, partID uuid.UUID, conditionID int32) error {
	result := s.db.WithContext(ctx).Model(&models.Part{}).
		Where("id = ?", partID).
		Update("part_condition_id", conditionID)
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	if s.cache != nil {
		// refresh cached part
		var p models.Part
		if err := s.db.WithContext(ctx).First(&p, "id = ?", partID).Error; err == nil {
			key := fmt.Sprintf("part:%s", partID.String())
			if b, err := json.Marshal(&p); err == nil {
				_ = s.cache.Set(ctx, key, b)
			}
		}
	}
	return result.Error
}

func (s *inventoryService) MarkPartScrapped(ctx context.Context, partID uuid.UUID) error {
	now := time.Now()
	result := s.db.WithContext(ctx).Model(&models.Part{}).
		Where("id = ?", partID).
		Update("scrapped_date", now)
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	if s.cache != nil {
		var p models.Part
		if err := s.db.WithContext(ctx).First(&p, "id = ?", partID).Error; err == nil {
			key := fmt.Sprintf("part:%s", partID.String())
			if b, err := json.Marshal(&p); err == nil {
				_ = s.cache.Set(ctx, key, b)
			}
		}
	}
	return result.Error
}

func (s *inventoryService) InstallPart(ctx context.Context, partID uuid.UUID, productID uuid.UUID) error {
	now := time.Now()
	result := s.db.WithContext(ctx).Model(&models.Part{}).
		Where("id = ?", partID).
		Updates(map[string]interface{}{
			"product_id":        productID,
			"installation_date": now,
		})
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	if s.cache != nil {
		var p models.Part
		if err := s.db.WithContext(ctx).First(&p, "id = ?", partID).Error; err == nil {
			key := fmt.Sprintf("part:%s", partID.String())
			if b, err := json.Marshal(&p); err == nil {
				_ = s.cache.Set(ctx, key, b)
			}
		}
	}
	return result.Error
}

func (s *inventoryService) RemovePart(ctx context.Context, partID uuid.UUID) error {
	now := time.Now()
	result := s.db.WithContext(ctx).Model(&models.Part{}).
		Where("id = ?", partID).
		Updates(map[string]interface{}{
			"product_id":   nil,
			"removal_date": now,
		})
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	if s.cache != nil {
		var p models.Part
		if err := s.db.WithContext(ctx).First(&p, "id = ?", partID).Error; err == nil {
			key := fmt.Sprintf("part:%s", partID.String())
			if b, err := json.Marshal(&p); err == nil {
				_ = s.cache.Set(ctx, key, b)
			}
		}
	}
	return result.Error
}

func (s *inventoryService) CreatePartCatalog(ctx context.Context, pc *models.PartCatalog, price float64) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(pc).Error; err != nil {
			return err
		}
		desc := ""
		if pc.Description != nil {
			desc = *pc.Description
		}
		stk := models.ComponentStock{
			SKU:          pc.PartNumber,
			Name:         desc,
			Category:     "Components",
			StockQty:     0,
			ReorderPoint: 0,
			UnitCost:     price,
			Status:       models.ComponentStatusInStock,
			LeadTimeDays: 0,
		}
		return tx.Create(&stk).Error
	})
}

func (s *inventoryService) DeletePartCatalogBySKU(ctx context.Context, sku string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("sku = ?", sku).Delete(&models.ComponentStock{}).Error; err != nil {
			return err
		}
		if err := tx.Where("part_number = ?", sku).Delete(&models.PartCatalog{}).Error; err != nil {
			return err
		}
		return nil
	})
}

func (s *inventoryService) GetPartCatalogBySKU(ctx context.Context, sku string) (*models.PartCatalog, float64, int, error) {
	var pc models.PartCatalog
	if err := s.db.WithContext(ctx).First(&pc, "part_number = ?", sku).Error; err != nil {
		return nil, 0, 0, ErrNotFound
	}
	var stock models.ComponentStock
	if err := s.db.WithContext(ctx).First(&stock, "sku = ?", sku).Error; err != nil {
		return &pc, 0, 0, ErrNotFound
	}
	return &pc, stock.UnitCost, stock.StockQty, nil
}

func (s *inventoryService) UpdatePartCatalogBySKU(ctx context.Context, sku string, fields map[string]any) (*models.PartCatalog, error) {
	var pc models.PartCatalog
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&pc, "part_number = ?", sku).Error; err != nil {
			return err
		}
		updates := make(map[string]any)
		if desc, ok := fields["description"]; ok {
			updates["description"] = desc
			if err := tx.Model(&models.ComponentStock{}).Where("sku = ?", sku).Update("name", desc).Error; err != nil {
				return err
			}
		}
		if status, ok := fields["part_mfg_status"]; ok {
			updates["part_mfg_status"] = status
		}
		if len(updates) > 0 {
			if err := tx.Model(&pc).Updates(updates).Error; err != nil {
				return err
			}
		}
		if price, ok := fields["price"].(float64); ok {
			if err := tx.Model(&models.ComponentStock{}).Where("sku = ?", sku).Update("unit_cost", price).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// update cache for part catalog if present
	if s.cache != nil {
		key := fmt.Sprintf("part_catalog:sku:%s", sku)
		if b, err := json.Marshal(&pc); err == nil {
			_ = s.cache.Set(ctx, key, b)
		}
	}

	return &pc, nil
}

func (s *inventoryService) GetPartCatalog(ctx context.Context, id uuid.UUID) (*models.PartCatalog, error) {
	// cache-aside
	if s.cache != nil {
		key := fmt.Sprintf("part_catalog:%s", id.String())
		if data, err := s.cache.Get(ctx, key); err == nil && data != nil {
			var pc models.PartCatalog
			if err := json.Unmarshal(data, &pc); err == nil {
				return &pc, nil
			}
		}
	}

	var pc models.PartCatalog
	if err := s.db.WithContext(ctx).First(&pc, "id = ?", id).Error; err != nil {
		return nil, ErrNotFound
	}
	if s.cache != nil {
		key := fmt.Sprintf("part_catalog:%s", id.String())
		if b, err := json.Marshal(&pc); err == nil {
			_ = s.cache.Set(ctx, key, b)
		}
	}
	return &pc, nil
}

func (s *inventoryService) ListPartCatalog(ctx context.Context, typeID *int32, params pagination.Params, q string) ([]models.PartCatalog, *pagination.Meta, error) {
	query := s.db.WithContext(ctx).Model(&models.PartCatalog{})
	if typeID != nil {
		query = query.Where("part_types_id = ?", *typeID)
	}
	if q != "" {
		like := "%" + q + "%"
		query = query.Where(
			"part_number LIKE ? OR mfg_number LIKE ? OR description LIKE ?",
			like, like, like,
		)
	}
	var catalogs []models.PartCatalog
	meta, err := pagination.Paginate(query, params, &catalogs, "created_at", "updated_at", "part_number", "mfg_number")
	if err != nil {
		return nil, nil, err
	}
	return catalogs, meta, nil
}

func (s *inventoryService) resolvePrimarySupplier(ctx context.Context, sku string) (*models.Supplier, error) {
	var mappings []models.SkuMapping
	if err := s.db.WithContext(ctx).Where("sku = ?", sku).Order("unit_price ASC").Find(&mappings).Error; err != nil {
		return nil, err
	}
	if len(mappings) == 0 {
		return nil, nil
	}
	selected := mappings[pickRandomIndex(len(mappings))]
	var supplier models.Supplier
	if err := s.db.WithContext(ctx).First(&supplier, "id = ?", selected.SupplierID).Error; err != nil {
		return nil, err
	}
	return &supplier, nil
}

func (s *inventoryService) hydrateStockSuppliers(ctx context.Context, stocks []models.ComponentStock) ([]models.ComponentStock, error) {
	supplierNames := make(map[uuid.UUID]string)
	for i := range stocks {
		if stocks[i].PrimarySupplierID == uuid.Nil {
			continue
		}
		if name, ok := supplierNames[stocks[i].PrimarySupplierID]; ok {
			stocks[i].PrimarySupplier = name
			continue
		}
		var supplier models.Supplier
		if err := s.db.WithContext(ctx).First(&supplier, "id = ?", stocks[i].PrimarySupplierID).Error; err != nil {
			continue
		}
		supplierNames[stocks[i].PrimarySupplierID] = supplier.Name
		stocks[i].PrimarySupplier = supplier.Name
	}
	return stocks, nil
}

func (s *inventoryService) CreateComponentStock(ctx context.Context, stock *models.ComponentStock) error {
	if stock == nil {
		return fmt.Errorf("component stock is required")
	}
	if stock.Category == "" {
		stock.Category = "Components"
	}
	if stock.Location == "" {
		stock.Location = "WH-A / Zone-C1"
	}
	if stock.Status == "" {
		stock.Status = deriveComponentStatus(stock.StockQty, stock.ReorderPoint)
	}
	if stock.PrimarySupplierID == uuid.Nil {
		if supplier, err := s.resolvePrimarySupplier(ctx, stock.SKU); err == nil && supplier != nil {
			stock.PrimarySupplierID = supplier.ID
			stock.PrimarySupplier = supplier.Name
			if stock.LeadTimeDays == 0 {
				stock.LeadTimeDays = supplier.LeadTimeDays
			}
		}
	}
	return s.db.WithContext(ctx).Create(stock).Error
}

func (s *inventoryService) ListStocks(ctx context.Context, params pagination.Params, status, q string) ([]models.ComponentStock, *pagination.Meta, error) {
	status = normalizeComponentStatusFilter(status)
	query := s.db.WithContext(ctx).Model(&models.ComponentStock{})
	if status != "" && !strings.EqualFold(status, "all") {
		query = query.Where("status = ?", status)
	}
	if q != "" {
		like := "%" + q + "%"
		query = query.Where("sku LIKE ? OR name LIKE ? OR category LIKE ? OR location LIKE ?", like, like, like, like)
	}
	var stocks []models.ComponentStock
	meta, err := pagination.Paginate(query, params, &stocks, "created_at", "updated_at", "sku", "name", "category")
	if err != nil {
		return nil, nil, err
	}
	hydrated, err := s.hydrateStockSuppliers(ctx, stocks)
	if err != nil {
		return nil, nil, err
	}
	return hydrated, meta, nil
}

func (s *inventoryService) GetStockBySKU(ctx context.Context, sku string) (*models.ComponentStock, error) {
	var stock models.ComponentStock
	if err := s.db.WithContext(ctx).First(&stock, "sku = ?", sku).Error; err != nil {
		return nil, ErrNotFound
	}
	if stock.PrimarySupplierID != uuid.Nil {
		var supplier models.Supplier
		if err := s.db.WithContext(ctx).First(&supplier, "id = ?", stock.PrimarySupplierID).Error; err == nil {
			stock.PrimarySupplier = supplier.Name
		}
	}
	return &stock, nil
}
