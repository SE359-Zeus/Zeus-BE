package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"time"

	"zeus-scm-service/internal/config"
	"zeus-scm-service/internal/handler"
	"zeus-scm-service/internal/handler/middleware"
	"zeus-scm-service/internal/infrastructure/cache"
	"zeus-scm-service/internal/infrastructure/messaging"
	sqliteRepo "zeus-scm-service/internal/repository/sqlite"
	"zeus-scm-service/internal/service"

	openapiui "github.com/PeterTakahashi/gin-openapi/openapiui"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

const scmAPIPrefix = "/api/v1/scm"

func main() {
	cfg := config.Load()

	db, err := sqliteRepo.NewDB(cfg.DBPath)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	mq, err := messaging.NewRabbitMQ(cfg.RabbitMQURL)
	if err != nil {
		log.Printf("running in degraded mode: RabbitMQ unavailable [RabbitMQURL: %s], deficit pool disabled: %v", cfg.RabbitMQURL, err)
		mq = nil
	} else {
		log.Printf("RabbitMQ connected: %s", cfg.RabbitMQURL)
		defer mq.Close()
		stop := make(chan struct{})
		defer close(stop)
		mq.StartExpiryReconciler(5*time.Minute, stop)
	}

	vendorRepo := sqliteRepo.NewVendorRepository(db)
	poRepo := sqliteRepo.NewPORepository(db)
	grRepo := sqliteRepo.NewGoodsReceiptRepository(db)
	shipmentRepo := sqliteRepo.NewShipmentRepository(db)
	inventoryRepo := sqliteRepo.NewInventoryRepository(db)
	stockRepo := sqliteRepo.NewStockRepository(db)

	vendorSvc := service.NewVendorService(vendorRepo)
	poSvc := service.NewPOService(poRepo, stockRepo, cfg.RabbitMQURL)
	grSvc := service.NewGoodsReceiptService(grRepo, stockRepo, poRepo, cfg.AgingThresholdYears)
	shipmentSvc := service.NewShipmentService(shipmentRepo, stockRepo)
	inventorySvc := service.NewInventoryService(inventoryRepo)

	vendorH := handler.NewVendorHandler(vendorSvc)
	poH := handler.NewPOHandler(poSvc)
	grH := handler.NewGoodsReceiptHandler(grSvc)
	shipmentH := handler.NewShipmentHandler(shipmentSvc)

	jwtSvc, err := service.NewJWTService(cfg.JwtPublicKeyPath)
	if err != nil {
		log.Fatalf("JWT service init failed: %v", err)
	}

	rolesWorker := []string{"admin", "scm_operator", "scm_worker"}
	rolesOperator := []string{"admin", "scm_operator"}

	var cacheBackend cache.Cache = cache.NewNoop()
	if cfg.ValkeyAddr != "" {
		if valkeyCache, err := cache.NewValkey(cfg.ValkeyAddr); err != nil {
			log.Printf("running in degraded mode: Valkey unavailable [ValkeyAddr: %s], cache disabled: %v", cfg.ValkeyAddr, err)
		} else {
			cacheBackend = valkeyCache
			log.Printf("Valkey connected: %s", cfg.ValkeyAddr)
			service.WarmupCache(context.Background(), db, cacheBackend)
		}
	}
	inventorySvc = service.NewCachedInventoryService(inventorySvc, cacheBackend)
	inventoryH := handler.NewInventoryHandler(inventorySvc)

	r := gin.New()
	// Disable trailing-slash redirect: prevents Gin from issuing a 301/302
	// from /docs/ → /docs, which would escape the nginx /scm/docs/ proxy
	// and fall through to the catch-all "Success!" location.
	r.RedirectTrailingSlash = false
	r.Use(gin.Logger(), middleware.Recovery())

	// ── Public routes (no auth) ──────────────────────────────────────────────
	{
		specPath := findOpenAPISpec()
		specURL := runtimeServerURL(cfg.ServerPort)
		spec, err := loadOpenAPISpec(specPath, specURL)
		if err != nil {
			log.Printf("warning: could not load openapi spec at %s: %v", specPath, err)
		}

		// NOTE: SpecURL must be a RELATIVE path ("./openapi.json"), NOT absolute.
		// When the browser is at https://.../scm/docs/, a relative URL resolves
		// to https://.../scm/docs/openapi.json → nginx proxies to SCM correctly.
		// An absolute path "/docs/openapi.json" would resolve to
		// https://.../docs/openapi.json → nginx catch-all → "Success!" ❌
		//
		// The /docs (no-slash) case is handled by nginx:
		//   location = /scm/docs { return 301 /scm/docs/; }
		// We do NOT add a Go-side redirect because it would emit Location: /docs/
		// (without /scm/), causing the browser to escape the proxy prefix.
		r.GET("/docs/*any", openapiui.WrapHandler(openapiui.Config{
			Title:   "Zeus SCM API",
			SpecURL: "./openapi.json",
			SpecProvider: func() ([]byte, error) {
				if spec == nil {
					return buildOpenAPISpec(cfg.ServerPort)()
				}
				return spec, nil
			},
			Theme: "dark",
		}))
		r.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})
	}

	api := r.Group(scmAPIPrefix)
	api.Use(middleware.Authenticate(jwtSvc, db))
	{
		api.GET("/vendors/optimal", middleware.RequireRoles(rolesOperator...), vendorH.GetOptimalSupplier)
		api.POST("/vendors/:id/recalc-metrics", middleware.RequireRoles(rolesOperator...), vendorH.UpdateSupplierMetrics)

		api.POST("/purchase-orders/draft", middleware.RequireRoles(rolesWorker...), poH.CreateDraft)
		api.POST("/purchase-orders/:poId/line-items", middleware.RequireRoles(rolesWorker...), poH.AddLineItemWithLock)
		api.POST("/purchase-orders/:poId/approve", middleware.RequireRoles(rolesOperator...), poH.ApprovePO)
		api.PUT("/purchase-orders/:poId/state", middleware.RequireRoles(rolesOperator...), poH.TransitionState)

		api.POST("/goods-receipts/:grId/lock", middleware.RequireRoles(rolesWorker...), grH.AcquireLock)
		api.POST("/goods-receipts/:grId/process", middleware.RequireRoles(rolesWorker...), grH.ProcessBlindReceipt)
		api.DELETE("/goods-receipts/:grId/lock", middleware.RequireRoles(rolesWorker...), grH.ReleaseLock)

		api.POST("/shipments/:shipmentId/lock", middleware.RequireRoles(rolesWorker...), shipmentH.AcquireDispatchLock)
		api.POST("/shipments/:shipmentId/dispatch", middleware.RequireRoles(rolesWorker...), shipmentH.DispatchShipment)

		api.GET("/inventory/products", middleware.RequireRoles(rolesWorker...), inventoryH.ListProducts)
		api.GET("/inventory/products/:id", middleware.RequireRoles(rolesWorker...), inventoryH.GetProduct)
		api.POST("/inventory/products", middleware.RequireRoles(rolesOperator...), inventoryH.CreateProduct)
		api.POST("/inventory/products/register", middleware.RequireRoles(rolesOperator...), inventoryH.RegisterProduct)
		api.PUT("/inventory/products/:id", middleware.RequireRoles(rolesOperator...), inventoryH.UpdateProduct)
		api.GET("/inventory/product-models/:code", middleware.RequireRoles(rolesWorker...), inventoryH.GetProductModel)
		api.POST("/inventory/product-models", middleware.RequireRoles(rolesOperator...), inventoryH.CreateProductModel)
		api.GET("/inventory/parts", middleware.RequireRoles(rolesWorker...), inventoryH.ListParts)
		api.GET("/inventory/parts/:id", middleware.RequireRoles(rolesWorker...), inventoryH.GetPart)
		api.POST("/inventory/parts", middleware.RequireRoles(rolesOperator...), inventoryH.CreatePart)
		api.PUT("/inventory/parts/:id", middleware.RequireRoles(rolesOperator...), inventoryH.UpdatePart)
		api.PUT("/inventory/parts/:id/condition", middleware.RequireRoles(rolesOperator...), inventoryH.UpdatePartCondition)
		api.POST("/inventory/parts/:id/scrap", middleware.RequireRoles(rolesWorker...), inventoryH.MarkPartScrapped)
		api.POST("/inventory/parts/:id/install", middleware.RequireRoles(rolesWorker...), inventoryH.InstallPart)
		api.POST("/inventory/parts/:id/remove", middleware.RequireRoles(rolesWorker...), inventoryH.RemovePart)
		api.GET("/inventory/part-catalog", middleware.RequireRoles(rolesWorker...), inventoryH.ListPartCatalog)
		api.GET("/inventory/part-catalog/:id", middleware.RequireRoles(rolesWorker...), inventoryH.GetPartCatalog)
	}

	log.Printf("Zeus SCM service starting on :%s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func findOpenAPISpec() string {
	paths := []string{"docs/openapi.yaml", "./docs/openapi.yaml", filepath.Join(".", "docs", "openapi.yaml")}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "docs/openapi.yaml"
}

func loadOpenAPISpec(specPath, serverURL string) ([]byte, error) {
	data, err := os.ReadFile(specPath)
	if err != nil {
		return nil, err
	}

	var parsed map[string]any
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		return nil, err
	}
	parsed["servers"] = []any{map[string]any{"url": serverURL}}

	return json.Marshal(parsed)
}

func buildOpenAPISpec(serverPort string) func() ([]byte, error) {
	return func() ([]byte, error) {
		specPath := findOpenAPISpec()
		data, err := os.ReadFile(specPath)
		if err != nil {
			return nil, err
		}

		var parsed map[string]any
		if err := yaml.Unmarshal(data, &parsed); err != nil {
			return nil, err
		}
		parsed["servers"] = []any{map[string]any{"url": runtimeServerURL(serverPort)}}

		return json.Marshal(parsed)
	}
}

func runtimeServerURL(port string) string {
	// PUBLIC_BASE_URL is set in stack.env on the production server.
	// e.g. PUBLIC_BASE_URL=https://zeus.ryanandexen.qzz.io
	// Swagger UI will call: PUBLIC_BASE_URL + /api/v1/scm + <path-from-spec>
	//
	// Locally (no env var set), falls back to http://localhost:<port>.
	// Swagger UI will call: http://localhost:8081 + /api/v1/scm + <path>
	base := os.Getenv("PUBLIC_BASE_URL")
	if base == "" {
		if port == "" {
			port = "8081"
		}
		base = "http://localhost:" + port
	}
	return base + scmAPIPrefix
}
