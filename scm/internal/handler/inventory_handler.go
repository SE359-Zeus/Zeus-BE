package handler

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"time"

	"zeus-scm-service/internal/exception"
	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/pagination"
	"zeus-scm-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type InventoryHandler struct {
	svc service.IInventoryService
}

func NewInventoryHandler(svc service.IInventoryService) *InventoryHandler {
	return &InventoryHandler{svc: svc}
}

func (h *InventoryHandler) GetProduct(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		exception.WriteError(c, exception.ErrInvalidResourceID)
		return
	}
	p, err := h.svc.GetProduct(c.Request.Context(), id)
	if err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal)
		return
	}
	writeJSON(c, 200, p)
}

func (h *InventoryHandler) GetProductBySerialNumber(c *gin.Context) {
	serialNumber := c.Param("serialNumber")
	if serialNumber == "" {
		exception.WriteError(c, exception.ErrInvalidInput)
		return
	}
	p, err := h.svc.GetProductBySerialNumber(c.Request.Context(), serialNumber)
	if err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal)
		return
	}
	writeJSON(c, 200, p)
}

func parsePaginationParams(c *gin.Context) pagination.Params {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "15"))
	return pagination.Params{
		Page:  page,
		Limit: limit,
		Sort:  c.DefaultQuery("sort_by", "created_at"),
		Order: c.DefaultQuery("sort_dir", "desc"),
	}
}

func (h *InventoryHandler) ListProducts(c *gin.Context) {
	params := parsePaginationParams(c)
	q := c.Query("q")

	var customerID *uuid.UUID
	if v := c.Query("customer_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			customerID = &id
		}
	}

	products, meta, err := h.svc.ListProducts(c.Request.Context(), params, q, customerID)
	if err != nil {
		exception.WriteError(c, exception.ErrInternal.WithError(err))
		return
	}
	writeJSON(c, 200, pagination.Response{Data: products, Pagination: *meta})
}

func (h *InventoryHandler) CreateProduct(c *gin.Context) {
	var p models.Product
	if err := c.ShouldBindJSON(&p); err != nil {
		exception.WriteError(c, exception.ErrInvalidBody)
		return
	}
	if err := h.svc.CreateProduct(c.Request.Context(), &p); err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal)
		return
	}
	writeJSON(c, 201, p)
}

func (h *InventoryHandler) GetProductModel(c *gin.Context) {
	code := c.Param("code")
	m, err := h.svc.GetProductModel(c.Request.Context(), code)
	if err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal)
		return
	}
	writeJSON(c, 200, m)
}

func (h *InventoryHandler) ListProductModels(c *gin.Context) {
	params := parsePaginationParams(c)
	q := c.Query("q")

	models, meta, err := h.svc.ListProductModels(c.Request.Context(), params, q)
	if err != nil {
		exception.WriteError(c, exception.ErrInternal.WithError(err))
		return
	}
	writeJSON(c, 200, pagination.Response{Data: models, Pagination: *meta})
}

func (h *InventoryHandler) CreateProductModel(c *gin.Context) {
	var m models.ProductModel
	if err := c.ShouldBindJSON(&m); err != nil {
		exception.WriteError(c, exception.ErrInvalidBody)
		return
	}
	if err := h.svc.CreateProductModel(c.Request.Context(), &m); err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal)
		return
	}
	writeJSON(c, 201, m)
}

func (h *InventoryHandler) DeleteProductModel(c *gin.Context) {
	code := c.Param("code")
	if err := h.svc.DeleteProductModel(c.Request.Context(), code); err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal)
		return
	}
	writeJSON(c, 204, nil)
}

func (h *InventoryHandler) GetPart(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		exception.WriteError(c, exception.ErrInvalidResourceID)
		return
	}
	p, err := h.svc.GetPart(c.Request.Context(), id)
	if err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal)
		return
	}
	writeJSON(c, 200, p)
}

func (h *InventoryHandler) ListParts(c *gin.Context) {
	var catalogID, productID *uuid.UUID
	var conditionID *int32
	if v := c.Query("catalog_id"); v != "" {
		id, err := uuid.Parse(v)
		if err == nil {
			catalogID = &id
		}
	}
	if v := c.Query("product_id"); v != "" {
		id, err := uuid.Parse(v)
		if err == nil {
			productID = &id
		}
	}
	if v := c.Query("condition_id"); v != "" {
		if parsed, err := parseInt32(v); err == nil {
			conditionID = &parsed
		}
	}

	params := parsePaginationParams(c)
	q := c.Query("q")

	parts, meta, err := h.svc.ListParts(c.Request.Context(), catalogID, productID, conditionID, params, q)
	if err != nil {
		exception.WriteError(c, exception.ErrInternal.WithError(err))
		return
	}
	writeJSON(c, 200, pagination.Response{Data: parts, Pagination: *meta})
}

func (h *InventoryHandler) CreatePart(c *gin.Context) {
	var p models.Part
	if err := c.ShouldBindJSON(&p); err != nil {
		exception.WriteError(c, exception.ErrInvalidBody)
		return
	}
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	now := time.Now()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	if p.UpdatedAt.IsZero() {
		p.UpdatedAt = now
	}
	if err := h.svc.CreatePart(c.Request.Context(), &p); err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal)
		return
	}
	writeJSON(c, 201, p)
}

func (h *InventoryHandler) UpdatePartCondition(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		exception.WriteError(c, exception.ErrInvalidResourceID)
		return
	}
	var req struct {
		ConditionID int32 `json:"condition_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		exception.WriteError(c, exception.ErrInvalidBody)
		return
	}
	if err := h.svc.UpdatePartCondition(c.Request.Context(), id, req.ConditionID); err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal)
		return
	}
	writeJSON(c, 200, gin.H{"message": "condition updated"})
}

func (h *InventoryHandler) MarkPartScrapped(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		exception.WriteError(c, exception.ErrInvalidResourceID)
		return
	}
	if err := h.svc.MarkPartScrapped(c.Request.Context(), id); err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal)
		return
	}
	writeJSON(c, 200, gin.H{"message": "part scrapped"})
}

func (h *InventoryHandler) InstallPart(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		exception.WriteError(c, exception.ErrInvalidResourceID)
		return
	}
	var req struct {
		ProductID string `json:"product_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		exception.WriteError(c, exception.ErrInvalidBody)
		return
	}
	productID, err := uuid.Parse(req.ProductID)
	if err != nil {
		exception.WriteError(c, exception.ErrInvalidResourceID.WithMessage("The product_id provided is not a valid UUID"))
		return
	}
	if err := h.svc.InstallPart(c.Request.Context(), id, productID); err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal)
		return
	}
	writeJSON(c, 200, gin.H{"message": "part installed"})
}

func (h *InventoryHandler) RemovePart(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		exception.WriteError(c, exception.ErrInvalidResourceID)
		return
	}
	if err := h.svc.RemovePart(c.Request.Context(), id); err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal)
		return
	}
	writeJSON(c, 200, gin.H{"message": "part removed"})
}

func (h *InventoryHandler) GetPartCatalog(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		exception.WriteError(c, exception.ErrInvalidResourceID)
		return
	}
	cat, err := h.svc.GetPartCatalog(c.Request.Context(), id)
	if err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal)
		return
	}
	price := 0.0
	stock := 0
	if cat != nil {
		if _, p, s, err := h.svc.GetPartCatalogBySKU(c.Request.Context(), cat.PartNumber); err == nil {
			price = p
			stock = s
		}
	}
	writeJSON(c, 200, gin.H{
		"id":            cat.ID,
		"part_number":   cat.PartNumber,
		"part_types_id": cat.PartTypesID,
		"mfg_number":    cat.MfgNumber,
		"description":   cat.Description,
		"status":        cat.PartMfgStatus,
		"price":         price,
		"stock_qty":     stock,
	})
}

func (h *InventoryHandler) ListPartCatalog(c *gin.Context) {
	var typeID *int32
	if v := c.Query("type_id"); v != "" {
		if parsed, err := parseInt32(v); err == nil {
			typeID = &parsed
		}
	}

	params := parsePaginationParams(c)
	q := c.Query("q")

	catalogs, meta, err := h.svc.ListPartCatalog(c.Request.Context(), typeID, params, q)
	if err != nil {
		exception.WriteError(c, exception.ErrInternal.WithError(err))
		return
	}
	writeJSON(c, 200, pagination.Response{Data: catalogs, Pagination: *meta})
}

func (h *InventoryHandler) UpdateProduct(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		exception.WriteError(c, exception.ErrInvalidResourceID)
		return
	}
	var fields map[string]any
	if err := c.ShouldBindJSON(&fields); err != nil {
		exception.WriteError(c, exception.ErrInvalidBody)
		return
	}
	delete(fields, "id")
	delete(fields, "ID")
	delete(fields, "created_at")
	delete(fields, "createdAt")
	delete(fields, "updated_at")
	delete(fields, "updatedAt")
	delete(fields, "deleted_at")
	delete(fields, "deletedAt")

	p, err := h.svc.UpdateProduct(c.Request.Context(), id, fields)
	if err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal)
		return
	}
	writeJSON(c, 200, p)
}

func (h *InventoryHandler) UpdatePart(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		exception.WriteError(c, exception.ErrInvalidResourceID)
		return
	}
	var fields map[string]any
	if err := c.ShouldBindJSON(&fields); err != nil {
		exception.WriteError(c, exception.ErrInvalidBody)
		return
	}
	delete(fields, "id")
	delete(fields, "ID")
	delete(fields, "created_at")
	delete(fields, "createdAt")
	delete(fields, "updated_at")
	delete(fields, "updatedAt")
	delete(fields, "deleted_at")
	delete(fields, "deletedAt")

	p, err := h.svc.UpdatePart(c.Request.Context(), id, fields)
	if err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal)
		return
	}
	writeJSON(c, 200, p)
}

func (h *InventoryHandler) RegisterProduct(c *gin.Context) {
	var req struct {
		ProductModelCode string    `json:"product_model_code" binding:"required"`
		CustomerID       uuid.UUID `json:"customer_id" binding:"required"`
		ProductName      string    `json:"product_name" binding:"required"`
		SerialNumber     string    `json:"serial_number" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		exception.WriteError(c, exception.ErrInvalidBody)
		return
	}
	p := models.Product{
		ID:               uuid.New(),
		ProductModelCode: req.ProductModelCode,
		CustomerID:       req.CustomerID,
		ProductName:      req.ProductName,
		SerialNumber:     req.SerialNumber,
	}
	if err := h.svc.CreateProduct(c.Request.Context(), &p); err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal)
		return
	}
	writeJSON(c, 201, p)
}

func (h *InventoryHandler) CreatePartCatalog(c *gin.Context) {
	var req struct {
		SKU         string  `json:"sku" binding:"required"`
		Description string  `json:"description"`
		Price       float64 `json:"price"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		exception.WriteError(c, exception.ErrInvalidBody)
		return
	}

	pc := models.PartCatalog{
		ID:            uuid.New(),
		PartNumber:    req.SKU,
		PartTypesID:   1, // default part type
		MfgNumber:     "MFG-" + req.SKU,
		Description:   &req.Description,
		PartMfgStatus: 1, // Active
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := h.svc.CreatePartCatalog(c.Request.Context(), &pc, req.Price); err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal.WithError(err))
		return
	}

	writeJSON(c, 201, gin.H{
		"id":          pc.ID,
		"sku":         pc.PartNumber,
		"description": pc.Description,
		"price":       req.Price,
	})
}

func (h *InventoryHandler) UpdatePartCatalog(c *gin.Context) {
	sku := c.Param("sku")
	if sku == "" {
		exception.WriteError(c, exception.ErrInvalidResourceID)
		return
	}

	var req struct {
		Description *string  `json:"description"`
		Price       *float64 `json:"price"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		exception.WriteError(c, exception.ErrInvalidBody)
		return
	}

	fields := make(map[string]any)
	if req.Description != nil {
		fields["description"] = *req.Description
	}
	if req.Price != nil {
		fields["price"] = *req.Price
	}

	pc, err := h.svc.UpdatePartCatalogBySKU(c.Request.Context(), sku, fields)
	if err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal.WithError(err))
		return
	}

	price := 0.0
	if req.Price != nil {
		price = *req.Price
	} else {
		if _, p, _, err := h.svc.GetPartCatalogBySKU(c.Request.Context(), sku); err == nil {
			price = p
		}
	}

	writeJSON(c, 200, gin.H{
		"id":          pc.ID,
		"sku":         pc.PartNumber,
		"description": pc.Description,
		"price":       price,
	})
}

func (h *InventoryHandler) DeletePartCatalog(c *gin.Context) {
	sku := c.Param("sku")
	if sku == "" {
		exception.WriteError(c, exception.ErrInvalidResourceID)
		return
	}

	if err := h.svc.DeletePartCatalogBySKU(c.Request.Context(), sku); err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal.WithError(err))
		return
	}

	writeJSON(c, 200, gin.H{"message": "catalog part deleted"})
}

func (h *InventoryHandler) GetPartCatalogBySKU(c *gin.Context) {
	sku := c.Param("sku")
	if sku == "" {
		exception.WriteError(c, exception.ErrInvalidResourceID)
		return
	}

	pc, price, stock, err := h.svc.GetPartCatalogBySKU(c.Request.Context(), sku)
	if err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal.WithError(err))
		return
	}

	writeJSON(c, 200, gin.H{
		"id":          pc.ID,
		"sku":         pc.PartNumber,
		"description": pc.Description,
		"price":       price,
		"stock_qty":   stock,
	})
}

func (h *InventoryHandler) ListStocks(c *gin.Context) {
	params := parsePaginationParams(c)
	status := c.Query("status")
	q := c.Query("q")
	var supplierID *uuid.UUID
	if v := c.Query("supplier_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			supplierID = &id
		}
	}

	stocks, meta, err := h.svc.ListStocks(c.Request.Context(), params, status, q, supplierID)
	if err != nil {
		exception.WriteError(c, exception.ErrInternal.WithError(err))
		return
	}
	writeJSON(c, 200, pagination.Response{Data: stocks, Pagination: *meta})
}

type createComponentStockRequest struct {
	SKU               string  `json:"sku" binding:"required"`
	Name              string  `json:"name" binding:"required"`
	Category          string  `json:"category" binding:"required"`
	StockQty          int     `json:"stock_qty"`
	ReorderPoint      int     `json:"reorder_point"`
	UnitCost          float64 `json:"unit_cost" binding:"required"`
	Location          string  `json:"location"`
	PrimarySupplierID string  `json:"primary_supplier_id"`
}

func (h *InventoryHandler) CreateComponentStock(c *gin.Context) {
	var req createComponentStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		exception.WriteError(c, exception.ErrInvalidBody)
		return
	}

	stock := &models.ComponentStock{
		SKU:          req.SKU,
		Name:         req.Name,
		Category:     req.Category,
		StockQty:     req.StockQty,
		ReorderPoint: req.ReorderPoint,
		UnitCost:     req.UnitCost,
		Location:     req.Location,
	}
	if req.PrimarySupplierID != "" {
		if supplierID, err := uuid.Parse(req.PrimarySupplierID); err == nil {
			stock.PrimarySupplierID = supplierID
		}
	}

	if err := h.svc.CreateComponentStock(c.Request.Context(), stock); err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal.WithError(err))
		return
	}
	writeJSON(c, 201, stock)
}

func (h *InventoryHandler) GetStockBySKU(c *gin.Context) {
	sku := c.Param("sku")
	if sku == "" {
		exception.WriteError(c, exception.ErrInvalidResourceID)
		return
	}

	stock, err := h.svc.GetStockBySKU(c.Request.Context(), sku)
	if err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal.WithError(err))
		return
	}
	writeJSON(c, 200, stock)
}

func (h *InventoryHandler) GetInventoryMetrics(c *gin.Context) {
	totalSKUs, lowStock, outOfStock, stockValue, err := h.svc.GetInventoryMetrics(c.Request.Context())
	if err != nil {
		exception.WriteError(c, exception.ErrInternal.WithError(err))
		return
	}
	writeJSON(c, 200, gin.H{
		"total_skus":   totalSKUs,
		"low_stock":    lowStock,
		"out_of_stock": outOfStock,
		"stock_value":  stockValue,
	})
}

func (h *InventoryHandler) ExportInventoryReport(c *gin.Context) {
	stocks, err := h.svc.FindAllStocks(c.Request.Context())
	if err != nil {
		exception.WriteError(c, exception.ErrInternal.WithError(err))
		return
	}

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", `attachment; filename="inventory_report.csv"`)
	c.Header("Content-Type", "text/csv")
	c.Header("Transfer-Encoding", "chunked")
	c.Writer.WriteHeader(http.StatusOK)

	csvWriter := csv.NewWriter(c.Writer)
	header := []string{
		"SKU", "Name", "Category", "Stock Qty", "Reorder Point",
		"Unit Cost", "Status", "Primary Supplier", "Lead Time Days", "Location",
	}
	if err := csvWriter.Write(header); err != nil {
		return
	}
	csvWriter.Flush()
	c.Writer.Flush()

	for _, s := range stocks {
		row := []string{
			s.SKU,
			s.Name,
			s.Category,
			strconv.Itoa(s.StockQty),
			strconv.Itoa(s.ReorderPoint),
			strconv.FormatFloat(s.UnitCost, 'f', 2, 64),
			string(s.Status),
			s.PrimarySupplier,
			strconv.Itoa(s.LeadTimeDays),
			s.Location,
		}
		if err := csvWriter.Write(row); err != nil {
			return
		}
	}

	csvWriter.Flush()
	c.Writer.Flush()
}
