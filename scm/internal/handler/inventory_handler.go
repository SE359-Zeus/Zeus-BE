package handler

import (
	"net/http"
	"strconv"

	"zeus-be/pkg/response"
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	p, err := h.svc.GetProduct(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	response.OK(c, p)
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response.OK(c, pagination.Response{Data: products, Pagination: *meta})
}

func (h *InventoryHandler) CreateProduct(c *gin.Context) {
	var p models.Product
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.CreateProduct(c.Request.Context(), &p); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response.Created(c, p)
}

func (h *InventoryHandler) GetProductModel(c *gin.Context) {
	code := c.Param("code")
	m, err := h.svc.GetProductModel(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	response.OK(c, m)
}

func (h *InventoryHandler) CreateProductModel(c *gin.Context) {
	var m models.ProductModel
	if err := c.ShouldBindJSON(&m); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.CreateProductModel(c.Request.Context(), &m); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response.Created(c, m)
}

func (h *InventoryHandler) GetPart(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	p, err := h.svc.GetPart(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	response.OK(c, p)
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response.OK(c, pagination.Response{Data: parts, Pagination: *meta})
}

func (h *InventoryHandler) CreatePart(c *gin.Context) {
	var p models.Part
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.CreatePart(c.Request.Context(), &p); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response.Created(c, p)
}

func (h *InventoryHandler) UpdatePartCondition(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		ConditionID int32 `json:"condition_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.UpdatePartCondition(c.Request.Context(), id, req.ConditionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response.OKWithMessage(c, http.StatusOK, "condition updated")
}

func (h *InventoryHandler) MarkPartScrapped(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.svc.MarkPartScrapped(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response.OKWithMessage(c, http.StatusOK, "part scrapped")
}

func (h *InventoryHandler) InstallPart(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		ProductID string `json:"product_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	productID, err := uuid.Parse(req.ProductID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product_id"})
		return
	}
	if err := h.svc.InstallPart(c.Request.Context(), id, productID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response.OKWithMessage(c, http.StatusOK, "part installed")
}

func (h *InventoryHandler) RemovePart(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.svc.RemovePart(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response.OKWithMessage(c, http.StatusOK, "part removed")
}

func (h *InventoryHandler) GetPartCatalog(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	cat, err := h.svc.GetPartCatalog(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	response.OK(c, cat)
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response.OK(c, pagination.Response{Data: catalogs, Pagination: *meta})
}

func (h *InventoryHandler) UpdateProduct(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var fields map[string]any
	if err := c.ShouldBindJSON(&fields); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// strip internal fields
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
		if err == service.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response.OK(c, p)
}

func (h *InventoryHandler) UpdatePart(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var fields map[string]any
	if err := c.ShouldBindJSON(&fields); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// strip internal fields
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
		if err == service.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response.OK(c, p)
}
