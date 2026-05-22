package handler

import (
	"strconv"

	"zeus-be/pkg/exception"
	"zeus-be/pkg/response"
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
		exception.WriteError(c, exception.ErrInvalidBody)
		return
	}

	user, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal)
		return
	}

	response.Created(c, models.ToUserResponse(user))
}

func (h *UserHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		exception.WriteError(c, exception.ErrInvalidResourceID)
		return
	}

	user, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal)
		return
	}

	response.OK(c, models.ToUserResponse(user))
}

func (h *UserHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "15"))
	q := c.Query("q")

	users, meta, err := h.svc.List(c.Request.Context(), page, limit, q)
	if err != nil {
		exception.WriteError(c, exception.ErrInternal)
		return
	}

	resp := make([]models.UserResponse, len(users))
	for i, u := range users {
		resp[i] = models.ToUserResponse(&u)
	}
	response.OK(c, gin.H{
		"items":      resp,
		"pagination": meta,
	})
}

func (h *UserHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		exception.WriteError(c, exception.ErrInvalidResourceID)
		return
	}

	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		exception.WriteError(c, exception.ErrInvalidBody)
		return
	}

	user, err := h.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal)
		return
	}

	response.OK(c, models.ToUserResponse(user))
}

func (h *UserHandler) SetStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		exception.WriteError(c, exception.ErrInvalidResourceID)
		return
	}

	var req struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		exception.WriteError(c, exception.ErrInvalidBody)
		return
	}

	status := models.AccountStatus(req.Status)
	if status != models.AccountStatusActive && status != models.AccountStatusInactive {
		exception.WriteError(c, exception.ErrInvalidInput.WithMessage("status must be ACTIVE or INACTIVE"))
		return
	}

	if err := h.svc.SetStatus(c.Request.Context(), id, status); err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal)
		return
	}

	response.OKWithMessage(c, 200, "status updated")
}
