package handler

import (
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"zeus-scm-service/internal/exception"
	"zeus-scm-service/internal/handler/middleware"
	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/pagination"
	"zeus-scm-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
	ctx := context.WithValue(c.Request.Context(), middleware.ContextKeyFullName, c.GetString(middleware.ContextKeyFullName))
	if err := h.svc.AcquireDispatchLock(ctx, shipmentID, req.OperatorID); err != nil {
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
	ctx := context.WithValue(c.Request.Context(), middleware.ContextKeyFullName, c.GetString(middleware.ContextKeyFullName))
	if err := h.svc.DispatchShipment(ctx, shipmentID, req.OperatorID); err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal)
		return
	}
	writeJSON(c, 200, gin.H{"message": "shipment dispatched"})
}

type markDeliveredRequest struct {
	OperatorID string `json:"operator_id" binding:"required"`
}

func (h *ShipmentHandler) MarkDelivered(c *gin.Context) {
	shipmentID := c.Param("shipmentId")
	var req markDeliveredRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		exception.WriteError(c, exception.ErrInvalidBody)
		return
	}
	ctx := context.WithValue(c.Request.Context(), middleware.ContextKeyFullName, c.GetString(middleware.ContextKeyFullName))
	if err := h.svc.MarkDelivered(ctx, shipmentID, req.OperatorID); err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal)
		return
	}
	writeJSON(c, 200, gin.H{"message": "shipment delivered"})
}

type transitionShipmentStateRequest struct {
	NewState string `json:"new_state" binding:"required"`
}

func (h *ShipmentHandler) TransitionState(c *gin.Context) {
	shipmentID := c.Param("shipmentId")
	var req transitionShipmentStateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		exception.WriteError(c, exception.ErrInvalidBody)
		return
	}
	if err := h.svc.TransitionState(c.Request.Context(), shipmentID, models.ShipmentStatus(req.NewState)); err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal)
		return
	}
	writeJSON(c, 200, gin.H{"message": "state transitioned"})
}

func (h *ShipmentHandler) ListShipments(c *gin.Context) {
	status := c.Query("status")
	params := parsePaginationParams(c)

	shipments, meta, err := h.svc.ListShipments(c.Request.Context(), status, params)
	if err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal.WithError(err))
		return
	}
	writeJSON(c, 200, pagination.Response{Data: shipments, Pagination: *meta})
}

func (h *ShipmentHandler) GetShipment(c *gin.Context) {
	shipmentID := c.Param("shipmentId")
	shipment, err := h.svc.GetShipment(c.Request.Context(), shipmentID)
	if err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal.WithError(err))
		return
	}
	writeJSON(c, 200, shipment)
}

type createShipmentRequest struct {
	PORef      string    `json:"po_ref" binding:"required"`
	SupplierID uuid.UUID `json:"supplier_id" binding:"required"`
	Carrier    string    `json:"carrier" binding:"required"`
	TrackingNo string    `json:"tracking_no"`
	Origin     string    `json:"origin"`
	ShipDate   time.Time `json:"ship_date"`
}

func (h *ShipmentHandler) CreateShipment(c *gin.Context) {
	var req createShipmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		exception.WriteError(c, exception.ErrInvalidBody)
		return
	}

	year := time.Now().Year()
	shipmentID := fmt.Sprintf("SHP-%d-%d", year, time.Now().UnixNano()%100000)

	shipDate := req.ShipDate
	if shipDate.IsZero() {
		shipDate = time.Now()
	}

	shipment := &models.Shipment{
		ID:         shipmentID,
		PORef:      req.PORef,
		SupplierID: req.SupplierID,
		Status:     models.ShipmentStatusScheduled,
		Carrier:    req.Carrier,
		TrackingNo: req.TrackingNo,
		Origin:     req.Origin,
		ShipDate:   shipDate,
	}

	if err := h.svc.CreateShipment(c.Request.Context(), shipment); err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal.WithError(err))
		return
	}
	writeJSON(c, 201, shipment)
}

func (h *ShipmentHandler) GetMetrics(c *gin.Context) {
	total, inTransit, delayed, onTimeRate, err := h.svc.GetMetrics(c.Request.Context())
	if err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal.WithError(err))
		return
	}
	writeJSON(c, 200, gin.H{
		"total_shipments": total,
		"in_transit":      inTransit,
		"delayed":         delayed,
		"on_time_rate":    onTimeRate,
	})
}

func (h *ShipmentHandler) ListCarriers(c *gin.Context) {
	carriers, err := h.svc.ListCarriers(c.Request.Context())
	if err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal.WithError(err))
		return
	}
	writeJSON(c, 200, carriers)
}

func (h *ShipmentHandler) ExportShipmentReport(c *gin.Context) {
	shipments, err := h.svc.FindAllShipments(c.Request.Context())
	if err != nil {
		exception.WriteError(c, exception.Resolve(err))
		return
	}

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", `attachment; filename="shipments_report.csv"`)
	c.Header("Content-Type", "text/csv")
	c.Header("Transfer-Encoding", "chunked")
	c.Writer.WriteHeader(http.StatusOK)

	csvWriter := csv.NewWriter(c.Writer)
	header := []string{
		"Shipment ID", "PO Ref", "Supplier", "Status", "Carrier",
		"Tracking No", "Origin", "Ship Date", "ETA",
		"Item SKU", "Item Description", "Item Qty",
	}
	if err := csvWriter.Write(header); err != nil {
		return
	}
	csvWriter.Flush()
	c.Writer.Flush()

	for _, s := range shipments {
		if len(s.Items) == 0 {
			row := []string{
				s.ID, s.PORef, s.SupplierName, string(s.Status),
				s.Carrier, s.TrackingNo, s.Origin,
				s.ShipDate.Format(time.RFC3339),
				s.ETA.Format(time.RFC3339),
				"", "", "",
			}
			if err := csvWriter.Write(row); err != nil {
				return
			}
			continue
		}
		for _, item := range s.Items {
			row := []string{
				s.ID, s.PORef, s.SupplierName, string(s.Status),
				s.Carrier, s.TrackingNo, s.Origin,
				s.ShipDate.Format(time.RFC3339),
				s.ETA.Format(time.RFC3339),
				item.SKU, item.Description, strconv.Itoa(item.Qty),
			}
			if err := csvWriter.Write(row); err != nil {
				return
			}
		}
	}

	csvWriter.Flush()
	c.Writer.Flush()
}
