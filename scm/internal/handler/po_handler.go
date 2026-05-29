package handler

import (
	"time"
	"zeus-scm-service/internal/exception"
	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/pagination"
	"zeus-scm-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type POHandler struct {
	svc service.IPOService
}

func NewPOHandler(svc service.IPOService) *POHandler {
	return &POHandler{svc: svc}
}

type createDraftRequest struct {
	VendorID    string `json:"vendor_id" binding:"required"`
	TargetBuild string `json:"target_build"`
}

func (h *POHandler) CreateDraft(c *gin.Context) {
	var req createDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		exception.WriteError(c, exception.ErrInvalidBody)
		return
	}
	vendorID, err := uuid.Parse(req.VendorID)
	if err != nil {
		exception.WriteError(c, exception.ErrInvalidResourceID.WithMessage("invalid vendor_id"))
		return
	}
	po, err := h.svc.CreateDraft(c.Request.Context(), vendorID, req.TargetBuild)
	if err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal)
		return
	}
	writeJSON(c, 201, po)
}

type addLineItemRequest struct {
	SKU string `json:"sku" binding:"required"`
	Qty int    `json:"qty" binding:"required,min=1"`
}

func (h *POHandler) AddLineItemWithLock(c *gin.Context) {
	poID := c.Param("poId")
	var req addLineItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		exception.WriteError(c, exception.ErrInvalidBody)
		return
	}
	if err := h.svc.AddLineItemWithLock(c.Request.Context(), poID, req.SKU, req.Qty); err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal)
		return
	}
	writeJSON(c, 200, gin.H{"message": "line item added"})
}

func (h *POHandler) ApprovePO(c *gin.Context) {
	poID := c.Param("poId")
	if err := h.svc.ApprovePO(c.Request.Context(), poID); err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal)
		return
	}
	writeJSON(c, 200, gin.H{"message": "PO approved"})
}

type transitionStateRequest struct {
	NewState string `json:"new_state" binding:"required"`
}

func (h *POHandler) TransitionState(c *gin.Context) {
	poID := c.Param("poId")
	var req transitionStateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		exception.WriteError(c, exception.ErrInvalidBody)
		return
	}
	if err := h.svc.TransitionState(c.Request.Context(), poID, models.POStatus(req.NewState)); err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal)
		return
	}
	writeJSON(c, 200, gin.H{"message": "state transitioned"})
}

func (h *POHandler) ListPOs(c *gin.Context) {
	params := parsePaginationParams(c)
	q := c.Query("q")

	pos, meta, err := h.svc.ListPOs(c.Request.Context(), params, q)
	if err != nil {
		exception.WriteError(c, exception.Resolve(err))
		return
	}
	writeJSON(c, 200, pagination.Response{
		Data:       pos,
		Pagination: *meta,
	})
}

func (h *POHandler) GetPO(c *gin.Context) {
	poID := c.Param("poId")
	po, err := h.svc.GetPO(c.Request.Context(), poID)
	if err != nil {
		exception.WriteError(c, exception.Resolve(err))
		return
	}
	writeJSON(c, 200, po)
}

type createPOLineItemRequest struct {
	SKU string `json:"sku" binding:"required"`
	Qty int    `json:"qty" binding:"required,min=1"`
}

type createPORequest struct {
	ID               string                    `json:"id" binding:"required"`
	ExpectedDelivery time.Time                 `json:"expected_delivery" binding:"required"`
	VendorID         string                    `json:"vendor_id" binding:"required"`
	Items            []createPOLineItemRequest `json:"items" binding:"required,min=1"`
	Notes            string                    `json:"notes"`
}

func (h *POHandler) CreatePO(c *gin.Context) {
	var req createPORequest
	if err := c.ShouldBindJSON(&req); err != nil {
		exception.WriteError(c, exception.ErrInvalidBody)
		return
	}

	vendorUUID, err := uuid.Parse(req.VendorID)
	if err != nil {
		exception.WriteError(c, exception.ErrInvalidResourceID.WithMessage("invalid vendor_id"))
		return
	}

	lineItems := make([]models.POLineItem, len(req.Items))
	for i, item := range req.Items {
		lineItems[i] = models.POLineItem{
			SKU:        item.SKU,
			OrderedQty: item.Qty,
		}
	}

	poModel := &models.PurchaseOrder{
		ID:               req.ID,
		VendorID:         vendorUUID,
		ExpectedDelivery: req.ExpectedDelivery,
		Notes:            req.Notes,
		LineItems:        lineItems,
	}

	if err := h.svc.CreatePO(c.Request.Context(), poModel); err != nil {
		exception.WriteError(c, exception.Resolve(err))
		return
	}

	writeJSON(c, 201, poModel)
}
