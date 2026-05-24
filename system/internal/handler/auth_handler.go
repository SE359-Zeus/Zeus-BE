package handler

import (
	"log"
	"strings"

	"zeus-be/pkg/exception"
	"zeus-system-service/internal/models"
	"zeus-system-service/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authSvc  service.AuthService
	auditSvc service.AuditService
}

func NewAuthHandler(authSvc service.AuthService, auditSvc ...service.AuditService) *AuthHandler {
	h := &AuthHandler{authSvc: authSvc}
	if len(auditSvc) > 0 {
		h.auditSvc = auditSvc[0]
	}
	return h
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteAppError(c, exception.ErrInvalidBody)
		return
	}

	pair, err := h.authSvc.Login(c.Request.Context(), req)
	if err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			WriteAppError(c, appErr)
			return
		}
		WriteAppError(c, exception.ErrInternal)
		return
	}

	if h.auditSvc != nil {
		if claims, err := h.authSvc.VerifyAccessToken(pair.AccessToken); err == nil {
			if err := h.auditSvc.Ingest(c.Request.Context(), models.IngestAuditRequest{
				UserID:         claims.UserID,
				UserEmail:      claims.Email,
				ActionType:     models.ActionType("LOGIN"),
				TargetResource: "auth/login",
				Details:        "Successful login",
				IPAddress:      c.ClientIP(),
			}); err != nil {
				log.Printf("warning: failed to record login audit event: %v", err)
			}
		} else {
			log.Printf("warning: failed to verify access token for login audit: %v", err)
		}
	}

	WriteJSON(c, 200, gin.H{
		"access_token":  pair.AccessToken,
		"refresh_token": pair.RefreshToken,
	})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req models.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteAppError(c, exception.ErrInvalidBody)
		return
	}

	pair, err := h.authSvc.Refresh(c.Request.Context(), req)
	if err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			WriteAppError(c, appErr)
			return
		}
		WriteAppError(c, exception.ErrInternal)
		return
	}

	WriteJSON(c, 200, pair)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		WriteAppError(c, exception.ErrMissingAuthHeader)
		return
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		WriteAppError(c, exception.ErrInvalidAuthHeader)
		return
	}

	if err := h.authSvc.Logout(c.Request.Context(), parts[1]); err != nil {
		WriteAppError(c, exception.ErrInvalidToken)
		return
	}

	WriteEnvelope(c, 200, "logged out successfully", gin.H{}, nil)
}
