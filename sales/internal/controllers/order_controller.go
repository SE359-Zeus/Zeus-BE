package controllers

import (
	"net/http"
	"strings"
	"time"

	"zeus-sales-service/internal/models"
	"zeus-sales-service/internal/service"
)

type OrderController struct {
	svc *service.OrderService
}

func NewOrderController(svc *service.OrderService) *OrderController {
	return &OrderController{svc: svc}
}

func (controller *OrderController) HandleOrders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req models.CreateOrderRequest
		if err := readJSON(r, &req); err != nil {
			writeErrorJSON(w, http.StatusBadRequest, err.Error(), nil)
			return
		}
		order, err := controller.svc.CreateOrder(r.Context(), req)
		if err != nil {
			panic(err)
		}
		writeJSON(w, http.StatusCreated, order)
	case http.MethodGet:
		// parse optional filters: states (CSV of status codes), date (YYYY-MM-DD)
		q := r.URL.Query()
		var states []string
		if s := q.Get("states"); s != "" {
			for _, part := range strings.Split(s, ",") {
				code := strings.ToUpper(strings.TrimSpace(part))
				if code != "" {
					states = append(states, code)
				}
			}
		}
		var datePtr *time.Time
		if d := q.Get("date"); d != "" {
			if parsed, err := time.Parse("2006-01-02", d); err == nil {
				// use UTC midnight
				t := parsed.UTC()
				datePtr = &t
			}
		}
		orders, err := controller.svc.ListOrdersWithFilters(r.Context(), states, datePtr)
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, err.Error(), nil)
			return
		}
		writeJSON(w, http.StatusOK, orders)
	default:
		writeErrorJSON(w, http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed), nil)
	}
}

func (controller *OrderController) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	metrics, err := controller.svc.GetMetrics(r.Context())
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, metrics)
}

func (controller *OrderController) HandleOrderByID(w http.ResponseWriter, r *http.Request) {
	id, action, ok := parseIDAndAction(r.URL.Path, "/api/v1/sales/orders/")
	if !ok {
		writeErrorJSON(w, http.StatusBadRequest, "invalid order id", nil)
		return
	}
	// handle action-specific routes like POST /orders/{id}/reserve
	if action != "" {
		if action == "reserve" && r.Method == http.MethodPost {
			if err := controller.svc.ReserveInventory(r.Context(), id); err != nil {
				writeErrorJSON(w, http.StatusInternalServerError, err.Error(), nil)
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"message": "reserve triggered"})
			return
		}
		writeErrorJSON(w, http.StatusNotFound, "not found", nil)
		return
	}

	switch r.Method {
	case http.MethodGet:
		order, err := controller.svc.GetOrder(r.Context(), id)
		if err != nil {
			writeErrorJSON(w, http.StatusNotFound, err.Error(), nil)
			return
		}
		writeJSON(w, http.StatusOK, order)
	case http.MethodPatch:
		var req models.UpdateOrderRequest
		if err := readJSON(r, &req); err != nil {
			writeErrorJSON(w, http.StatusBadRequest, err.Error(), nil)
			return
		}
		order, err := controller.svc.UpdateOrder(r.Context(), id, req)
		if err != nil {
			panic(err)
		}
		writeJSON(w, http.StatusOK, order)
	default:
		writeErrorJSON(w, http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed), nil)
	}
}

func (controller *OrderController) HandleCancelOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed), nil)
		return
	}
	id, action, ok := parseIDAndAction(r.URL.Path, "/api/v1/sales/orders/")
	if !ok || action != "cancel" {
		writeErrorJSON(w, http.StatusBadRequest, "invalid order id", nil)
		return
	}
	if err := controller.svc.CancelOrder(r.Context(), id); err != nil {
		panic(err)
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "order cancelled"})
}
