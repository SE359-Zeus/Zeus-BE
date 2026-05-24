package middleware

import (
	"strings"

	"zeus-be/pkg/exception"

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
			exception.WriteError(c, exception.ErrForbidden)
			return
		}

		role, exists := c.Get(ContextKeyRole)
		if !exists {
			exception.WriteError(c, exception.ErrMissingRole)
			return
		}

		roleStr, ok := role.(string)
		if !ok {
			exception.WriteError(c, exception.ErrInternal)
			return
		}

		roleStr = strings.TrimSpace(roleStr)
		for _, allowedRole := range normalizedRoles {
			if strings.EqualFold(allowedRole, roleStr) {
				c.Next()
				return
			}
		}

		exception.WriteError(c, exception.ErrForbidden)
	}
}
