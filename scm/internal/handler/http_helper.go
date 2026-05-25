package handler

import (
	"net/http"
	"zeus-scm-service/internal/pagination"

	"github.com/gin-gonic/gin"
)

type ResponseEnvelope struct {
	Message    string `json:"message"`
	StatusCode int    `json:"statusCode"`
	Metadata   any    `json:"metadata"`
	Data       any    `json:"data"`
}

func writeEnvelope(c *gin.Context, status int, message string, metadata any, data any) {
	if metadata == nil {
		metadata = gin.H{}
	}
	c.JSON(status, ResponseEnvelope{
		Message:    message,
		StatusCode: status,
		Metadata:   metadata,
		Data:       data,
	})
}

func writeJSON(c *gin.Context, status int, data any) {
	if paginated, ok := data.(pagination.Response); ok {
		writeEnvelope(c, status, http.StatusText(status), gin.H{"pagination": paginated.Pagination}, paginated.Data)
		return
	}
	writeEnvelope(c, status, http.StatusText(status), gin.H{}, data)
}

func writeErrorJSON(c *gin.Context, status int, message string, metadata any) {
	writeEnvelope(c, status, message, metadata, nil)
}
