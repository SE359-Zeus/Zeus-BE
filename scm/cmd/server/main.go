package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"time"

	"zeus-scm-service/internal/cache"
	"zeus-scm-service/internal/config"
	"zeus-scm-service/internal/handler"
	"zeus-scm-service/internal/handler/middleware"
	"zeus-scm-service/internal/messaging"
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

	vendorSvc := service.NewVendorService(db)
	poSvc := service.NewPOService(db, cfg.RabbitMQURL)
	grSvc := service.NewGoodsReceiptService(db, cfg.AgingThresholdYears)
	shipmentSvc := service.NewShipmentService(db)
	inventorySvc := service.NewInventoryService(db)

	vendorH := handler.NewVendorHandler(vendorSvc)
	poH := handler.NewPOHandler(poSvc)
	grH := handler.NewGoodsReceiptHandler(grSvc)
	shipmentH := handler.NewShipmentHandler(shipmentSvc)

	jwtSvc, err := service.NewJWTService(cfg.JwtPublicKeyPath)
	if err != nil {
		log.Fatalf("JWT service init failed: %v", err)
	}

	routeAccessRules := []service.RouteAccessRule{
		{Method: "GET", Path: "/api/v1/scm/inventory/products", RequiredLevel: "Worker"},
		{Method: "GET", Path: "/api/v1/scm/inventory/products/:id", RequiredLevel: "Worker"},
		{Method: "POST", Path: "/api/v1/scm/inventory/products", RequiredLevel: "Operator"},
		{Method: "PUT", Path: "/api/v1/scm/inventory/products/:id", RequiredLevel: "Operator"},
		{Method: "GET", Path: "/api/v1/scm/inventory/product-models/:code", RequiredLevel: "Worker"},
		{Method: "POST", Path: "/api/v1/scm/inventory/product-models", RequiredLevel: "Operator"},
		{Method: "GET", Path: "/api/v1/scm/inventory/parts", RequiredLevel: "Worker"},
		{Method: "GET", Path: "/api/v1/scm/inventory/parts/:id", RequiredLevel: "Worker"},
		{Method: "POST", Path: "/api/v1/scm/inventory/parts", RequiredLevel: "Operator"},
		{Method: "PUT", Path: "/api/v1/scm/inventory/parts/:id", RequiredLevel: "Operator"},
		{Method: "PUT", Path: "/api/v1/scm/inventory/parts/:id/condition", RequiredLevel: "Operator"},
		{Method: "POST", Path: "/api/v1/scm/inventory/parts/:id/scrap", RequiredLevel: "Worker"},
		{Method: "POST", Path: "/api/v1/scm/inventory/parts/:id/install", RequiredLevel: "Worker"},
		{Method: "POST", Path: "/api/v1/scm/inventory/parts/:id/remove", RequiredLevel: "Worker"},
		{Method: "GET", Path: "/api/v1/scm/inventory/part-catalog", RequiredLevel: "Worker"},
		{Method: "GET", Path: "/api/v1/scm/inventory/part-catalog/:id", RequiredLevel: "Worker"},
		{Method: "GET", Path: "/api/v1/scm/vendors/optimal", RequiredLevel: "Operator"},
		{Method: "POST", Path: "/api/v1/scm/vendors/:id/recalc-metrics", RequiredLevel: "Operator"},
		{Method: "POST", Path: "/api/v1/scm/purchase-orders/draft", RequiredLevel: "Worker"},
		{Method: "POST", Path: "/api/v1/scm/purchase-orders/:poId/line-items", RequiredLevel: "Worker"},
		{Method: "POST", Path: "/api/v1/scm/purchase-orders/:poId/approve", RequiredLevel: "Operator"},
		{Method: "PUT", Path: "/api/v1/scm/purchase-orders/:poId/state", RequiredLevel: "Operator"},
		{Method: "POST", Path: "/api/v1/scm/goods-receipts/:grId/lock", RequiredLevel: "Worker"},
		{Method: "POST", Path: "/api/v1/scm/goods-receipts/:grId/process", RequiredLevel: "Worker"},
		{Method: "DELETE", Path: "/api/v1/scm/goods-receipts/:grId/lock", RequiredLevel: "Worker"},
		{Method: "POST", Path: "/api/v1/scm/shipments/:shipmentId/lock", RequiredLevel: "Worker"},
		{Method: "POST", Path: "/api/v1/scm/shipments/:shipmentId/dispatch", RequiredLevel: "Worker"},
	}

	roleLevels := map[string]int{
		"admin":          3,
		"scm_operator":   2,
		"scm_worker":     1,
		"mrp_operator":   2,
		"mrp_worker":     1,
		"sales_operator": 2,
		"sales_worker":   1,
	}

	rbacSvc := service.NewRBACService(routeAccessRules, roleLevels)

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

	public := r.Group("/")
	public.Use(middleware.Public())
	{
		specPath := findOpenAPISpec()
		specURL := runtimeServerURL(cfg.ServerPort)
		spec, err := loadOpenAPISpec(specPath, specURL)
		if err != nil {
			log.Printf("warning: could not load openapi spec at %s: %v", specPath, err)
		}

		// /docs (no trailing slash) → /docs/ so the UI assets load correctly.
		// Works both locally (localhost:8081/docs → /docs/) and in production
		// where nginx strips the /scm prefix before forwarding to this server.
		public.GET("/docs", func(c *gin.Context) { c.Redirect(302, "/docs/") })
		public.GET("/docs/*any", openapiui.WrapHandler(openapiui.Config{
			Title:   "Zeus SCM API",
			// Absolute path: avoids browser resolving "./openapi.json" relative
			// to whatever page URL the user happens to be on, which would
			// produce a duplicated or wrong path.
			SpecURL: "/docs/openapi.json",
			SpecProvider: func() ([]byte, error) {
				if spec == nil {
					return buildOpenAPISpec(cfg.ServerPort)()
				}
				return spec, nil
			},
			Theme: "dark",
		}))
		public.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})
	}

	api := r.Group(scmAPIPrefix)
	api.Use(middleware.Authenticate(jwtSvc, db), middleware.RequireRoleLevel(rbacSvc))
	{
		api.GET("/vendors/optimal", vendorH.GetOptimalSupplier)
		api.POST("/vendors/:id/recalc-metrics", vendorH.UpdateSupplierMetrics)

		api.POST("/purchase-orders/draft", poH.CreateDraft)
		api.POST("/purchase-orders/:poId/line-items", poH.AddLineItemWithLock)
		api.POST("/purchase-orders/:poId/approve", poH.ApprovePO)
		api.PUT("/purchase-orders/:poId/state", poH.TransitionState)

		api.POST("/goods-receipts/:grId/lock", grH.AcquireLock)
		api.POST("/goods-receipts/:grId/process", grH.ProcessBlindReceipt)
		api.DELETE("/goods-receipts/:grId/lock", grH.ReleaseLock)

		api.POST("/shipments/:shipmentId/lock", shipmentH.AcquireDispatchLock)
		api.POST("/shipments/:shipmentId/dispatch", shipmentH.DispatchShipment)

		api.GET("/inventory/products", inventoryH.ListProducts)
		api.GET("/inventory/products/:id", inventoryH.GetProduct)
		api.POST("/inventory/products", inventoryH.CreateProduct)
		api.PUT("/inventory/products/:id", inventoryH.UpdateProduct)
		api.GET("/inventory/product-models/:code", inventoryH.GetProductModel)
		api.POST("/inventory/product-models", inventoryH.CreateProductModel)
		api.GET("/inventory/parts", inventoryH.ListParts)
		api.GET("/inventory/parts/:id", inventoryH.GetPart)
		api.POST("/inventory/parts", inventoryH.CreatePart)
		api.PUT("/inventory/parts/:id", inventoryH.UpdatePart)
		api.PUT("/inventory/parts/:id/condition", inventoryH.UpdatePartCondition)
		api.POST("/inventory/parts/:id/scrap", inventoryH.MarkPartScrapped)
		api.POST("/inventory/parts/:id/install", inventoryH.InstallPart)
		api.POST("/inventory/parts/:id/remove", inventoryH.RemovePart)
		api.GET("/inventory/part-catalog", inventoryH.ListPartCatalog)
		api.GET("/inventory/part-catalog/:id", inventoryH.GetPartCatalog)
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
