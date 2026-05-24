package handler

import (
	"strconv"

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

	products, meta, err := h.svc.ListProducts(c.Request.Context(), params, q)
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
		exception.WriteError(c, exception.ErrInvalidResourceID.WithMessage("invalid product_id"))
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
	writeJSON(c, 200, cat)
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
