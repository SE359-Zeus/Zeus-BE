package main

import (
	"context"
	"log"
	"os"
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
)

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
		{Method: "GET", Path: "/api/v1/inventory/products", RequiredLevel: "Worker"},
		{Method: "GET", Path: "/api/v1/inventory/products/:id", RequiredLevel: "Worker"},
		{Method: "POST", Path: "/api/v1/inventory/products", RequiredLevel: "Operator"},
		{Method: "PUT", Path: "/api/v1/inventory/products/:id", RequiredLevel: "Operator"},
		{Method: "GET", Path: "/api/v1/inventory/product-models/:code", RequiredLevel: "Worker"},
		{Method: "POST", Path: "/api/v1/inventory/product-models", RequiredLevel: "Operator"},
		{Method: "GET", Path: "/api/v1/inventory/parts", RequiredLevel: "Worker"},
		{Method: "GET", Path: "/api/v1/inventory/parts/:id", RequiredLevel: "Worker"},
		{Method: "POST", Path: "/api/v1/inventory/parts", RequiredLevel: "Operator"},
		{Method: "PUT", Path: "/api/v1/inventory/parts/:id", RequiredLevel: "Operator"},
		{Method: "PUT", Path: "/api/v1/inventory/parts/:id/condition", RequiredLevel: "Operator"},
		{Method: "POST", Path: "/api/v1/inventory/parts/:id/scrap", RequiredLevel: "Worker"},
		{Method: "POST", Path: "/api/v1/inventory/parts/:id/install", RequiredLevel: "Worker"},
		{Method: "POST", Path: "/api/v1/inventory/parts/:id/remove", RequiredLevel: "Worker"},
		{Method: "GET", Path: "/api/v1/inventory/part-catalog", RequiredLevel: "Worker"},
		{Method: "GET", Path: "/api/v1/inventory/part-catalog/:id", RequiredLevel: "Worker"},
		{Method: "GET", Path: "/api/v1/vendors/optimal", RequiredLevel: "Operator"},
		{Method: "POST", Path: "/api/v1/vendors/:id/recalc-metrics", RequiredLevel: "Operator"},
		{Method: "POST", Path: "/api/v1/purchase-orders/draft", RequiredLevel: "Worker"},
		{Method: "POST", Path: "/api/v1/purchase-orders/:poId/line-items", RequiredLevel: "Worker"},
		{Method: "POST", Path: "/api/v1/purchase-orders/:poId/approve", RequiredLevel: "Operator"},
		{Method: "PUT", Path: "/api/v1/purchase-orders/:poId/state", RequiredLevel: "Operator"},
		{Method: "POST", Path: "/api/v1/goods-receipts/:grId/lock", RequiredLevel: "Worker"},
		{Method: "POST", Path: "/api/v1/goods-receipts/:grId/process", RequiredLevel: "Worker"},
		{Method: "DELETE", Path: "/api/v1/goods-receipts/:grId/lock", RequiredLevel: "Worker"},
		{Method: "POST", Path: "/api/v1/shipments/:shipmentId/lock", RequiredLevel: "Worker"},
		{Method: "POST", Path: "/api/v1/shipments/:shipmentId/dispatch", RequiredLevel: "Worker"},
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
	r.Use(gin.Logger(), middleware.Recovery())

	public := r.Group("/")
	public.Use(middleware.Public())
	{
		public.GET("/docs/*any", openapiui.WrapHandler(openapiui.Config{
			Title: "Zeus SCM API",
			SpecProvider: func() ([]byte, error) {
				data, err := os.ReadFile("docs/openapi.yaml")
				if err != nil {
					return nil, err
				}
				var parsed any
				if err := yaml.Unmarshal(data, &parsed); err != nil {
					return nil, err
				}
				return json.Marshal(parsed)
			},
			Theme: "dark",
		}))
		public.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})
	}

	api := r.Group("/api/v1")
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
