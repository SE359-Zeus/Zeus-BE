package handler

import (
	"strings"

	"zeus-be/pkg/exception"
	"zeus-system-service/internal/models"
	"zeus-system-service/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	svc service.AuthService
}

func NewAuthHandler(svc service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		exception.WriteError(c, exception.ErrInvalidBody)
		return
	}

	pair, err := h.svc.Login(c.Request.Context(), req)
	if err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal)
		return
	}

	c.JSON(200, gin.H{"access_token": pair.AccessToken})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req models.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		exception.WriteError(c, exception.ErrInvalidBody)
		return
	}

	pair, err := h.svc.Refresh(c.Request.Context(), req)
	if err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal)
		return
	}

	c.JSON(200, pair)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		exception.WriteError(c, exception.ErrMissingAuthHeader)
		return
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		exception.WriteError(c, exception.ErrInvalidAuthHeader)
		return
	}

	if err := h.svc.Logout(c.Request.Context(), parts[1]); err != nil {
		exception.WriteError(c, exception.ErrInvalidToken)
		return
	}

	c.JSON(200, gin.H{"message": "logged out successfully"})
}
