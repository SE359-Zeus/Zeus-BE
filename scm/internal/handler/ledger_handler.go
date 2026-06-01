package handler

import (
	"zeus-scm-service/internal/exception"
	"zeus-scm-service/internal/pagination"
	"zeus-scm-service/internal/service"

	"github.com/gin-gonic/gin"
)

type LedgerHandler struct {
	svc service.ILedgerService
}

func NewLedgerHandler(svc service.ILedgerService) *LedgerHandler {
	return &LedgerHandler{svc: svc}
}

func (h *LedgerHandler) ListEntries(c *gin.Context) {
	params := parsePaginationParams(c)
	txnType := c.Query("type")
	sku := c.Query("sku")

	entries, meta, err := h.svc.ListEntries(c.Request.Context(), params, txnType, sku)
	if err != nil {
		exception.WriteError(c, exception.ErrInternal.WithError(err))
		return
	}
	writeJSON(c, 200, pagination.Response{Data: entries, Pagination: *meta})
}

func (h *LedgerHandler) GetEntryByID(c *gin.Context) {
	id := c.Param("id")
	entry, err := h.svc.GetEntryByID(c.Request.Context(), id)
	if err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrNotFound.WithMessage("Ledger entry not found"))
		return
	}
	if entry == nil {
		exception.WriteError(c, exception.ErrNotFound.WithMessage("Ledger entry not found"))
		return
	}
	writeJSON(c, 200, entry)
}
