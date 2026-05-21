package controllers

import "net/http"

// GET /api/v1/mrp/inventory/ledger
func (c *ProductionController) GetInventoryLedger(w http.ResponseWriter, r *http.Request) {
	rows, err := c.svc.GetInventoryLedger(r.Context())
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, rows)
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
