package controllers

import (
	"net/http"

	"zeus-mrp-service/internal/models"
)

// POST /api/v1/production/orders
func (c *ProductionController) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req models.CreateProductionOrderRequest
	if err := readJSON(r, &req); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid json payload", nil)
		return
	}
	res, err := c.svc.PlanProduction(r.Context(), req)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusCreated, res)
}
