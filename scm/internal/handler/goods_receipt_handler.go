package handler

import (
	"zeus-scm-service/internal/exception"
	"zeus-scm-service/internal/pagination"
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
	writeJSON(c, 200, gin.H{"message": "lock acquired"})
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
	writeJSON(c, 200, gin.H{"message": "blind receipt processed"})
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
	writeJSON(c, 200, gin.H{"message": "lock released"})
}

func (h *GoodsReceiptHandler) ListGRs(c *gin.Context) {
	status := c.Query("status")
	params := parsePaginationParams(c)

	grs, meta, err := h.svc.ListGRs(c.Request.Context(), status, params)
	if err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal.WithError(err))
		return
	}
	writeJSON(c, 200, pagination.Response{Data: grs, Pagination: *meta})
}

func (h *GoodsReceiptHandler) GetGR(c *gin.Context) {
	grID := c.Param("grId")
	gr, err := h.svc.GetGR(c.Request.Context(), grID)
	if err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal.WithError(err))
		return
	}
	writeJSON(c, 200, gr)
}

func (h *GoodsReceiptHandler) GetMetrics(c *gin.Context) {
	pending, completedToday, discrepancies, queue, err := h.svc.GetMetrics(c.Request.Context())
	if err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal.WithError(err))
		return
	}
	writeJSON(c, 200, gin.H{
		"pending_receipts":     pending,
		"completed_today":      completedToday,
		"active_discrepancies": discrepancies,
		"inspection_queue":     queue,
	})
}
