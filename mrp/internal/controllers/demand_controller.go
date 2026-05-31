package controllers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"zeus-mrp-service/internal/models"
)

type demandPOGenerateResponse struct {
	Message string `json:"message"`
}

func parseDemandOrderID(r *http.Request) (uuid.UUID, error) {
	rawOrderID := r.PathValue("orderId")
	if rawOrderID == "" {
		return uuid.Nil, errors.New("orderId path parameter is required")
	}

	orderID, err := uuid.Parse(rawOrderID)
	if err != nil {
		return uuid.Nil, errors.New("orderId must be a valid UUID")
	}

	return orderID, nil
}

// GET /api/v1/mrp/demand/metrics
func (c *ProductionController) GetDemandMetrics(w http.ResponseWriter, r *http.Request) {
	m, err := c.svc.GetDemandMetrics(r.Context())
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, m)
}

// GET /api/v1/mrp/demand
func (c *ProductionController) GetDemandSummary(w http.ResponseWriter, r *http.Request) {
	page, per, err := parsePaginationParams(r)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid pagination params", nil)
		return
	}

	search := strings.TrimSpace(r.URL.Query().Get("search"))
	status := strings.TrimSpace(r.URL.Query().Get("status"))

	summary, err := c.svc.GetDemandSummary(r.Context())
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	// Filter by search and status
	var filtered []models.DemandPOSummary
	for _, item := range summary {
		match := true
		if search != "" {
			sLower := strings.ToLower(search)
			matchID := strings.Contains(strings.ToLower(item.OrderID), sLower)
			matchBuild := strings.Contains(strings.ToLower(item.TargetBuild), sLower)
			if !matchID && !matchBuild {
				match = false
			}
		}
		if status != "" {
			if !strings.EqualFold(item.Status, status) {
				match = false
			}
		}
		if match {
			filtered = append(filtered, item)
		}
	}

	items := make([]any, 0, len(filtered))
	for _, s := range filtered {
		items = append(items, s)
	}
	pageItems, meta := paginateAny(items, page, per)
	writeEnvelope(w, http.StatusOK, http.StatusText(http.StatusOK), meta, pageItems)
}

// POST /api/v1/mrp/demand/generate-pos
func (c *ProductionController) GeneratePOs(w http.ResponseWriter, r *http.Request) {
	if err := c.svc.GeneratePOsForShortages(r.Context()); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	writeJSON(w, http.StatusOK, demandPOGenerateResponse{Message: "PO generation started"})
}

// GET /api/v1/mrp/demand/{orderId}/pick-list
func (c *ProductionController) GetPickList(w http.ResponseWriter, r *http.Request) {
	orderID, err := parseDemandOrderID(r)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	pickList, err := c.svc.GeneratePickList(r.Context(), orderID)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	if pickList == nil {
		writeErrorJSON(w, http.StatusNotFound, "pick list not found", nil)
		return
	}

	writeJSON(w, http.StatusOK, pickList)
}

// POST /api/v1/mrp/demand/{orderId}/pick-list
func (c *ProductionController) GeneratePickList(w http.ResponseWriter, r *http.Request) {
	orderID, err := parseDemandOrderID(r)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	pickList, err := c.svc.GeneratePickList(r.Context(), orderID)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	if pickList == nil {
		writeErrorJSON(w, http.StatusNotFound, "pick list not found", nil)
		return
	}

	writeJSON(w, http.StatusCreated, pickList)
}

// DELETE /api/v1/mrp/demand/{orderId}
func (c *ProductionController) DeleteDemand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", nil)
		return
	}

	orderID, err := parseDemandOrderID(r)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	if err := c.svc.DeleteProductionOrder(r.Context(), orderID); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
