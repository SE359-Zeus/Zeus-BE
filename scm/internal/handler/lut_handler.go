package handler

import (
	"zeus-scm-service/internal/exception"
	"zeus-scm-service/internal/service"

	"github.com/gin-gonic/gin"
)

type LUTHandler struct {
	svc service.ILUTService
}

func NewLUTHandler(svc service.ILUTService) *LUTHandler {
	return &LUTHandler{svc: svc}
}

func (h *LUTHandler) GetAllLUTs(c *gin.Context) {
	luts, err := h.svc.GetAllLUTs(c.Request.Context())
	if err != nil {
		exception.WriteError(c, exception.ErrInternal.WithError(err))
		return
	}
	writeJSON(c, 200, luts)
}
