package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type SuccessResponse struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
	Data       any    `json:"data"`
}

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, SuccessResponse{
		StatusCode: http.StatusOK,
		Message:    "success",
		Data:       data,
	})
}

func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, SuccessResponse{
		StatusCode: http.StatusCreated,
		Message:    "created",
		Data:       data,
	})
}

func OKWithMessage(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, SuccessResponse{
		StatusCode: statusCode,
		Message:    message,
		Data:       nil,
	})
}
