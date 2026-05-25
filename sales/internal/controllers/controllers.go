package controllers

import (
	"net/http"

	"zeus-sales-service/internal/middlewares"
	"zeus-sales-service/internal/service"
)

func NewMux(services *service.Services, authVerifier middlewares.TokenVerifier) http.Handler {
	mux := http.NewServeMux()
	orderController := NewOrderController(services.Orders)
	clientController := NewClientController(services.Clients)
	fulfillmentController := NewFulfillmentController(services.Fulfillment)
	authenticate := middlewares.Authenticate(authVerifier)
	protect := func(handler http.Handler, methodRoles map[string][]string) http.Handler {
		return middlewares.Chain(handler, authenticate, middlewares.RequireMethodRoles(methodRoles))
	}

	mux.Handle("/api/v1/sales/orders", protect(http.HandlerFunc(orderController.HandleOrders), map[string][]string{
		http.MethodGet:  {"sales_operator", "sales_worker", "admin"},
		http.MethodPost: {"sales_operator", "admin"},
	}))
	mux.Handle("/api/v1/sales/orders/", protect(http.HandlerFunc(orderController.HandleOrderByID), map[string][]string{
		http.MethodGet:   {"sales_operator", "sales_worker", "admin"},
		http.MethodPatch: {"sales_operator", "admin"},
	}))
	mux.Handle("/api/v1/sales/orders/:id", protect(http.HandlerFunc(orderController.HandleOrderByID), map[string][]string{
		http.MethodGet:   {"sales_operator", "sales_worker", "admin"},
		http.MethodPatch: {"sales_operator", "admin"},
	}))
	mux.Handle("/api/v1/sales/orders/{id}/cancel", protect(http.HandlerFunc(orderController.HandleCancelOrder), map[string][]string{
		http.MethodPost: {"sales_operator", "admin"},
	}))
	mux.Handle("/api/v1/sales/clients", protect(http.HandlerFunc(clientController.HandleClients), map[string][]string{
		http.MethodGet: {"sales_operator", "sales_worker", "admin"},
	}))
	mux.Handle("/api/v1/sales/clients/", protect(http.HandlerFunc(clientController.HandleClientByID), map[string][]string{
		http.MethodGet:   {"sales_operator", "sales_worker", "admin"},
		http.MethodPatch: {"sales_operator", "admin"},
	}))
	mux.Handle("/api/v1/sales/clients/:id", protect(http.HandlerFunc(clientController.HandleClientByID), map[string][]string{
		http.MethodGet:   {"sales_operator", "sales_worker", "admin"},
		http.MethodPatch: {"sales_operator", "admin"},
	}))
	mux.Handle("/api/v1/sales/fulfillment/process", protect(http.HandlerFunc(fulfillmentController.HandleProcessQueue), map[string][]string{
		http.MethodPost: {"sales_worker", "admin"},
	}))
	mux.Handle("/api/v1/sales/fulfillment/queue", protect(http.HandlerFunc(fulfillmentController.HandleQueueStatus), map[string][]string{
		http.MethodGet: {"sales_worker", "admin"},
	}))
	mux.Handle("/api/v1/sales/metrics", protect(http.HandlerFunc(orderController.HandleMetrics), map[string][]string{
		http.MethodGet: {"admin"},
	}))
	return middlewares.ErrorHandler(mux)
}
