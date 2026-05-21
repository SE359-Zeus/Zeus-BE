package handler

import (
	"zeus-be/pkg/exception"
	"zeus-scm-service/internal/models"
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
	c.JSON(201, po)
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
	c.JSON(200, gin.H{"message": "line item added"})
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
	c.JSON(200, gin.H{"message": "PO approved"})
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
	c.JSON(200, gin.H{"message": "state transitioned"})
}
