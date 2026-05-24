package middleware

import (
	"strings"

	"zeus-be/pkg/exception"
	"zeus-system-service/internal/service"

	"github.com/gin-gonic/gin"
)

const (
	ContextKeyUserID   = "user_id"
	ContextKeyRole     = "role"
	ContextKeyEmail    = "email"
	ContextKeyFullName = "full_name"
	ContextKeyStatus   = "status"
)

func JWTAuth(authSvc service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
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

		claims, err := authSvc.VerifyAccessToken(parts[1])
		if err != nil {
			exception.WriteError(c, exception.ErrInvalidToken)
			return
		}

		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyRole, claims.Role)
		c.Set(ContextKeyEmail, claims.Email)
		c.Set(ContextKeyFullName, claims.FullName)
		c.Set(ContextKeyStatus, claims.Status)
		c.Next()
	}
}
