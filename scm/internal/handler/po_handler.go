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

func (h *POHandler) ExportPOReport(c *gin.Context) {
	pos, err := h.svc.FindAllPOs(c.Request.Context())
	if err != nil {
		exception.WriteError(c, exception.Resolve(err))
		return
	}

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", `attachment; filename="purchase_orders_report.csv"`)
	c.Header("Content-Type", "text/csv")
	c.Header("Transfer-Encoding", "chunked")
	c.Writer.WriteHeader(http.StatusOK)

	csvWriter := csv.NewWriter(c.Writer)
	header := []string{
		"PO ID", "Vendor Name", "Target Build", "Status", "Expected Delivery",
		"Total Value", "Notes", "Line Item SKU", "Line Item Description",
		"Ordered Qty", "Unit Price", "Line Item Subtotal",
	}
	if err := csvWriter.Write(header); err != nil {
		return
	}
	csvWriter.Flush()
	c.Writer.Flush()

	for _, po := range pos {
		if len(po.LineItems) == 0 {
			row := []string{
				po.ID,
				po.VendorName,
				po.TargetBuild,
				string(po.Status),
				po.ExpectedDelivery.Format(time.RFC3339),
				strconv.FormatFloat(po.TotalValue, 'f', 2, 64),
				po.Notes,
				"", "", "", "", "",
			}
			if err := csvWriter.Write(row); err != nil {
				return
			}
			continue
		}

		for _, item := range po.LineItems {
			row := []string{
				po.ID,
				po.VendorName,
				po.TargetBuild,
				string(po.Status),
				po.ExpectedDelivery.Format(time.RFC3339),
				strconv.FormatFloat(po.TotalValue, 'f', 2, 64),
				po.Notes,
				item.SKU,
				item.Description,
				strconv.Itoa(item.OrderedQty),
				strconv.FormatFloat(item.UnitPrice, 'f', 2, 64),
				strconv.FormatFloat(float64(item.OrderedQty)*item.UnitPrice, 'f', 2, 64),
			}
			if err := csvWriter.Write(row); err != nil {
				return
			}
		}
	}

	csvWriter.Flush()
	c.Writer.Flush()
}

type createPOLineItemRequest struct {
	SKU string `json:"sku" binding:"required"`
	Qty int    `json:"qty" binding:"required,min=1"`
}

type createPORequest struct {
	ID               string                    `json:"id" binding:"required"`
	ExpectedDelivery time.Time                 `json:"expected_delivery" binding:"required"`
	VendorID         string                    `json:"vendor_id" binding:"required"`
	TargetBuild      string                    `json:"target_build"`
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
		TargetBuild:      req.TargetBuild,
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
