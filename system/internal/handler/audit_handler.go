package handler

import (
	"strconv"
	"time"

	"zeus-be/pkg/exception"
	"zeus-be/pkg/response"
	"zeus-system-service/internal/models"
	"zeus-system-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AuditHandler struct {
	svc service.AuditService
}

func NewAuditHandler(svc service.AuditService) *AuditHandler {
	return &AuditHandler{svc: svc}
}

func (h *AuditHandler) Ingest(c *gin.Context) {
	var req models.IngestAuditRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		exception.WriteError(c, exception.ErrInvalidBody)
		return
	}

	if err := h.svc.Ingest(c.Request.Context(), req); err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal)
		return
	}

	response.OKWithMessage(c, 201, "log entry created")
}

func (h *AuditHandler) Query(c *gin.Context) {
	filter := models.AuditFilter{}

	if actionStr := c.Query("action_type"); actionStr != "" {
		at := models.ActionType(actionStr)
		filter.ActionType = &at
	}

	if userIDStr := c.Query("user_id"); userIDStr != "" {
		uid, err := uuid.Parse(userIDStr)
		if err == nil {
			filter.UserID = &uid
		}
	}

	if startStr := c.Query("start_date"); startStr != "" {
		t, err := time.Parse(time.RFC3339, startStr)
		if err == nil {
			filter.StartDate = &t
		}
	}

	if endStr := c.Query("end_date"); endStr != "" {
		t, err := time.Parse(time.RFC3339, endStr)
		if err == nil {
			filter.EndDate = &t
		}
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "15"))

	logs, meta, err := h.svc.Query(c.Request.Context(), filter, page, limit)
	if err != nil {
		exception.WriteError(c, exception.ErrInternal)
		return
	}

	response.OK(c, gin.H{
		"items":      logs,
		"pagination": meta,
	})
}

func (h *AuditHandler) GetMetrics(c *gin.Context) {
	metrics, err := h.svc.GetMetrics(c.Request.Context())
	if err != nil {
		exception.WriteError(c, exception.ErrInternal)
		return
	}

	response.OK(c, metrics)
}
