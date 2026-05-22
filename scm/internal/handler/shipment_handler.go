package handler

import (
	"net/http"

	"zeus-be/pkg/response"
	"zeus-scm-service/internal/service"

	"github.com/gin-gonic/gin"
)

type ShipmentHandler struct {
	svc service.IShipmentService
}

func NewShipmentHandler(svc service.IShipmentService) *ShipmentHandler {
	return &ShipmentHandler{svc: svc}
}

type acquireDispatchLockRequest struct {
	OperatorID string `json:"operator_id" binding:"required"`
}

func (h *ShipmentHandler) AcquireDispatchLock(c *gin.Context) {
	shipmentID := c.Param("shipmentId")
	var req acquireDispatchLockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.AcquireDispatchLock(c.Request.Context(), shipmentID, req.OperatorID); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	response.OKWithMessage(c, http.StatusOK, "dispatch lock acquired")
}

type dispatchShipmentRequest struct {
	OperatorID string `json:"operator_id" binding:"required"`
	Carrier    string `json:"carrier"`
	TrackingNo string `json:"tracking_no"`
}

func (h *ShipmentHandler) DispatchShipment(c *gin.Context) {
	shipmentID := c.Param("shipmentId")
	var req dispatchShipmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.DispatchShipment(c.Request.Context(), shipmentID, req.OperatorID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response.OKWithMessage(c, http.StatusOK, "shipment dispatched")
}
