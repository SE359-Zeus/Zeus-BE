package controllers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"zeus-mrp-service/internal/models"

	"github.com/google/uuid"
)

type readinessPOResponse struct {
	OrderID  uuid.UUID                   `json:"order_id"`
	Deficits []models.BOMExplosionResult `json:"deficits"`
	Message  string                      `json:"message"`
}

func readinessHTTPStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}

	message := strings.ToLower(err.Error())
	if strings.Contains(message, "must be") || strings.Contains(message, "invalid") || strings.Contains(message, "required") {
		return http.StatusBadRequest
	}
	if strings.Contains(message, "not found") {
		return http.StatusNotFound
	}
	return http.StatusInternalServerError
}

func parseReadinessOrderID(r *http.Request) (uuid.UUID, error) {
	rawOrderID := strings.TrimSpace(r.PathValue("orderId"))
	if rawOrderID == "" {
		return uuid.Nil, errors.New("orderId path parameter is required")
	}

	orderID, err := uuid.Parse(rawOrderID)
	if err != nil {
		return uuid.Nil, errors.New("orderId must be a valid UUID")
	}

	return orderID, nil
}

func parseReadinessMatrixRequest(r *http.Request) (models.ReadinessFilter, models.PaginationParams, error) {
	query := r.URL.Query()

	page := 1
	if rawPage := strings.TrimSpace(query.Get("page")); rawPage != "" {
		parsedPage, err := strconv.Atoi(rawPage)
		if err != nil {
			return models.ReadinessFilter{}, models.PaginationParams{}, errors.New("page must be a valid integer")
		}
		page = parsedPage
	}

	perPage := 20
	if rawPerPage := strings.TrimSpace(query.Get("per_page")); rawPerPage != "" {
		parsedPerPage, err := strconv.Atoi(rawPerPage)
		if err != nil {
			return models.ReadinessFilter{}, models.PaginationParams{}, errors.New("per_page must be a valid integer")
		}
		perPage = parsedPerPage
	}

	return models.ReadinessFilter{
		Status: strings.TrimSpace(query.Get("status")),
		Search: strings.TrimSpace(query.Get("search")),
	}, models.PaginationParams{Page: page, PerPage: perPage}, nil
}

// GET /api/v1/mrp/readiness
func (c *ProductionController) GetReadinessMatrix(w http.ResponseWriter, r *http.Request) {
	filter, page, err := parseReadinessMatrixRequest(r)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	rows, total, err := c.svc.GetReadinessMatrix(r.Context(), filter, page)
	if err != nil {
		writeErrorJSON(w, readinessHTTPStatus(err), err.Error(), nil)
		return
	}

	totalPages := 0
	if page.PerPage > 0 {
		totalPages = (total + page.PerPage - 1) / page.PerPage
	}

	meta := map[string]any{
		"page":        page.Page,
		"per_page":    page.PerPage,
		"total":       total,
		"total_pages": totalPages,
	}

	writeEnvelope(w, http.StatusOK, http.StatusText(http.StatusOK), meta, rows)
}

// GET /api/v1/mrp/readiness/metrics
func (c *ProductionController) GetReadinessMetrics(w http.ResponseWriter, r *http.Request) {
	metrics, err := c.svc.GetReadinessMetrics(r.Context())
	if err != nil {
		writeErrorJSON(w, readinessHTTPStatus(err), err.Error(), nil)
		return
	}

	writeJSON(w, http.StatusOK, metrics)
}

// GET /api/v1/mrp/readiness/export
func (c *ProductionController) ExportReport(w http.ResponseWriter, r *http.Request) {
	data, err := c.svc.ExportReadinessReport(r.Context())
	if err != nil {
		writeErrorJSON(w, readinessHTTPStatus(err), err.Error(), nil)
		return
	}

	if data == nil || len(data) == 0 {
		writeErrorJSON(w, http.StatusNotFound, "no report available", nil)
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="readiness_report.csv"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// GET /api/v1/mrp/shortages
func (c *ProductionController) GetShortages(w http.ResponseWriter, r *http.Request) {
	page, per, err := parsePaginationParams(r)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid pagination params", nil)
		return
	}

	shortages, err := c.svc.GetAggregatedDemand(r.Context())
	if err != nil {
		writeErrorJSON(w, readinessHTTPStatus(err), err.Error(), nil)
		return
	}
	items := make([]any, 0, len(shortages))
	for _, s := range shortages {
		items = append(items, s)
	}
	pageItems, meta := paginateAny(items, page, per)
	writeEnvelope(w, http.StatusOK, http.StatusText(http.StatusOK), meta, pageItems)
}

// GET /api/v1/mrp/readiness/{orderId}
func (c *ProductionController) GetReadinessByOrderID(w http.ResponseWriter, r *http.Request) {
	orderID, err := parseReadinessOrderID(r)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	row, err := c.svc.GetReadinessByOrderID(r.Context(), orderID)
	if err != nil {
		writeErrorJSON(w, readinessHTTPStatus(err), err.Error(), nil)
		return
	}
	if row == nil {
		writeErrorJSON(w, http.StatusNotFound, "production order not found", nil)
		return
	}

	writeJSON(w, http.StatusOK, row)
}

// POST /api/v1/mrp/readiness/{orderId}/generate-po
func (c *ProductionController) GeneratePOForDeficits(w http.ResponseWriter, r *http.Request) {
	orderID, err := parseReadinessOrderID(r)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	deficits, err := c.svc.GeneratePOForDeficits(r.Context(), orderID)
	if err != nil {
		writeErrorJSON(w, readinessHTTPStatus(err), err.Error(), nil)
		return
	}
	if deficits == nil {
		writeErrorJSON(w, http.StatusNotFound, "production order not found", nil)
		return
	}

	writeJSON(w, http.StatusCreated, readinessPOResponse{
		OrderID:  orderID,
		Deficits: deficits,
		Message:  "purchase order draft generated",
	})
}
