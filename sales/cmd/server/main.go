package main

import (
	"context"
	"encoding/json"
	"log"
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
	cfg := config.Load()

	sqliteRepo, err := sqlite.Open(cfg.SQLiteDBPath)
	if err != nil {
		log.Fatal(err)
	}
	defer sqliteRepo.Close()

	valkeyRepo := valkey.New(cfg.ValkeyAddr)
	salesCache := infraCache.NewStore(cfg.ValkeyAddr)
	var publisher infraMessaging.Publisher
	if err := infraCache.New(cfg.ValkeyAddr).Ping(context.Background()); err != nil {
		log.Printf("warning: valkey connection failed: %v", err)
	} else {
		log.Printf("valkey connection successful")
	}
	if rabbitmq, err := infraMessaging.NewRabbitMQ(cfg.RabbitMQURL); err != nil {
		log.Printf("warning: sales messaging disabled: %v", err)
	} else if err := rabbitmq.Ping(context.Background()); err != nil {
		log.Printf("warning: rabbitmq connection failed: %v", err)
	} else {
		log.Printf("rabbitmq connection successful")
		publisher = rabbitmq
	}
	infra := service.NewInfrastructure(salesCache, publisher)

	services := service.NewServices(sqliteRepo, valkeyRepo, infra)
	authVerifier, err := middlewares.NewJWTVerifierFromFile(cfg.JwtPublicKeyPath)
	if err != nil {
		log.Fatal(err)
	}
	mux := controllers.NewMux(services, authVerifier)

	// Create main gin engine
	r := gin.Default()
	r.Use(middlewares.AllowAllCORS())

	// Load OpenAPI spec
	specPath := findOpenAPISpec()
	specURL := runtimeServerURL(cfg.BaseURL, cfg.Port)
	spec, err := loadOpenAPISpec(specPath, specURL)
	if err != nil {
		log.Printf("warning: could not load openapi spec at %s: %v", specPath, err)
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
					log.Printf("error reading openapi.yaml: %v", err)
					return nil, err
				}
				var parsed any
				if err := yaml.Unmarshal(data, &parsed); err != nil {
					log.Printf("error parsing openapi.yaml: %v", err)
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

	log.Printf("Zeus Sales Service running on :%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
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
