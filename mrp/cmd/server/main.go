package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"zeus-mrp-service/configs"
	"zeus-mrp-service/internal/controllers"
	"zeus-mrp-service/internal/infrastructure/cache"
	messaginginfra "zeus-mrp-service/internal/infrastructure/messaging"
	scminfra "zeus-mrp-service/internal/infrastructure/scm"
	"zeus-mrp-service/internal/middlewares"
	reposqlite "zeus-mrp-service/internal/repository/sqlite"
	repoValkey "zeus-mrp-service/internal/repository/valkey"
	"zeus-mrp-service/internal/service"

	openapiui "github.com/PeterTakahashi/gin-openapi/openapiui"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

func main() {
	cfg := configs.Load()
	valkeyConn := cache.DialValkey(cfg.ValkeyAddr)
	rabbitmq := messaginginfra.NewRabbitMQ(cfg.RabbitMQURL)
	if err := rabbitmq.DeclareQueue(messaginginfra.AuditQueue, true); err != nil {
		log.Printf("warning: failed to declare audit queue: %v", err)
	}

	dbPath := os.Getenv("MRP_DB_PATH")
	if dbPath == "" {
		dbPath = "mrp.db"
	}
	db, err := reposqlite.OpenDatabase(dbPath)
	if err != nil {
		log.Fatalf("failed to open sqlite db: %v", err)
	}

	scmClient := scminfra.NewClient()
	repo := reposqlite.NewSqliteMRPRepository(db)
	cacheRepo := repoValkey.NewWithClient(valkeyConn)
	svc := service.NewProductionService(repo, scmClient, cacheRepo, rabbitmq)
	authVerifier, err := middlewares.NewJWTVerifierFromFile(cfg.JwtPublicKeyPath)
	if err != nil {
		log.Fatalf("failed to initialize access-token verifier: %v", err)
	}
	mux := controllers.NewMux(svc, authVerifier)

	r := gin.Default()
	r.Use(middlewares.AllowAllCORS())

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
	return "/api/v1/mrp"
}
