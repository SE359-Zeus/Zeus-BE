package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"zeus-scm-service/internal/config"
	"zeus-scm-service/internal/handler"
	"zeus-scm-service/internal/handler/middleware"
	"zeus-scm-service/internal/messaging"
	sqliteRepo "zeus-scm-service/internal/repository/sqlite"
	"zeus-scm-service/internal/service"

	openapiui "github.com/PeterTakahashi/gin-openapi/openapiui"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

func main() {
	cfg := config.Load()

	db, err := sqliteRepo.NewDB(cfg.DBPath)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	if err := sqliteRepo.AutoMigrate(db); err != nil {
		log.Printf("auto-migrate warning: %v", err)
	}

	db.Exec(`
		CREATE TABLE IF NOT EXISTS api_keys (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			key_prefix TEXT NOT NULL,
			key_hash TEXT NOT NULL UNIQUE,
			active INTEGER NOT NULL DEFAULT 1,
			expires_at DATETIME,
			last_used_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			deleted_at DATETIME
		)
	`)
	var keyCount int64
	db.Raw("SELECT COUNT(*) FROM api_keys WHERE deleted_at IS NULL").Scan(&keyCount)
	if keyCount == 0 {
		db.Exec(
			"INSERT INTO api_keys (id, name, key_prefix, key_hash, active) VALUES (?, ?, ?, ?, ?)",
			uuid.New().String(), "Default ZeuS API Key", "scm_zeus",
			"$2a$10$QYXXHtQZn541zxmM15P0kebAjJMg6.VzkRbIWk9F.AZPF6FD3dI7a", true,
		)
		log.Println("seeded default API key: scm_zeus_master_key_2026")
	}

	if err := messaging.EnsureQueues(cfg.RabbitMQURL); err != nil {
		log.Printf("RabbitMQ unavailable (deficit pool disabled): %v", err)
	} else {
		ctx, stop := context.WithCancel(context.Background())
		defer stop()
		messaging.StartExpiryReconciler(ctx, cfg.RabbitMQURL, 5*time.Minute)
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
	inventoryH := handler.NewInventoryHandler(inventorySvc)

	r := gin.Default()

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
	api.Use(middleware.APIKeyAuth(db))
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
