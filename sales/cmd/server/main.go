package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"zeus-sales-service/config"
	"zeus-sales-service/internal/controllers"
	infraCache "zeus-sales-service/internal/infrastructure/cache"
	infraMessaging "zeus-sales-service/internal/infrastructure/messaging"
	"zeus-sales-service/internal/middlewares"
	"zeus-sales-service/internal/repository/sqlite"
	"zeus-sales-service/internal/repository/valkey"
	"zeus-sales-service/internal/service"

	openapiui "github.com/PeterTakahashi/gin-openapi/openapiui"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

func main() {
	setupLogger()
	cfg := config.Load()

	sqliteRepo, err := sqlite.Open(cfg.SQLiteDBPath)
	if err != nil {
		slog.Error("failed to open sqlite database", slog.String("service", "sales"), slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer sqliteRepo.Close()

	valkeyRepo := valkey.New(cfg.ValkeyAddr)
	salesCache := infraCache.NewStore(cfg.ValkeyAddr)
	var publisher infraMessaging.Publisher
	if err := infraCache.New(cfg.ValkeyAddr).Ping(context.Background()); err != nil {
		slog.Warn("valkey connection failed", slog.String("service", "sales"), slog.String("component", "valkey"), slog.String("error", err.Error()))
	} else {
		slog.Info("valkey connection successful", slog.String("service", "sales"), slog.String("component", "valkey"))
	}
	if rabbitmq, err := infraMessaging.NewRabbitMQ(cfg.RabbitMQURL); err != nil {
		slog.Warn("sales messaging disabled", slog.String("service", "sales"), slog.String("component", "rabbitmq"), slog.String("error", err.Error()))
	} else if err := rabbitmq.Ping(context.Background()); err != nil {
		slog.Warn("rabbitmq connection failed", slog.String("service", "sales"), slog.String("component", "rabbitmq"), slog.String("error", err.Error()))
	} else {
		slog.Info("rabbitmq connection successful", slog.String("service", "sales"), slog.String("component", "rabbitmq"))
		publisher = rabbitmq
	}
	infra := service.NewInfrastructure(salesCache, publisher)

	services := service.NewServices(sqliteRepo, valkeyRepo, infra)
	authVerifier, err := middlewares.NewJWTVerifierFromFile(cfg.JwtPublicKeyPath)
	if err != nil {
		slog.Error("failed to initialize access-token verifier", slog.String("service", "sales"), slog.String("error", err.Error()))
		os.Exit(1)
	}
	mux := controllers.NewMux(services, authVerifier)

	r := gin.New()
	r.Use(gin.CustomRecovery(func(c *gin.Context, recovered any) {
		slog.Error("gin recovery triggered",
			slog.String("service", "sales"),
			slog.String("event", "panic"),
			slog.String("path", c.Request.URL.Path),
			slog.Any("error", recovered),
		)
		c.AbortWithStatus(http.StatusInternalServerError)
	}))
	r.Use(middlewares.AllowAllCORS())

	// Load OpenAPI spec
	specPath := findOpenAPISpec()
	specURL := runtimeServerURL(cfg.BaseURL, cfg.Port)
	spec, err := loadOpenAPISpec(specPath, specURL)
	if err != nil {
		slog.Warn("could not load openapi spec", slog.String("service", "sales"), slog.String("spec_path", specPath), slog.String("error", err.Error()))
	}

	// Serve OpenAPI UI at /docs/*any
	r.GET("/docs/*any", openapiui.WrapHandler(openapiui.Config{
		Title:   "Zeus Sales API",
		SpecURL: "./openapi.json",
		SpecProvider: func() ([]byte, error) {
			if spec == nil {
				// Fallback: try to load on-demand if it wasn't loaded at startup
				data, err := os.ReadFile(specPath)
				if err != nil {
					slog.Error("error reading openapi.yaml", slog.String("service", "sales"), slog.String("spec_path", specPath), slog.String("error", err.Error()))
					return nil, err
				}
				var parsed any
				if err := yaml.Unmarshal(data, &parsed); err != nil {
					slog.Error("error parsing openapi.yaml", slog.String("service", "sales"), slog.String("spec_path", specPath), slog.String("error", err.Error()))
					return nil, err
				}
				if specMap, ok := parsed.(map[string]any); ok {
					specMap["servers"] = []any{
						map[string]any{"url": specURL},
					}
				}
				return json.Marshal(parsed)
			}
			return spec, nil
		},
		Theme: "dark",
	}))

	// Mount the net/http mux (with API routes) at /api/v1/sales
	r.Any("/api/v1/sales/*path", gin.WrapF(func(w http.ResponseWriter, r *http.Request) {
		middlewares.ErrorHandler(mux).ServeHTTP(w, r)
	}))

	slog.Info("sales service started", slog.String("service", "sales"), slog.String("port", cfg.Port))
	if err := r.Run(":" + cfg.Port); err != nil {
		slog.Error("server error", slog.String("service", "sales"), slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func setupLogger() {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(handler))
}

// findOpenAPISpec locates the openapi.yaml file by trying multiple paths
func findOpenAPISpec() string {
	paths := []string{
		"docs/openapi.yaml",
		"./docs/openapi.yaml",
		filepath.Join(".", "docs", "openapi.yaml"),
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	// Default to relative path if none found
	return "docs/openapi.yaml"
}

// loadOpenAPISpec loads and parses the OpenAPI specification file
func loadOpenAPISpec(specPath, serverURL string) ([]byte, error) {
	data, err := os.ReadFile(specPath)
	if err != nil {
		return nil, err
	}

	var parsed map[string]any
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		return nil, err
	}
	parsed["servers"] = []any{
		map[string]any{"url": serverURL},
	}

	return json.Marshal(parsed)
}

func runtimeServerURL(baseURL, port string) string {
	return "/api/v1/sales"
}
