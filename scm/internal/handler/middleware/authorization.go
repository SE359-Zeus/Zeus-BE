package middleware

import (
	"zeus-scm-service/internal/exception"
	"zeus-scm-service/internal/service"

	"github.com/gin-gonic/gin"
)

func RequireRoleLevel(rbacSvc *service.RBACService) gin.HandlerFunc {
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

		allowed, err := rbacSvc.CanAccess(roleStr, method, path)
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
