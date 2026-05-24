package handler

import (
	"zeus-scm-service/internal/exception"
	"zeus-scm-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type VendorHandler struct {
	svc service.IVendorService
}

func NewVendorHandler(svc service.IVendorService) *VendorHandler {
	return &VendorHandler{svc: svc}
}

func (h *VendorHandler) GetOptimalSupplier(c *gin.Context) {
	sku := c.Query("sku")
	if sku == "" {
		exception.WriteError(c, exception.ErrInvalidInput.WithMessage("sku query param required"))
		return
	}
	supplier, mapping, err := h.svc.GetOptimalSupplier(c.Request.Context(), sku)
	if err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal)
		return
	}
	writeJSON(c, 200, gin.H{
		"supplier": supplier,
		"mapping":  mapping,
	})
}

func (h *VendorHandler) UpdateSupplierMetrics(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		exception.WriteError(c, exception.ErrInvalidResourceID.WithMessage("invalid supplier id"))
		return
	}
	if err := h.svc.UpdateSupplierMetrics(c.Request.Context(), id); err != nil {
		exception.WriteError(c, exception.ErrInternal.WithError(err))
		return
	}
	writeJSON(c, 200, gin.H{"message": "metrics updated"})
}
