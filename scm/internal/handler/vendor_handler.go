package handler

import (
	"net/http"

	"zeus-be/pkg/response"
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "sku query param required"})
		return
	}
	supplier, mapping, err := h.svc.GetOptimalSupplier(c.Request.Context(), sku)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	response.OK(c, gin.H{
		"supplier": supplier,
		"mapping":  mapping,
	})
}

func (h *VendorHandler) UpdateSupplierMetrics(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid supplier id"})
		return
	}
	if err := h.svc.UpdateSupplierMetrics(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response.OKWithMessage(c, http.StatusOK, "metrics updated")
}
