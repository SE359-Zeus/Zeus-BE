package controllers

import (
	"net/http"
	"strings"

	"zeus-sales-service/internal/middlewares"
	"zeus-sales-service/internal/models"
	"zeus-sales-service/internal/service"

	"github.com/google/uuid"
)

type ClientController struct {
	svc *service.ClientService
}

func NewClientController(svc *service.ClientService) *ClientController {
	return &ClientController{svc: svc}
}

func (controller *ClientController) HandleClients(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// support optional tiers filter: comma-separated
		q := r.URL.Query()
		page, pageSize := parsePagination(r)
		var tiers []string
		if t := q.Get("tiers"); t != "" {
			for _, part := range strings.Split(t, ",") {
				tier := strings.TrimSpace(part)
				if tier != "" {
					tiers = append(tiers, tier)
				}
			}
		}
		clients, err := controller.svc.ListClients(r.Context())
		if err != nil {
			panic(err)
		}
		if len(tiers) > 0 {
			filtered := make([]models.Client, 0, len(clients))
			m := make(map[string]struct{}, len(tiers))
			for _, tt := range tiers {
				m[strings.ToUpper(tt)] = struct{}{}
			}
			for _, c := range clients {
				if _, ok := m[strings.ToUpper(string(c.Tier))]; ok {
					filtered = append(filtered, c)
				}
			}
			clients = filtered
		}
		pageItems, pagination := paginateItems(clients, page, pageSize)
		writeEnvelope(w, http.StatusOK, "Clients listed", pagination, pageItems)
	case http.MethodPost:
		var req models.CreateClientRequest
		if err := readJSON(r, &req); err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "invalid JSON payload", nil)
			return
		}
		client, err := controller.svc.CreateClient(r.Context(), req)
		if err != nil {
			panic(err)
		}
		writeEnvelope(w, http.StatusCreated, "Client created", nil, client)
	default:
		writeErrorJSON(w, http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed), nil)
	}
}

func (controller *ClientController) HandleClientByID(w http.ResponseWriter, r *http.Request) {
	clientID, ok := parseID(r.URL.Path, "/api/v1/sales/clients/")
	if !ok {
		writeErrorJSON(w, http.StatusBadRequest, "invalid client id", nil)
		return
	}
	switch r.Method {
	case http.MethodGet:
		client, err := controller.svc.GetClient(r.Context(), clientID)
		if err != nil {
			panic(err)
		}
		writeJSON(w, http.StatusOK, client)
	case http.MethodPatch:
		var req models.UpdateClientRequest
		if err := readJSON(r, &req); err != nil {
			writeErrorJSON(w, http.StatusBadRequest, err.Error(), nil)
			return
		}
		client, err := controller.svc.UpdateClient(r.Context(), clientID, req)
		if err != nil {
			panic(err)
		}
		writeJSON(w, http.StatusOK, client)
	case http.MethodDelete:
		if err := controller.svc.DeleteClient(r.Context(), clientID); err != nil {
			panic(err)
		}
		writeEnvelope(w, http.StatusOK, "Client deleted", nil, nil)
	default:
		writeErrorJSON(w, http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed), nil)
	}
}

// PATCH /api/v1/sales/clients/me
// Clients update their own profile using API key authentication.
func (controller *ClientController) HandleUpdateMyProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeErrorJSON(w, http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed), nil)
		return
	}

	userIDVal := r.Context().Value(middlewares.ContextKeyUserID)
	if userIDVal == nil {
		writeErrorJSON(w, http.StatusUnauthorized, "unauthenticated", nil)
		return
	}

	var clientID uuid.UUID
	switch v := userIDVal.(type) {
	case string:
		id, err := uuid.Parse(v)
		if err != nil {
			writeErrorJSON(w, http.StatusUnauthorized, "invalid client identity", nil)
			return
		}
		clientID = id
	case uuid.UUID:
		clientID = v
	default:
		writeErrorJSON(w, http.StatusUnauthorized, "invalid client identity", nil)
		return
	}

	var req models.UpdateMyProfileRequest
	if err := readJSON(r, &req); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	if req.Name == nil && req.DefaultDestinationAddress == nil {
		writeErrorJSON(w, http.StatusBadRequest, "update request is empty", nil)
		return
	}

	updateReq := models.UpdateClientRequest{
		Name:                      req.Name,
		DefaultDestinationAddress: req.DefaultDestinationAddress,
	}

	client, err := controller.svc.UpdateClient(r.Context(), clientID, updateReq)
	if err != nil {
		panic(err)
	}

	writeJSON(w, http.StatusOK, client)
}
