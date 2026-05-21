package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"zeus-sales-service/config"
	"zeus-sales-service/internal/controllers"
	"zeus-sales-service/internal/middlewares"
	"zeus-sales-service/internal/repository/sqlite"
	"zeus-sales-service/internal/repository/valkey"
	"zeus-sales-service/internal/service"

	openapiui "github.com/PeterTakahashi/gin-openapi/openapiui"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gopkg.in/yaml.v3"
)

func main() {
	cfg := config.Load()

	sqliteRepo, err := sqlite.Open(cfg.SQLiteDBPath)
	if err != nil {
		log.Fatal(err)
	}
	defer sqliteRepo.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: cfg.ValkeyAddr})
	defer redisClient.Close()
	valkeyRepo := valkey.New(redisClient)

	services := service.NewServices(sqliteRepo, valkeyRepo)
	mux := controllers.NewMux(services)

	// Create main gin engine
	r := gin.Default()

	// Load OpenAPI spec
	specPath := findOpenAPISpec()
	specURL := runtimeServerURL(cfg.BaseURL, cfg.Port)
	spec, err := loadOpenAPISpec(specPath, specURL)
	if err != nil {
		log.Printf("warning: could not load openapi spec at %s: %v", specPath, err)
	}

	// Serve OpenAPI UI at /docs/*any
	r.GET("api/v1/sales/docs/*any", openapiui.WrapHandler(openapiui.Config{
		Title: "Zeus Sales API",
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
	if strings.TrimSpace(port) == "" {
		port = "8082"
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Sprintf("http://localhost:%s/api/v1/sales", port)
	}
	hostname := parsed.Hostname()
	if hostname == "" {
		hostname = "localhost"
	}
	parsed.Host = net.JoinHostPort(hostname, port)
	parsed.Path = "/api/v1/sales"
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}
