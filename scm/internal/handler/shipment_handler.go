package handler

import (
	"zeus-scm-service/internal/exception"
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
		exception.WriteError(c, exception.ErrInvalidBody)
		return
	}
	if err := h.svc.AcquireDispatchLock(c.Request.Context(), shipmentID, req.OperatorID); err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal)
		return
	}
	writeJSON(c, 200, gin.H{"message": "dispatch lock acquired"})
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
		exception.WriteError(c, exception.ErrInvalidBody)
		return
	}
	if err := h.svc.DispatchShipment(c.Request.Context(), shipmentID, req.OperatorID); err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal)
		return
	}
	writeJSON(c, 200, gin.H{"message": "shipment dispatched"})
}
