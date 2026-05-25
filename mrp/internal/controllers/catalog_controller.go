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

	res, err := c.svc.GetAssemblies(r.Context())
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
	id, err := uuid.Parse(rawID)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid id", nil)
		return
	}
	if err := c.svc.DeleteAssembly(r.Context(), id); err != nil {
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
