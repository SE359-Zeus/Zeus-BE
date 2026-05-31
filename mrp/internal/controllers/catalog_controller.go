package controllers

import (
	"net/http"

	"zeus-mrp-service/internal/models"

	"github.com/google/uuid"
)

// GET /api/v1/mrp/assemblies
func (c *ProductionController) GetAssemblies(w http.ResponseWriter, r *http.Request) {
	page, per, err := parsePaginationParams(r)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid pagination params", nil)
		return
	}

	res, total, err := c.svc.GetAssembliesPage(r.Context(), page, per)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	meta := map[string]any{
		"page":        page,
		"per_page":    per,
		"total":       total,
		"total_pages": (total + per - 1) / per,
	}
	writeEnvelope(w, http.StatusOK, http.StatusText(http.StatusOK), meta, res)
}

// POST /api/v1/mrp/assemblies
func (c *ProductionController) CreateAssembly(w http.ResponseWriter, r *http.Request) {
	var req models.CreateAssemblyRequest
	if err := readJSON(r, &req); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid json payload", nil)
		return
	}
	created, err := c.svc.CreateAssembly(r.Context(), req)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// GET /api/v1/mrp/assemblies/{id}
func (c *ProductionController) GetAssemblyDetail(w http.ResponseWriter, r *http.Request) {
	modelCode := r.PathValue("id")
	if modelCode == "" {
		writeErrorJSON(w, http.StatusBadRequest, "id path parameter is required", nil)
		return
	}
	assembly, err := c.svc.GetAssemblyByModelCode(r.Context(), modelCode)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	if assembly == nil {
		writeErrorJSON(w, http.StatusNotFound, "assembly not found", nil)
		return
	}
	writeJSON(w, http.StatusOK, assembly)
}

// PUT /api/v1/mrp/assemblies/{id}
func (c *ProductionController) UpdateAssembly(w http.ResponseWriter, r *http.Request) {
	rawID := r.PathValue("id")
	if rawID == "" {
		writeErrorJSON(w, http.StatusBadRequest, "id path parameter is required", nil)
		return
	}
	id, err := uuid.Parse(rawID)
	if err != nil {
		// treat as model code string if not a UUID
		id = uuid.Nil
	}
	var req models.UpdateAssemblyRequest
	if err := readJSON(r, &req); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid json payload", nil)
		return
	}
	// If id is Nil and the user passed a model-code string, put it in Name
	if id == uuid.Nil && req.Name == "" {
		req.Name = rawID
	}
	updated, err := c.svc.UpdateAssembly(r.Context(), id, req)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// DELETE /api/v1/mrp/assemblies/{id}
func (c *ProductionController) DeleteAssembly(w http.ResponseWriter, r *http.Request) {
	rawID := r.PathValue("id")
	if rawID == "" {
		writeErrorJSON(w, http.StatusBadRequest, "id path parameter is required", nil)
		return
	}
	if err := c.svc.DeleteAssembly(r.Context(), rawID); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

// GET /api/v1/mrp/catalog
func (c *ProductionController) GetCatalog(w http.ResponseWriter, r *http.Request) {
	page, per, err := parsePaginationParams(r)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid pagination params", nil)
		return
	}
	res, err := c.svc.GetCatalog(r.Context())
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	items := make([]any, 0, len(res))
	for _, v := range res {
		items = append(items, v)
	}
	pageItems, meta := paginateAny(items, page, per)
	writeEnvelope(w, http.StatusOK, http.StatusText(http.StatusOK), meta, pageItems)
}

// GET /api/v1/mrp/catalog/{sku}/where-used
func (c *ProductionController) GetWhereUsed(w http.ResponseWriter, r *http.Request) {
	sku := r.PathValue("sku")
	if sku == "" {
		writeErrorJSON(w, http.StatusBadRequest, "sku path parameter is required", nil)
		return
	}
	res, err := c.svc.GetWhereUsed(r.Context(), sku)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

type CreatePartRequest struct {
	SKU         string  `json:"sku"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
}

type UpdatePartRequest struct {
	Description string  `json:"description"`
	Price       float64 `json:"price"`
}

// POST /api/v1/mrp/catalog
func (c *ProductionController) CreateCatalogPart(w http.ResponseWriter, r *http.Request) {
	var req CreatePartRequest
	if err := readJSON(r, &req); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid json payload", nil)
		return
	}

	part, err := c.svc.CreateCatalogPart(r.Context(), req.SKU, req.Description, req.Price)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	writeEnvelope(w, http.StatusCreated, http.StatusText(http.StatusCreated), nil, part)
}

// PUT /api/v1/mrp/catalog/{sku}
func (c *ProductionController) UpdateCatalogPart(w http.ResponseWriter, r *http.Request) {
	sku := r.PathValue("sku")
	if sku == "" {
		writeErrorJSON(w, http.StatusBadRequest, "sku path parameter is required", nil)
		return
	}

	var req UpdatePartRequest
	if err := readJSON(r, &req); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid json payload", nil)
		return
	}

	part, err := c.svc.UpdateCatalogPart(r.Context(), sku, req.Description, req.Price)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	writeEnvelope(w, http.StatusOK, http.StatusText(http.StatusOK), nil, part)
}

// DELETE /api/v1/mrp/catalog/{sku}
func (c *ProductionController) DeleteCatalogPart(w http.ResponseWriter, r *http.Request) {
	sku := r.PathValue("sku")
	if sku == "" {
		writeErrorJSON(w, http.StatusBadRequest, "sku path parameter is required", nil)
		return
	}

	if err := c.svc.DeleteCatalogPart(r.Context(), sku); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	writeJSON(w, http.StatusNoContent, nil)
}
