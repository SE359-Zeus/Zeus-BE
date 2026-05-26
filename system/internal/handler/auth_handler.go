package handler

import (
	"log"
	"net/http"
	"strings"
	"time"

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

// setRefreshCookie writes the refresh token as a regular cookie.
func setRefreshCookie(c *gin.Context, token string) {
	c.SetSameSite(http.SameSiteNoneMode)
	c.SetCookie(
		models.RefreshTokenCookieName,
		token,
		int(models.RefreshTokenDuration/time.Second),
		"/",
		"localhost", // domain — empty = same host
		false,       // Secure (HTTPS only)
		false,       // HttpOnly
	)
}

// clearRefreshCookie removes the refresh token cookie on logout.
func clearRefreshCookie(c *gin.Context) {
	c.SetCookie(
		models.RefreshTokenCookieName,
		"",
		-1,
		"/",
		"",
		true,
		false,
	)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteAppError(c, exception.ErrInvalidBody)
		return
	}

	result, err := h.authSvc.Login(c.Request.Context(), req)
	if err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			WriteAppError(c, appErr)
			return
		}
		WriteAppError(c, exception.ErrInternal)
		return
	}

	if h.auditSvc != nil {
		if claims, err := h.authSvc.VerifyAccessToken(result.Tokens.AccessToken); err == nil {
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

	setRefreshCookie(c, result.Tokens.RefreshToken)

	WriteJSON(c, 200, models.AuthResponse{
		AccessToken: result.Tokens.AccessToken,
		TokenType:   result.Tokens.TokenType,
		ExpiresIn:   result.Tokens.ExpiresIn,
		User:        models.ToUserResponse(result.User),
	})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	refreshToken := resolveRefreshToken(c)
	if refreshToken == "" {
		WriteAppError(c, exception.ErrInvalidBody)
		return
	}

	result, err := h.authSvc.Refresh(c.Request.Context(), models.RefreshRequest{
		RefreshToken: refreshToken,
	})
	if err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			WriteAppError(c, appErr)
			return
		}
		WriteAppError(c, exception.ErrInternal)
		return
	}

	setRefreshCookie(c, result.Tokens.RefreshToken)

	WriteJSON(c, 200, models.AuthResponse{
		AccessToken: result.Tokens.AccessToken,
		TokenType:   result.Tokens.TokenType,
		ExpiresIn:   result.Tokens.ExpiresIn,
		User:        models.ToUserResponse(result.User),
	})
}

func resolveRefreshToken(c *gin.Context) string {
	var body models.RefreshRequest
	if c.Request != nil && c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&body); err == nil {
			if token := strings.TrimSpace(body.RefreshToken); token != "" {
				return token
			}
		} else if token, err := c.Cookie(models.RefreshTokenCookieName); err == nil && token != "" {
			return token
		}
	}

	if token, err := c.Cookie(models.RefreshTokenCookieName); err == nil {
		return strings.TrimSpace(token)
	}

	return ""
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

	clearRefreshCookie(c)
	WriteEnvelope(c, 200, "logged out successfully", gin.H{}, nil)
}
