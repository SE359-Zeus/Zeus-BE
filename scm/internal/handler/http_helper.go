package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ResponseEnvelope struct {
	Message    string `json:"message"`
	StatusCode int    `json:"statusCode"`
	Metadata   any    `json:"metadata,omitempty"`
	Data       any    `json:"data,omitempty"`
}

func writeEnvelope(c *gin.Context, status int, message string, metadata any, data any) {
	c.JSON(status, ResponseEnvelope{
		Message:    message,
		StatusCode: status,
		Metadata:   metadata,
		Data:       data,
	})
}

func writeJSON(c *gin.Context, status int, data any) {
	writeEnvelope(c, status, http.StatusText(status), nil, data)
}

func writeErrorJSON(c *gin.Context, status int, message string, metadata any) {
	writeEnvelope(c, status, message, metadata, nil)
}
