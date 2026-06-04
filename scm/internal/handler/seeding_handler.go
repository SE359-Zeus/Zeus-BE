package handler

import (
	"time"

	"zeus-scm-service/internal/exception"
	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type SeedingHandler struct {
	svc service.ISeedingService
}

func NewSeedingHandler(svc service.ISeedingService) *SeedingHandler {
	return &SeedingHandler{svc: svc}
}

func (h *SeedingHandler) CreateProduct(c *gin.Context) {
	var req struct {
		ProductModelCode string `json:"product_model_code" binding:"required"`
		ProductName      string `json:"product_name" binding:"required"`
		SerialNumber     string `json:"serial_number" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		exception.WriteError(c, exception.ErrInvalidBody)
		return
	}

	p := models.Product{
		ID:               uuid.New(),
		ProductModelCode: req.ProductModelCode,
		CustomerID:       &uuid.Nil,
		ProductName:      req.ProductName,
		SerialNumber:     req.SerialNumber,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
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

func (h *SeedingHandler) RegisterProduct(c *gin.Context) {
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
		CustomerID:       &req.CustomerID,
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

func (h *SeedingHandler) CreatePart(c *gin.Context) {
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
