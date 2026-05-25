package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"zeus-be/pkg/exception"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ResponseEnvelope struct {
	Message    string `json:"message"`
	StatusCode int    `json:"statusCode"`
	Metadata   any    `json:"metadata"`
	Data       any    `json:"data"`
}

func WriteJSON(c *gin.Context, status int, payload any) {
	WriteEnvelope(c, status, http.StatusText(status), gin.H{}, payload)
}

func WriteErrorJSON(c *gin.Context, status int, message string, metadata any) {
	WriteEnvelope(c, status, message, metadata, nil)
}

func WriteEnvelope(c *gin.Context, status int, message string, metadata any, data any) {
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

func WriteAppError(c *gin.Context, appErr *exception.AppError) {
	if appErr == nil {
		appErr = exception.ErrInternal
	}
	WriteErrorJSON(c, appErr.HTTPStatus, appErr.Message, gin.H{"code": appErr.Code})
}

func ReadJSON(r *http.Request, target any) error {
	return json.NewDecoder(r.Body).Decode(target)
}

func ParseID(path string, prefix string) (uuid.UUID, bool) {
	idPart := strings.TrimPrefix(path, prefix)
	idPart = strings.Trim(idPart, "/")
	if idPart == "" {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(idPart)
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

func ParseIDAndAction(path string, prefix string) (uuid.UUID, string, bool) {
	rest := strings.TrimPrefix(path, prefix)
	rest = strings.Trim(rest, "/")
	if rest == "" {
		return uuid.Nil, "", false
	}
	parts := strings.Split(rest, "/")
	idPart := parts[0]
	id, err := uuid.Parse(idPart)
	if err != nil {
		return uuid.Nil, "", false
	}
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}
	return id, action, true
}
