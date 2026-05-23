package controllers

import (
	"net/http"

	"zeus-sales-service/internal/middlewares"
	"zeus-sales-service/internal/service"
)

func NewMux(services *service.Services) http.Handler {
	mux := http.NewServeMux()
	orderController := NewOrderController(services.Orders)
	clientController := NewClientController(services.Clients)
	fulfillmentController := NewFulfillmentController(services.Fulfillment)

	mux.HandleFunc("/api/v1/sales/orders", orderController.HandleOrders)
	mux.HandleFunc("/api/v1/sales/orders/", orderController.HandleOrderByID)
	mux.HandleFunc("/api/v1/sales/orders/:id", orderController.HandleOrderByID)
	mux.HandleFunc("/api/v1/sales/orders/{id}/cancel", orderController.HandleCancelOrder)
	mux.HandleFunc("/api/v1/sales/clients", clientController.HandleClients)
	mux.HandleFunc("/api/v1/sales/clients/", clientController.HandleClientByID)
	mux.HandleFunc("/api/v1/sales/clients/:id", clientController.HandleClientByID)
	mux.HandleFunc("/api/v1/sales/fulfillment/process", fulfillmentController.HandleProcessQueue)
	mux.HandleFunc("/api/v1/sales/fulfillment/queue", fulfillmentController.HandleQueueStatus)
	mux.HandleFunc("/api/v1/sales/metrics", orderController.HandleMetrics)
	return middlewares.ErrorHandler(mux)
}
