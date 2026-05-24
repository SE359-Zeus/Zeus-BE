package controllers

import (
	"net/http"
	"strings"

	"zeus-sales-service/internal/models"
	"zeus-sales-service/internal/service"
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
		writeJSON(w, http.StatusOK, clients)
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
	default:
		writeErrorJSON(w, http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed), nil)
	}
}
