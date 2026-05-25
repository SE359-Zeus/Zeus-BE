package exception

import (
	"errors"

	"github.com/gin-gonic/gin"
)

func WriteError(c *gin.Context, appErr *AppError) {
	c.AbortWithStatusJSON(appErr.HTTPStatus, gin.H{
		"error_code": appErr.Code,
		"message":    appErr.Message,
	})
}

func Resolve(err error) *AppError {
	if err == nil {
		return nil
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return nil
}

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				WriteError(c, ErrPanic)
			}
		}()
		c.Next()
	}
}
