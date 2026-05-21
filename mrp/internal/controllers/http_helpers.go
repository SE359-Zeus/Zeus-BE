package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"zeus-mrp-service/internal/middlewares"

	"github.com/google/uuid"
)

func writeJSON(w http.ResponseWriter, status int, payload any) {
	writeEnvelope(w, status, http.StatusText(status), nil, payload)
}

func writeErrorJSON(w http.ResponseWriter, status int, message string, metadata any) {
	writeEnvelope(w, status, message, metadata, nil)
}

func writeEnvelope(w http.ResponseWriter, status int, message string, metadata any, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(middlewares.ResponseEnvelope{
		Message:    message,
		StatusCode: status,
		Metadata:   metadata,
		Data:       data,
	})
}

func readJSON(r *http.Request, target any) error {
	return json.NewDecoder(r.Body).Decode(target)
}

func parseID(path string, prefix string) (uuid.UUID, bool) {
	idPart := strings.TrimPrefix(path, prefix)
	idPart = strings.Trim(idPart, "/")
	if idPart == "" {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(idPart)
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

func parseIDAndAction(path string, prefix string) (uuid.UUID, string, bool) {
	rest := strings.TrimPrefix(path, prefix)
	rest = strings.Trim(rest, "/")
	if rest == "" {
		return uuid.Nil, "", false
	}
	parts := strings.Split(rest, "/")
	idPart := parts[0]
	id, err := uuid.Parse(idPart)
	if err != nil {
		return uuid.Nil, "", false
	}
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}
	return id, action, true
}

// parsePaginationParams reads `page` and `per_page` query params with defaults.
func parsePaginationParams(r *http.Request) (int, int, error) {
	q := r.URL.Query()
	page := 1
	per := 20
	if raw := strings.TrimSpace(q.Get("page")); raw != "" {
		p, err := strconv.Atoi(raw)
		if err != nil || p <= 0 {
			return 0, 0, err
		}
		page = p
	}
	if raw := strings.TrimSpace(q.Get("per_page")); raw != "" {
		p, err := strconv.Atoi(raw)
		if err != nil || p <= 0 {
			return 0, 0, err
		}
		per = p
	}
	return page, per, nil
}

func paginateAny(items []any, page, per int) ([]any, map[string]any) {
	total := len(items)
	if per <= 0 {
		per = 20
	}
	if page <= 0 {
		page = 1
	}
	start := (page - 1) * per
	if start >= total {
		return []any{}, map[string]any{"page": page, "per_page": per, "total": total, "total_pages": (total + per - 1) / per}
	}
	end := start + per
	if end > total {
		end = total
	}
	meta := map[string]any{"page": page, "per_page": per, "total": total, "total_pages": (total + per - 1) / per}
	return items[start:end], meta
}
