package middleware

import (
	"log"

	"zeus-system-service/internal/handler"

	"zeus-be/pkg/exception"

	"github.com/gin-gonic/gin"
)

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic recovered: %v", r)
				handler.WriteAppError(c, exception.ErrPanic)
			}
		}()
		c.Next()
	}
}
