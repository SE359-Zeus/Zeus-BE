package controllers

import (
	"encoding/json"
	"net/http"
	"strings"

	"zeus-sales-service/internal/middlewares"

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

// parseIDAndAction extracts a UUID id and an optional action suffix from a path.
// Example: /api/v1/sales/orders/{id}/reserve -> returns id, "reserve", true
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
