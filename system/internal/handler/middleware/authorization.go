package middleware

import (
	"zeus-be/pkg/exception"
	"zeus-system-service/internal/service"

	"github.com/gin-gonic/gin"
)

func RequireRoleLevel(rbacSvc service.EndpointRBACService) gin.HandlerFunc {
	return func(c *gin.Context) {
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

		method := c.Request.Method
		path := c.FullPath()

		allowed, err := rbacSvc.CanAccess(c.Request.Context(), roleStr, method, path)
		if err != nil {
			exception.WriteError(c, exception.ErrAccessCheck.WithError(err))
			return
		}
		if !allowed {
			exception.WriteError(c, exception.ErrForbidden)
			return
		}

		c.Next()
	}
}

func RequireEndpointAccess(rbacSvc service.EndpointRBACService) gin.HandlerFunc {
	return RequireRoleLevel(rbacSvc)
}
