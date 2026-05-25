package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

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
	logStartupValkey(valkeyConn, cfg.ValkeyAddr)
	rabbitmq := messaginginfra.NewRabbitMQ(cfg.RabbitMQURL)
	logStartupRabbitMQ(cfg.RabbitMQURL, rabbitmq)

	dbPath := os.Getenv("MRP_DB_PATH")
	if dbPath == "" {
		dbPath = "mrp.db"
	}
	db, err := reposqlite.OpenDatabase(dbPath)
	if err != nil {
		log.Fatalf("failed to open sqlite db: %v", err)
	}

	scmClient := scminfra.NewClient()
	logStartupSCM(scmBaseURL())
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

func scmBaseURL() string {
	baseURL := os.Getenv("SCM_BASE_URL")
	if baseURL == "" {
		return "http://localhost:8083"
	}
	return baseURL
}

func logStartupSCM(baseURL string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/health", nil)
	if err != nil {
		log.Printf("SCM connection failed: %v", err)
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("SCM connection failed at %s: %v", baseURL, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("SCM connection failed at %s: status %d", baseURL, resp.StatusCode)
		return
	}
	log.Printf("SCM connection successful at %s", baseURL)
}

func logStartupRabbitMQ(url string, rabbitmq *messaginginfra.RabbitMQ) {
	if url == "" {
		log.Println("RabbitMQ disabled: no URL configured")
		return
	}
	if rabbitmq == nil {
		log.Printf("RabbitMQ connection failed at %s: client is nil", url)
		return
	}
	if err := rabbitmq.DeclareQueue(messaginginfra.AuditQueue, true); err != nil {
		log.Printf("RabbitMQ connection failed at %s: %v", url, err)
		return
	}
	log.Printf("RabbitMQ connection successful at %s", url)
}

func logStartupValkey(conn cache.ValkeyConn, addr string) {
	if addr == "" {
		log.Println("Valkey cache disabled: no address configured")
		return
	}
	if conn == nil {
		log.Printf("Valkey connection failed at %s: client is nil", addr)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := conn.Exists(ctx, "mrp:startup:probe"); err != nil {
		log.Printf("Valkey connection failed at %s: %v", addr, err)
		return
	}
	log.Printf("Valkey connection successful at %s", addr)
}
