package middleware

import (
	"log/slog"
	"strings"
	"zeus-scm-service/internal/exception"

	"github.com/gin-gonic/gin"
)

func RequireRoles(allowedRoles ...string) gin.HandlerFunc {
	normalizedRoles := make([]string, 0, len(allowedRoles))
	for _, role := range allowedRoles {
		role = strings.TrimSpace(role)
		if role == "" {
			continue
		}
		normalizedRoles = append(normalizedRoles, role)
	}

	return func(c *gin.Context) {
		if len(normalizedRoles) == 0 {
			slog.Warn("role authorization rejected",
				slog.String("service", "scm"),
				slog.String("event", "authorization_rejected"),
				slog.String("reason", "no_allowed_roles"),
				slog.String("method", c.Request.Method),
				slog.String("path", c.Request.URL.Path),
			)
			exception.WriteError(c, exception.ErrForbidden)
			return
		}

		role, exists := c.Get(ContextKeyRole)
		if !exists {
			slog.Warn("role authorization rejected",
				slog.String("service", "scm"),
				slog.String("event", "authorization_rejected"),
				slog.String("reason", "missing_role"),
				slog.String("method", c.Request.Method),
				slog.String("path", c.Request.URL.Path),
			)
			exception.WriteError(c, exception.ErrMissingRole)
			return
		}

		roleStr, ok := role.(string)
		if !ok {
			slog.Warn("role authorization rejected",
				slog.String("service", "scm"),
				slog.String("event", "authorization_rejected"),
				slog.String("reason", "invalid_role_type"),
				slog.String("method", c.Request.Method),
				slog.String("path", c.Request.URL.Path),
			)
			exception.WriteError(c, exception.ErrInternal)
			return
		}

		roleStr = strings.TrimSpace(roleStr)
		if strings.EqualFold(roleStr, "api_key") {
			c.Next()
			return
		}

		for _, allowedRole := range normalizedRoles {
			if strings.EqualFold(allowedRole, roleStr) {
				c.Next()
				return
			}
		}

		slog.Warn("role authorization rejected",
			slog.String("service", "scm"),
			slog.String("event", "authorization_rejected"),
			slog.String("reason", "insufficient_role"),
			slog.String("role", roleStr),
			slog.String("allowed_roles", strings.Join(normalizedRoles, ",")),
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
		)
		exception.WriteError(c, exception.ErrForbidden)
	}
}
