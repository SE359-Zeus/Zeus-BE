package controllers

import (
	"net/http"

	"zeus-mrp-service/internal/middlewares"
	"zeus-mrp-service/internal/service"
)

type ProductionController struct {
	svc *service.ProductionService
}

func NewProductionController(svc *service.ProductionService) *ProductionController {
	return &ProductionController{svc: svc}
}

func NewMux(svc *service.ProductionService) http.Handler {
	mux := http.NewServeMux()
	controller := NewProductionController(svc)

	mux.HandleFunc("GET /api/v1/mrp/readiness", controller.GetReadinessMatrix)
	mux.HandleFunc("GET /api/v1/mrp/readiness/metrics", controller.GetReadinessMetrics)
	mux.HandleFunc("GET /api/v1/mrp/readiness/export", controller.ExportReport)
	mux.HandleFunc("GET /api/v1/mrp/readiness/{orderId}", controller.GetReadinessByOrderID)
	mux.HandleFunc("POST /api/v1/mrp/readiness/{orderId}/generate-po", controller.GeneratePOForDeficits)
	mux.HandleFunc("GET /api/v1/mrp/shortages", controller.GetShortages)

	mux.HandleFunc("GET /api/v1/mrp/assemblies", controller.GetAssemblies)
	mux.HandleFunc("POST /api/v1/mrp/assemblies", controller.CreateAssembly)
	mux.HandleFunc("PUT /api/v1/mrp/assemblies/{id}", controller.UpdateAssembly)
	mux.HandleFunc("DELETE /api/v1/mrp/assemblies/{id}", controller.DeleteAssembly)
	mux.HandleFunc("GET /api/v1/mrp/catalog", controller.GetCatalog)
	mux.HandleFunc("GET /api/v1/mrp/catalog/{sku}/where-used", controller.GetWhereUsed)

	mux.HandleFunc("GET /api/v1/mrp/demand", controller.GetDemandSummary)
	mux.HandleFunc("POST /api/v1/mrp/demand/generate-pos", controller.GeneratePOs)
	mux.HandleFunc("GET /api/v1/mrp/demand/{orderId}/pick-list", controller.GetPickList)
	mux.HandleFunc("POST /api/v1/mrp/demand/{orderId}/pick-list", controller.GeneratePickList)

	mux.HandleFunc("GET /api/v1/mrp/inventory/ledger", controller.GetInventoryLedger)
	mux.HandleFunc("GET /api/v1/mrp/inventory/metrics", controller.GetInventoryMetrics)
	mux.HandleFunc("GET /api/v1/mrp/inventory/ledger/export", controller.ExportInventoryCSV)
	mux.HandleFunc("GET /api/v1/mrp/inventory/transactions/{txnId}", controller.GetInventoryTransactionByID)
	mux.HandleFunc("GET /api/v1/mrp/inventory/balance/{sku}", controller.GetInventoryBalanceBySKU)

	mux.HandleFunc("POST /api/v1/production/orders", controller.CreateOrder)

	return middlewares.ErrorHandler(mux)
}
