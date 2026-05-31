package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"zeus-scm-service/internal/config"
	"zeus-scm-service/internal/handler"
	"zeus-scm-service/internal/handler/middleware"
	"zeus-scm-service/internal/infrastructure/cache"
	"zeus-scm-service/internal/infrastructure/cronjob"
	"zeus-scm-service/internal/infrastructure/messaging"
	"zeus-scm-service/internal/infrastructure/observability"
	sqliteRepo "zeus-scm-service/internal/repository/sqlite"
	valkeyRepo "zeus-scm-service/internal/repository/valkey"
	"zeus-scm-service/internal/service"

	openapiui "github.com/PeterTakahashi/gin-openapi/openapiui"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

const scmAPIPrefix = "/api/v1/scm"

func main() {
	cfg := config.Load()

	// ── Observability: logger + metrics (must be first) ───────────────────────
	env := cfg.Env
	if env == "" {
		env = os.Getenv("APP_ENV")
	}
	obs, shutdownObs := observability.Setup(observability.Config{
		ServiceName:   "scm",
		Env:           env,
		AlloyURL:      cfg.AlloyURL,
		AlloyUsername: cfg.AlloyUsername,
		AlloyPassword: cfg.AlloyPassword,
	})
	defer shutdownObs()
	slog.SetDefault(obs.Logger)

	// ── Periodic metrics collector ────────────────────────────────────────────
	scheduler := cronjob.NewScheduler()
	scheduler.Register("metrics", 30*time.Second, cronjob.MetricsCollectorJob(obs.Metrics))
	scheduler.Start(context.Background())
	defer scheduler.Stop()

	slog.Info("starting scm service",
		slog.String("service", "scm"),
		slog.String("event", "startup"),
		slog.String("db_path", cfg.DBPath),
		slog.String("rabbitmq_url", cfg.RabbitMQURL),
		slog.String("valkey_addr", cfg.ValkeyAddr),
	)

	db, err := sqliteRepo.NewDB(cfg.DBPath)
	if err != nil {
		slog.Error("failed to connect to database",
			slog.String("service", "scm"),
			slog.String("event", "startup_failed"),
			slog.String("component", "database"),
			slog.Any("error", err),
		)
		os.Exit(1)
	}

	mq, err := messaging.NewRabbitMQ(cfg.RabbitMQURL)
	if err != nil {
		slog.Warn("running in degraded mode: rabbitmq unavailable",
			slog.String("service", "scm"),
			slog.String("event", "dependency_unavailable"),
			slog.String("component", "rabbitmq"),
			slog.String("url", cfg.RabbitMQURL),
			slog.Any("error", err),
		)
		mq = nil
	} else {
		slog.Info("rabbitmq connected",
			slog.String("service", "scm"),
			slog.String("event", "dependency_ready"),
			slog.String("component", "rabbitmq"),
			slog.String("url", cfg.RabbitMQURL),
		)
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
	lutRepo := sqliteRepo.NewLUTRepository(db)
	ledgerRepo := sqliteRepo.NewLedgerRepository(db)
	carrierRepo := sqliteRepo.NewCarrierRepository(db)

	vendorSvc := service.NewVendorService(vendorRepo, cfg.RabbitMQURL)
	lutSvc := service.NewLUTService(lutRepo)
	ledgerSvc := service.NewLedgerService(ledgerRepo)
	poSvc := service.NewPOService(poRepo, stockRepo, vendorRepo, cfg.RabbitMQURL)
	grSvc := service.NewGoodsReceiptService(grRepo, stockRepo, poRepo, cfg.AgingThresholdYears, ledgerSvc)
	shipmentSvc := service.NewShipmentService(shipmentRepo, stockRepo, poRepo, vendorRepo, carrierRepo, ledgerSvc, grRepo)
	inventorySvc := service.NewInventoryService(inventoryRepo)

	jwtSvc, err := service.NewJWTService(cfg.JwtPublicKeyPath)
	if err != nil {
		slog.Error("jwt service init failed",
			slog.String("service", "scm"),
			slog.String("event", "startup_failed"),
			slog.String("component", "jwt"),
			slog.Any("error", err),
		)
		os.Exit(1)
	}

	rolesWorker := []string{"admin", "scm_operator", "scm_worker", "api_key"}
	rolesOperator := []string{"admin", "scm_operator", "api_key"}

	var cacheBackend cache.Cache = cache.NewNoop()
	productCache := valkeyRepo.NewProductCache(cacheBackend)
	vendorCache := valkeyRepo.NewVendorCache(cacheBackend)
	if cfg.ValkeyAddr != "" {
		if valkeyCache, err := cache.NewValkey(cfg.ValkeyAddr); err != nil {
			slog.Warn("running in degraded mode: valkey unavailable",
				slog.String("service", "scm"),
				slog.String("event", "dependency_unavailable"),
				slog.String("component", "valkey"),
				slog.String("addr", cfg.ValkeyAddr),
				slog.Any("error", err),
			)
		} else {
			cacheBackend = valkeyCache
			productCache = valkeyRepo.NewProductCache(cacheBackend)
			vendorCache = valkeyRepo.NewVendorCache(cacheBackend)
			slog.Info("valkey connected",
				slog.String("service", "scm"),
				slog.String("event", "dependency_ready"),
				slog.String("component", "valkey"),
				slog.String("addr", cfg.ValkeyAddr),
			)
			service.WarmupCache(context.Background(), db, productCache)
		}
	}
	inventorySvc = service.NewCachedInventoryService(inventorySvc, productCache)
	vendorSvc = service.NewCachedVendorService(vendorSvc, vendorCache, vendorRepo)

	vendorH := handler.NewVendorHandler(vendorSvc)
	poH := handler.NewPOHandler(poSvc)
	grH := handler.NewGoodsReceiptHandler(grSvc)
	shipmentH := handler.NewShipmentHandler(shipmentSvc)
	inventoryH := handler.NewInventoryHandler(inventorySvc)
	lutH := handler.NewLUTHandler(lutSvc)
	ledgerH := handler.NewLedgerHandler(ledgerSvc)

	r := gin.New()
	// Disable trailing-slash redirect: prevents Gin from issuing a 301/302
	// from /docs/ → /docs, which would escape the nginx /scm/docs/ proxy
	// and fall through to the catch-all "Success!" location.
	r.RedirectTrailingSlash = false
	r.Use(
		middleware.CORS(),
		observability.Tracing("scm"), // inject trace_id / span_id
		middleware.RequestLogger(),
		middleware.Recovery(),
	)

	// Internal metrics endpoint — Alloy scrapes this.
	r.GET("/metrics", gin.WrapF(observability.MetricsHTTPHandler(obs.Metrics)))

	// ── Public routes (no auth) ──────────────────────────────────────────────
	{
		specPath := findOpenAPISpec()
		specURL := runtimeServerURL(cfg.ServerPort)
		spec, err := loadOpenAPISpec(specPath, specURL)
		if err != nil {
			slog.Warn("could not load openapi spec",
				slog.String("service", "scm"),
				slog.String("event", "openapi_load_failed"),
				slog.String("path", specPath),
				slog.Any("error", err),
			)
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
	api.Use(middleware.Authenticate(jwtSvc, db), middleware.Audit(mq))
	{
		api.GET("/vendors/optimal", middleware.RequireRoles(rolesOperator...), vendorH.GetOptimalSupplier)
		api.POST("/vendors/:id/recalc-metrics", middleware.RequireRoles(rolesOperator...), vendorH.UpdateSupplierMetrics)
		api.GET("/vendors", middleware.RequireRoles(rolesWorker...), vendorH.ListSuppliers)
		api.GET("/vendors/metrics", middleware.RequireRoles(rolesWorker...), vendorH.GetSupplierMetrics)
		api.POST("/vendors", middleware.RequireRoles(rolesOperator...), vendorH.CreateSupplier)
		api.POST("/vendors/:id/sku-mappings", middleware.RequireRoles(rolesOperator...), vendorH.CreateSkuMapping)
		api.GET("/vendors/export", middleware.RequireRoles(rolesWorker...), vendorH.ExportSuppliersReport)
		api.GET("/vendors/shortage-summary", middleware.RequireRoles(rolesWorker...), vendorH.GetShortageSummary)

		api.POST("/purchase-orders/draft", middleware.RequireRoles(rolesWorker...), poH.CreateDraft)
		api.POST("/purchase-orders/:poId/line-items", middleware.RequireRoles(rolesWorker...), poH.AddLineItemWithLock)
		api.POST("/purchase-orders/:poId/approve", middleware.RequireRoles(rolesOperator...), poH.ApprovePO)
		api.PUT("/purchase-orders/:poId/state", middleware.RequireRoles(rolesOperator...), poH.TransitionState)
		api.GET("/purchase-orders/export", middleware.RequireRoles(rolesWorker...), poH.ExportPOReport)
		api.GET("/purchase-orders", middleware.RequireRoles(rolesWorker...), poH.ListPOs)
		api.GET("/purchase-orders/:poId", middleware.RequireRoles(rolesWorker...), poH.GetPO)
		api.POST("/purchase-orders", middleware.RequireRoles(rolesWorker...), poH.CreatePO)

		api.POST("/goods-receipts/:grId/lock", middleware.RequireRoles(rolesWorker...), grH.AcquireLock)
		api.POST("/goods-receipts/:grId/process", middleware.RequireRoles(rolesWorker...), grH.ProcessBlindReceipt)
		api.DELETE("/goods-receipts/:grId/lock", middleware.RequireRoles(rolesWorker...), grH.ReleaseLock)
		api.GET("/goods-receipts/export", middleware.RequireRoles(rolesWorker...), grH.ExportGRReport)
		api.GET("/goods-receipts/export", middleware.RequireRoles(rolesWorker...), grH.ExportGRReport)
		api.GET("/goods-receipts", middleware.RequireRoles(rolesWorker...), grH.ListGRs)
		api.GET("/goods-receipts/metrics", middleware.RequireRoles(rolesWorker...), grH.GetMetrics)
		api.GET("/goods-receipts/:grId", middleware.RequireRoles(rolesWorker...), grH.GetGR)

		api.POST("/shipments/:shipmentId/lock", middleware.RequireRoles(rolesWorker...), shipmentH.AcquireDispatchLock)
		api.POST("/shipments/:shipmentId/dispatch", middleware.RequireRoles(rolesWorker...), shipmentH.DispatchShipment)
		api.GET("/shipments/export", middleware.RequireRoles(rolesWorker...), shipmentH.ExportShipmentReport)
		api.GET("/shipments", middleware.RequireRoles(rolesWorker...), shipmentH.ListShipments)
		api.GET("/shipments/metrics", middleware.RequireRoles(rolesWorker...), shipmentH.GetMetrics)
		api.GET("/shipments/carriers", middleware.RequireRoles(rolesWorker...), shipmentH.ListCarriers)
		api.GET("/shipments/:shipmentId", middleware.RequireRoles(rolesWorker...), shipmentH.GetShipment)
		api.POST("/shipments", middleware.RequireRoles(rolesWorker...), shipmentH.CreateShipment)

		api.GET("/inventory/products", middleware.RequireRoles(rolesWorker...), inventoryH.ListProducts)
		api.GET("/inventory/products/:id", middleware.RequireRoles(rolesWorker...), inventoryH.GetProduct)
		api.POST("/inventory/products", middleware.RequireRoles(rolesOperator...), inventoryH.CreateProduct)
		api.POST("/inventory/products/register", middleware.RequireRoles(rolesOperator...), inventoryH.RegisterProduct)
		api.PUT("/inventory/products/:id", middleware.RequireRoles(rolesOperator...), inventoryH.UpdateProduct)
		api.GET("/inventory/product-models/:code", middleware.RequireRoles(rolesWorker...), inventoryH.GetProductModel)
		api.POST("/inventory/product-models", middleware.RequireRoles(rolesOperator...), inventoryH.CreateProductModel)
		api.GET("/inventory/stocks", middleware.RequireRoles(rolesWorker...), inventoryH.ListStocks)
		api.POST("/inventory/stocks", middleware.RequireRoles(rolesOperator...), inventoryH.CreateComponentStock)
		api.GET("/inventory/stocks/:sku", middleware.RequireRoles(rolesWorker...), inventoryH.GetStockBySKU)
		api.GET("/inventory/metrics", middleware.RequireRoles(rolesWorker...), inventoryH.GetInventoryMetrics)
		api.GET("/inventory/export", middleware.RequireRoles(rolesWorker...), inventoryH.ExportInventoryReport)
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
		api.POST("/inventory/part-catalog", middleware.RequireRoles(rolesOperator...), inventoryH.CreatePartCatalog)
		api.PUT("/inventory/part-catalog/:sku", middleware.RequireRoles(rolesOperator...), inventoryH.UpdatePartCatalog)
		api.DELETE("/inventory/part-catalog/:sku", middleware.RequireRoles(rolesOperator...), inventoryH.DeletePartCatalog)
		api.GET("/inventory/part-catalog/sku/:sku", middleware.RequireRoles(rolesWorker...), inventoryH.GetPartCatalogBySKU)

		api.GET("/luts", middleware.RequireRoles(rolesWorker...), lutH.GetAllLUTs)

		api.GET("/inventory/ledger", middleware.RequireRoles(rolesWorker...), ledgerH.ListEntries)
		api.GET("/inventory/ledger/:id", middleware.RequireRoles(rolesWorker...), ledgerH.GetEntryByID)
	}

	slog.Info("scm service listening",
		slog.String("service", "scm"),
		slog.String("event", "server_starting"),
		slog.String("port", cfg.ServerPort),
	)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		slog.Error("server failed",
			slog.String("service", "scm"),
			slog.String("event", "server_failed"),
			slog.Any("error", err),
		)
		shutdownObs()
		os.Exit(1)
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
