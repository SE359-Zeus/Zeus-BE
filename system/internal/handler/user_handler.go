package handler

import (
	"fmt"
	"log"
	"strconv"

	"zeus-be/pkg/exception"
	"zeus-system-service/internal/infrastructure/observability"
	"zeus-system-service/internal/models"
	"zeus-system-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UserHandler struct {
	svc      service.UserService
	auditSvc service.AuditService
}

func NewUserHandler(svc service.UserService, auditSvc ...service.AuditService) *UserHandler {
	h := &UserHandler{svc: svc}
	if len(auditSvc) > 0 {
		h.auditSvc = auditSvc[0]
	}
	return h
}

func (h *UserHandler) Create(c *gin.Context) {
	var req models.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteAppError(c, exception.ErrInvalidBody)
		return
	}

	user, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			WriteAppError(c, appErr)
			return
		}
		WriteAppError(c, exception.ErrInternal)
		return
	}

	if h.auditSvc != nil {
		userID, okID := c.Get("user_id")
		userEmail, okEmail := c.Get("email")
		if okID && okEmail {
			uid, _ := userID.(uuid.UUID)
			email, _ := userEmail.(string)
			if err := h.auditSvc.Ingest(c.Request.Context(), models.IngestAuditRequest{
				UserID:         uid,
				UserEmail:      email,
				ActionType:     models.ActionType("CREATE"),
				TargetResource: "users/" + user.ID.String(),
				Details:        fmt.Sprintf("Created user: %s (Role: %s)", user.Email, user.Role),
				IPAddress:      c.ClientIP(),
			}); err != nil {
				log.Printf("warning: failed to record user creation audit event: %v", err)
			}
		} else {
			log.Printf("warning: missing context keys for user creation audit: id_ok=%t, email_ok=%t", okID, okEmail)
		}
	}

	observability.DefaultRegistry.Counter(observability.MetricUserCreatedTotal).Inc()
	WriteEnvelope(c, 201, "created", gin.H{}, models.ToUserResponse(user))
}

func (h *UserHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		WriteAppError(c, exception.ErrInvalidResourceID)
		return
	}

	user, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			WriteAppError(c, appErr)
			return
		}
		WriteAppError(c, exception.ErrInternal)
		return
	}

	WriteJSON(c, 200, models.ToUserResponse(user))
}

func (h *UserHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "15"))
	q := c.Query("q")

	users, meta, err := h.svc.List(c.Request.Context(), page, limit, q)
	if err != nil {
		WriteAppError(c, exception.ErrInternal)
		return
	}

	resp := make([]models.UserResponse, len(users))
	for i, u := range users {
		resp[i] = models.ToUserResponse(&u)
	}
	WriteEnvelope(c, 200, "success", gin.H{"pagination": meta}, gin.H{"items": resp})
}

func (h *UserHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		WriteAppError(c, exception.ErrInvalidResourceID)
		return
	}

	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteAppError(c, exception.ErrInvalidBody)
		return
	}

	currentRole, _ := c.Get("role")
	currentUserID, _ := c.Get("user_id")
	role, _ := currentRole.(string)
	var uid uuid.UUID
	if v, ok := currentUserID.(uuid.UUID); ok {
		uid = v
	}

	user, err := h.svc.Update(c.Request.Context(), id, req, role, uid)
	if err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			WriteAppError(c, appErr)
			return
		}
		WriteAppError(c, exception.ErrInternal)
		return
	}

	WriteJSON(c, 200, models.ToUserResponse(user))
}

func (h *UserHandler) SetStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		WriteAppError(c, exception.ErrInvalidResourceID)
		return
	}

	var req struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteAppError(c, exception.ErrInvalidBody)
		return
	}

	status := models.AccountStatus(req.Status)
	if status != models.AccountStatusActive && status != models.AccountStatusInactive {
		WriteAppError(c, exception.ErrInvalidInput.WithMessage("status must be ACTIVE or INACTIVE"))
		return
	}

	if err := h.svc.SetStatus(c.Request.Context(), id, status); err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			WriteAppError(c, appErr)
			return
		}
		WriteAppError(c, exception.ErrInternal)
		return
	}

	WriteEnvelope(c, 200, "status updated", gin.H{}, nil)
}
