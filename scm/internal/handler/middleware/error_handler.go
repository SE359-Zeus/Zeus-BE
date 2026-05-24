package middleware

import (
	"log"

	"zeus-scm-service/internal/exception"

	"github.com/gin-gonic/gin"
)

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic recovered: %v", r)
				exception.WriteError(c, exception.ErrPanic)
			}
		}()
		c.Next()
	}
}
