package controllers

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
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
	summary, err := c.svc.GetDemandSummary(r.Context())
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	items := make([]any, 0, len(summary))
	for _, s := range summary {
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
