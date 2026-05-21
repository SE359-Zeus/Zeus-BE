package middleware

import (
	"log"

	"github.com/gin-gonic/gin"
	"zeus-be/pkg/exception"
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
