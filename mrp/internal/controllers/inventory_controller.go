package controllers

import (
	"net/http"
	"strings"
	"zeus-mrp-service/internal/models"
)

// GET /api/v1/mrp/inventory/ledger
func (c *ProductionController) GetInventoryLedger(w http.ResponseWriter, r *http.Request) {
	page, per, err := parsePaginationParams(r)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid pagination params", nil)
		return
	}

	filterType := strings.TrimSpace(r.URL.Query().Get("type"))

	rows, err := c.svc.GetInventoryLedger(r.Context())
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	// Filter rows by type
	var filtered []models.InventoryTransactionDTO
	if filterType == "" || strings.EqualFold(filterType, "All") {
		filtered = rows
	} else {
		for _, row := range rows {
			match := false
			t := strings.ToLower(row.Type)
			ft := strings.ToLower(filterType)
			if ft == "stock in" || ft == "stock_in" {
				match = (t == "stock_in" || t == "stock in")
			} else if ft == "stock out" || ft == "stock_out" {
				match = (t == "stock_out" || t == "stock out")
			} else if ft == "adjustments" || ft == "adjustment" {
				match = (t == "adjustment" || t == "adjustments" || t == "adj")
			} else {
				match = strings.Contains(t, ft)
			}
			if match {
				filtered = append(filtered, row)
			}
		}
	}

	items := make([]any, 0, len(filtered))
	for _, v := range filtered {
		items = append(items, v)
	}
	pageItems, meta := paginateAny(items, page, per)
	writeEnvelope(w, http.StatusOK, http.StatusText(http.StatusOK), meta, pageItems)
}

// GET /api/v1/mrp/inventory/metrics
func (c *ProductionController) GetInventoryMetrics(w http.ResponseWriter, r *http.Request) {
	m, err := c.svc.GetInventoryMetrics(r.Context())
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, m)
}

// GET /api/v1/mrp/inventory/ledger/export
func (c *ProductionController) ExportInventoryCSV(w http.ResponseWriter, r *http.Request) {
	data, err := c.svc.ExportInventoryCSV(r.Context())
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="inventory_ledger.csv"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// GET /api/v1/mrp/inventory/transactions/{txnId}
func (c *ProductionController) GetInventoryTransactionByID(w http.ResponseWriter, r *http.Request) {
	raw := r.PathValue("txnId")
	if raw == "" {
		writeErrorJSON(w, http.StatusBadRequest, "txnId is required", nil)
		return
	}
	tx, err := c.svc.GetInventoryTransactionByID(r.Context(), raw)
	if err != nil {
		writeErrorJSON(w, http.StatusNotFound, err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, tx)
}

// GET /api/v1/mrp/inventory/balance/{sku}
func (c *ProductionController) GetInventoryBalanceBySKU(w http.ResponseWriter, r *http.Request) {
	sku := r.PathValue("sku")
	if sku == "" {
		writeErrorJSON(w, http.StatusBadRequest, "sku is required", nil)
		return
	}
	bal, err := c.svc.GetInventoryBalanceBySKU(r.Context(), sku)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sku": sku, "balance": bal})
}
