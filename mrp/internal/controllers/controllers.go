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

func NewMux(svc *service.ProductionService, authVerifier middlewares.TokenVerifier) http.Handler {
	mux := http.NewServeMux()
	controller := NewProductionController(svc)
	authenticate := middlewares.Authenticate(authVerifier)
	protect := func(handler http.Handler, methodRoles map[string][]string) http.Handler {
		return middlewares.Chain(handler, authenticate, middlewares.RequireMethodRoles(methodRoles))
	}

	// --- Readiness / Dashboard ---
	mux.Handle("/api/v1/mrp/readiness", protect(http.HandlerFunc(controller.GetReadinessMatrix), map[string][]string{
		http.MethodGet: {"mrp_operator", "mrp_worker", "admin"},
	}))
	mux.Handle("/api/v1/mrp/readiness/metrics", protect(http.HandlerFunc(controller.GetReadinessMetrics), map[string][]string{
		http.MethodGet: {"mrp_operator", "mrp_worker", "admin"},
	}))
	mux.Handle("/api/v1/mrp/readiness/export", protect(http.HandlerFunc(controller.ExportReport), map[string][]string{
		http.MethodGet: {"mrp_operator", "mrp_worker", "admin"},
	}))
	mux.Handle("/api/v1/mrp/readiness/{orderId}", protect(http.HandlerFunc(controller.GetReadinessByOrderID), map[string][]string{
		http.MethodGet: {"mrp_operator", "mrp_worker", "admin"},
	}))
	mux.Handle("/api/v1/mrp/readiness/{orderId}/generate-po", protect(http.HandlerFunc(controller.GeneratePOForDeficits), map[string][]string{
		http.MethodPost: {"mrp_operator", "admin"},
	}))
	mux.Handle("/api/v1/mrp/shortages", protect(http.HandlerFunc(controller.GetShortages), map[string][]string{
		http.MethodGet: {"mrp_operator", "mrp_worker", "admin"},
	}))

	// --- BOM & Catalog ---
	mux.Handle("/api/v1/mrp/assemblies", protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			controller.CreateAssembly(w, r)
		default:
			controller.GetAssemblies(w, r)
		}
	}), map[string][]string{
		http.MethodGet:  {"mrp_operator", "mrp_worker", "admin"},
		http.MethodPost: {"mrp_operator", "admin"},
	}))
	mux.Handle("/api/v1/mrp/assemblies/{id}", protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			controller.GetAssemblyDetail(w, r)
		case http.MethodPut:
			controller.UpdateAssembly(w, r)
		case http.MethodDelete:
			controller.DeleteAssembly(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}), map[string][]string{
		http.MethodGet:    {"mrp_operator", "mrp_worker", "admin"},
		http.MethodPut:    {"mrp_operator", "admin"},
		http.MethodDelete: {"mrp_operator", "admin"},
	}))
	mux.Handle("/api/v1/mrp/catalog", protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			controller.GetCatalog(w, r)
		case http.MethodPost:
			controller.CreateCatalogPart(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}), map[string][]string{
		http.MethodGet:  {"mrp_operator", "mrp_worker", "admin"},
		http.MethodPost: {"mrp_operator", "admin"},
	}))
	mux.Handle("/api/v1/mrp/catalog/{sku}", protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			controller.UpdateCatalogPart(w, r)
		case http.MethodDelete:
			controller.DeleteCatalogPart(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}), map[string][]string{
		http.MethodPut:    {"mrp_operator", "admin"},
		http.MethodDelete: {"mrp_operator", "admin"},
	}))
	mux.Handle("/api/v1/mrp/catalog/{sku}/where-used", protect(http.HandlerFunc(controller.GetWhereUsed), map[string][]string{
		http.MethodGet: {"mrp_operator", "mrp_worker", "admin"},
	}))

	// --- Demand ---
	mux.Handle("/api/v1/mrp/demand/metrics", protect(http.HandlerFunc(controller.GetDemandMetrics), map[string][]string{
		http.MethodGet: {"mrp_operator", "mrp_worker", "admin"},
	}))
	mux.Handle("/api/v1/mrp/demand", protect(http.HandlerFunc(controller.GetDemandSummary), map[string][]string{
		http.MethodGet: {"mrp_operator", "mrp_worker", "admin"},
	}))
	mux.Handle("/api/v1/mrp/demand/generate-pos", protect(http.HandlerFunc(controller.GeneratePOs), map[string][]string{
		http.MethodPost: {"mrp_operator", "admin"},
	}))
	mux.Handle("/api/v1/mrp/demand/{orderId}/pick-list", protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			controller.GeneratePickList(w, r)
		default:
			controller.GetPickList(w, r)
		}
	}), map[string][]string{
		http.MethodGet:  {"mrp_operator", "mrp_worker", "admin"},
		http.MethodPost: {"mrp_operator", "admin"},
	}))
	mux.Handle("/api/v1/mrp/demand/{orderId}", protect(http.HandlerFunc(controller.DeleteDemand), map[string][]string{
		http.MethodDelete: {"mrp_operator", "admin"},
	}))

	// --- Inventory Ledger (read-only proxy to SCM) ---
	mux.Handle("/api/v1/mrp/inventory/ledger", protect(http.HandlerFunc(controller.GetInventoryLedger), map[string][]string{
		http.MethodGet: {"mrp_operator", "mrp_worker", "admin"},
	}))
	mux.Handle("/api/v1/mrp/inventory/metrics", protect(http.HandlerFunc(controller.GetInventoryMetrics), map[string][]string{
		http.MethodGet: {"mrp_operator", "mrp_worker", "admin"},
	}))
	mux.Handle("/api/v1/mrp/inventory/ledger/export", protect(http.HandlerFunc(controller.ExportInventoryCSV), map[string][]string{
		http.MethodGet: {"mrp_operator", "mrp_worker", "admin"},
	}))
	mux.Handle("/api/v1/mrp/inventory/transactions/{txnId}", protect(http.HandlerFunc(controller.GetInventoryTransactionByID), map[string][]string{
		http.MethodGet: {"mrp_operator", "mrp_worker", "admin"},
	}))
	mux.Handle("/api/v1/mrp/inventory/balance/{sku}", protect(http.HandlerFunc(controller.GetInventoryBalanceBySKU), map[string][]string{
		http.MethodGet: {"mrp_operator", "mrp_worker", "admin"},
	}))

	// --- Production Orders ---
	mux.Handle("/api/v1/production/orders", protect(http.HandlerFunc(controller.CreateOrder), map[string][]string{
		http.MethodPost: {"mrp_operator", "admin"},
	}))

	return middlewares.ErrorHandler(mux)
}
