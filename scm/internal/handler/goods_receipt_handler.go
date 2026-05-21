package handler

import (
	"zeus-be/pkg/exception"
	"zeus-scm-service/internal/service"

	"github.com/gin-gonic/gin"
)

type GoodsReceiptHandler struct {
	svc service.IGoodsReceiptService
}

func NewGoodsReceiptHandler(svc service.IGoodsReceiptService) *GoodsReceiptHandler {
	return &GoodsReceiptHandler{svc: svc}
}

type acquireLockRequest struct {
	OperatorID string `json:"operator_id" binding:"required"`
}

func (h *GoodsReceiptHandler) AcquireLock(c *gin.Context) {
	grID := c.Param("grId")
	var req acquireLockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		exception.WriteError(c, exception.ErrInvalidBody)
		return
	}
	if err := h.svc.AcquireLock(c.Request.Context(), grID, req.OperatorID); err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal)
		return
	}
	c.JSON(200, gin.H{"message": "lock acquired"})
}

type lineItemCount struct {
	Received  int `json:"received" binding:"required,min=0"`
	Defective int `json:"defective" binding:"required,min=0"`
}

type processBlindReceiptRequest struct {
	OperatorID string                   `json:"operator_id" binding:"required"`
	Counts     map[string]lineItemCount `json:"counts" binding:"required"`
}

func (h *GoodsReceiptHandler) ProcessBlindReceipt(c *gin.Context) {
	grID := c.Param("grId")
	var req processBlindReceiptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		exception.WriteError(c, exception.ErrInvalidBody)
		return
	}
	counts := make(map[string]struct {
		Received  int
		Defective int
	})
	for sku, cnt := range req.Counts {
		counts[sku] = struct {
			Received  int
			Defective int
		}{Received: cnt.Received, Defective: cnt.Defective}
	}
	if err := h.svc.ProcessBlindReceipt(c.Request.Context(), grID, req.OperatorID, counts); err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal)
		return
	}
	c.JSON(200, gin.H{"message": "blind receipt processed"})
}

func (h *GoodsReceiptHandler) ReleaseLock(c *gin.Context) {
	grID := c.Param("grId")
	if err := h.svc.ReleaseLock(c.Request.Context(), grID); err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal)
		return
	}
	c.JSON(200, gin.H{"message": "lock released"})
}
