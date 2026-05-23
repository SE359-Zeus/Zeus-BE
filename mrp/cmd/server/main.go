package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
	"zeus-mrp-service/configs"
	"zeus-mrp-service/internal/controllers"
	"zeus-mrp-service/internal/middlewares"
	reposqlite "zeus-mrp-service/internal/repository/sqlite"
	"zeus-mrp-service/internal/service"

	"context"

	openapiui "github.com/PeterTakahashi/gin-openapi/openapiui"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gopkg.in/yaml.v3"
)

func probeValkey(addr string) {
	if addr == "" {
		log.Println("Valkey probe skipped: no address configured")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	client := redis.NewClient(&redis.Options{Addr: addr})
	defer client.Close()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Valkey connection failed at %s: %v", addr, err)
		return
	}

	log.Printf("Valkey connection successful at %s", addr)
}

func main() {
	cfg := configs.Load()
	probeValkey(cfg.ValkeyAddr)

	// Open sqlite DB (creates file if not present)
	dbPath := os.Getenv("MRP_DB_PATH")
	if dbPath == "" {
		dbPath = "mrp.db"
	}
	db, err := reposqlite.OpenDatabase(dbPath)
	if err != nil {
		log.Fatalf("failed to open sqlite db: %v", err)
	}

	repo := reposqlite.NewSqliteMRPRepository(db)
	svc := service.NewProductionService(repo)
	mux := controllers.NewMux(svc)

	r := gin.Default()

	specPath := findOpenAPISpec()
	specURL := runtimeServerURL(cfg.Port)
	spec, err := loadOpenAPISpec(specPath, specURL)
	if err != nil {
		log.Printf("warning: could not load openapi spec at %s: %v", specPath, err)
	}

	r.GET("/docs/*any", openapiui.WrapHandler(openapiui.Config{
		Title:   "Zeus MRP API",
		SpecURL: "./openapi.json",
		SpecProvider: func() ([]byte, error) {
			if spec == nil {
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
					specMap["servers"] = []any{map[string]any{"url": specURL}}
				}
				return json.Marshal(parsed)
			}
			return spec, nil
		},
		Theme: "dark",
	}))

	r.Any("/api/v1/mrp/*path", gin.WrapF(func(w http.ResponseWriter, r *http.Request) {
		middlewares.ErrorHandler(mux).ServeHTTP(w, r)
	}))

	log.Printf("Zeus MRP Service running on :%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

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

func runtimeServerURL(port string) string {
	if port == "" {
		port = "8081"
	}
	return fmt.Sprintf("http://localhost:%s/api/v1/mrp", port)
}
