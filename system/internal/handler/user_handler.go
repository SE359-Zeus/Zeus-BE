package handler

import (
	"strconv"

	"zeus-be/pkg/exception"
	"zeus-system-service/internal/models"
	"zeus-system-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UserHandler struct {
	svc service.UserService
}

func NewUserHandler(svc service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
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
	WriteJSON(c, 200, gin.H{
		"items":      resp,
		"pagination": meta,
	})
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

	user, err := h.svc.Update(c.Request.Context(), id, req)
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
