package handler

import (
	"encoding/csv"
	"net/http"
	"strconv"
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
	VendorID   string `json:"vendor_id"`
	SupplierID string `json:"supplier_id"`
}

func (h *POHandler) CreateDraft(c *gin.Context) {
	var req createDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		exception.WriteError(c, exception.ErrInvalidBody)
		return
	}
	supplierStr := req.SupplierID
	if supplierStr == "" {
		supplierStr = req.VendorID
	}
	if supplierStr == "" {
		exception.WriteError(c, exception.ErrInvalidBody.WithMessage("supplier_id is required"))
		return
	}
	vendorID, err := uuid.Parse(supplierStr)
	if err != nil {
		exception.WriteError(c, exception.ErrInvalidResourceID.WithMessage("invalid supplier_id"))
		return
	}
	po, err := h.svc.CreateDraft(c.Request.Context(), vendorID)
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
		"PO ID", "Vendor Name", "Status",
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
				string(po.Status),
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
				string(po.Status),
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

func (h *POHandler) GetMetrics(c *gin.Context) {
	total, draft, approved, inTransit, received, partial, void, err := h.svc.GetMetrics(c.Request.Context())
	if err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal.WithError(err))
		return
	}
	writeJSON(c, 200, gin.H{
		"total":      total,
		"draft":      draft,
		"approved":   approved,
		"in_transit": inTransit,
		"received":   received,
		"partial":    partial,
		"void":       void,
	})
}

type createPOLineItemRequest struct {
	SKU        string `json:"sku"`
	PartNumber string `json:"part_number"`
	Qty        int    `json:"qty"`
	Quantity   int    `json:"quantity"`
}

type createPORequest struct {
	ID             string                    `json:"id"`
	PONumber       string                    `json:"po_number"`
	VendorID       string                    `json:"vendor_id"`
	SupplierID     string                    `json:"supplier_id"`
	Supplier       string                    `json:"supplier"`
	Items          []createPOLineItemRequest `json:"items"`
	ListOrderItems []createPOLineItemRequest `json:"list_order_items"`
	Notes          string                    `json:"notes"`
}

func (h *POHandler) CreatePO(c *gin.Context) {
	var req createPORequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid request body: " + err.Error(),
		})
		return
	}

	poID := req.PONumber
	if poID == "" {
		poID = req.ID
	}
	if poID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "po_number or id is required",
		})
		return
	}

	supplierStr := req.Supplier
	if supplierStr == "" {
		supplierStr = req.SupplierID
	}
	if supplierStr == "" {
		supplierStr = req.VendorID
	}
	if supplierStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "supplier_id is required",
		})
		return
	}

	vendorUUID, err := uuid.Parse(supplierStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid supplier uuid: " + err.Error(),
		})
		return
	}

	itemsReq := req.Items
	if len(itemsReq) == 0 {
		itemsReq = req.ListOrderItems
	}
	if len(itemsReq) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "items list is required and cannot be empty",
		})
		return
	}

	lineItems := make([]models.POLineItem, len(itemsReq))
	for i, item := range itemsReq {
		itemSKU := item.PartNumber
		if itemSKU == "" {
			itemSKU = item.SKU
		}
		if itemSKU == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "sku or part_number is required for all line items",
			})
			return
		}

		itemQty := item.Quantity
		if itemQty <= 0 {
			itemQty = item.Qty
		}
		if itemQty <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "quantity/qty must be greater than 0 for all line items",
			})
			return
		}

		lineItems[i] = models.POLineItem{
			SKU:        itemSKU,
			OrderedQty: itemQty,
		}
	}

	poModel := &models.PurchaseOrder{
		ID:        poID,
		VendorID:  vendorUUID,
		Notes:     req.Notes,
		LineItems: lineItems,
	}

	if err := h.svc.CreatePO(c.Request.Context(), poModel); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":     true,
		"total_value": poModel.TotalValue,
	})
}
