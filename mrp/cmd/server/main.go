package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"zeus-mrp-service/configs"
	"zeus-mrp-service/internal/consumer"
	"zeus-mrp-service/internal/controllers"
	"zeus-mrp-service/internal/infrastructure/cache"
	"zeus-mrp-service/internal/infrastructure/cronjob"
	messaginginfra "zeus-mrp-service/internal/infrastructure/messaging"
	"zeus-mrp-service/internal/infrastructure/observability"
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

	// ── Observability: logger + metrics (must be first) ───────────────────────
	env := cfg.Env
	if env == "" {
		env = os.Getenv("APP_ENV")
	}
	obs, shutdownObs := observability.Setup(observability.Config{
		ServiceName:   "mrp",
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
		slog.Error("failed to open sqlite db", slog.String("service", "mrp"), slog.String("error", err.Error()))
		os.Exit(1)
	}

	scmClient := scminfra.NewClient()
	logStartupSCM(scmBaseURL())
	repo := reposqlite.NewSqliteMRPRepository(db)
	cacheRepo := repoValkey.NewWithClient(valkeyConn)
	svc := service.NewProductionService(repo, scmClient, cacheRepo, rabbitmq)

	orderConsumer := consumer.NewOrderConsumer(cfg.RabbitMQURL, svc)
	if err := orderConsumer.Start(context.Background()); err != nil {
		slog.Error("failed to start sales order consumer", slog.String("service", "mrp"), slog.String("error", err.Error()))
	} else {
		defer orderConsumer.Close()
	}

	authVerifier, err := middlewares.NewJWTVerifierFromFile(cfg.JwtPublicKeyPath)
	if err != nil {
		slog.Error("failed to initialize access-token verifier", slog.String("service", "mrp"), slog.String("error", err.Error()))
		os.Exit(1)
	}
	mux := controllers.NewMux(svc, authVerifier)

	r := gin.New()
	r.Use(observability.Tracing("mrp")) // inject trace_id / span_id
	r.Use(gin.CustomRecovery(func(c *gin.Context, recovered any) {
		slog.Error("gin recovery triggered",
			slog.String("service", "mrp"),
			slog.String("event", "panic"),
			slog.String("path", c.Request.URL.Path),
			slog.Any("error", recovered),
		)
		c.AbortWithStatus(http.StatusInternalServerError)
	}))
	r.Use(middlewares.AllowAllCORS())

	// Internal metrics endpoint — Alloy scrapes this.
	r.GET("/metrics", gin.WrapF(observability.MetricsHTTPHandler(obs.Metrics)))

	specPath := findOpenAPISpec()
	specURL := runtimeServerURL(cfg.Port)
	spec, err := loadOpenAPISpec(specPath, specURL)
	if err != nil {
		slog.Warn("could not load openapi spec", slog.String("service", "mrp"), slog.String("spec_path", specPath), slog.String("error", err.Error()))
	}

	r.GET("/docs/*any", openapiui.WrapHandler(openapiui.Config{
		Title:   "Zeus MRP API",
		SpecURL: "./openapi.json",
		SpecProvider: func() ([]byte, error) {
			if spec == nil {
				data, err := os.ReadFile(specPath)
				if err != nil {
					slog.Error("error reading openapi.yaml", slog.String("service", "mrp"), slog.String("spec_path", specPath), slog.String("error", err.Error()))
					return nil, err
				}
				var parsed any
				if err := yaml.Unmarshal(data, &parsed); err != nil {
					slog.Error("error parsing openapi.yaml", slog.String("service", "mrp"), slog.String("spec_path", specPath), slog.String("error", err.Error()))
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

	slog.Info("mrp service started", slog.String("service", "mrp"), slog.String("port", cfg.Port))
	if err := r.Run(":" + cfg.Port); err != nil {
		slog.Error("server error", slog.String("service", "mrp"), slog.String("error", err.Error()))
		shutdownObs()
		os.Exit(1)
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
		slog.Error("scm connection failed", slog.String("service", "mrp"), slog.String("component", "scm"), slog.String("error", err.Error()))
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("scm connection failed", slog.String("service", "mrp"), slog.String("component", "scm"), slog.String("base_url", baseURL), slog.String("error", err.Error()))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Error("scm connection failed", slog.String("service", "mrp"), slog.String("component", "scm"), slog.String("base_url", baseURL), slog.Int("status", resp.StatusCode))
		return
	}
	slog.Info("scm connection successful", slog.String("service", "mrp"), slog.String("component", "scm"), slog.String("base_url", baseURL))
}

func logStartupRabbitMQ(url string, rabbitmq *messaginginfra.RabbitMQ) {
	if url == "" {
		slog.Info("rabbitmq disabled", slog.String("service", "mrp"), slog.String("component", "rabbitmq"), slog.String("reason", "no_url_configured"))
		return
	}
	if rabbitmq == nil {
		slog.Error("rabbitmq connection failed", slog.String("service", "mrp"), slog.String("component", "rabbitmq"), slog.String("url", url), slog.String("error", "client is nil"))
		return
	}
	queues := []string{
		messaginginfra.AuditQueue,
		messaginginfra.DeficitPoolQueue,
		messaginginfra.SalesOrderCreatedQueue,
		messaginginfra.SalesOrderUpdatedQueue,
	}
	for _, queue := range queues {
		if err := rabbitmq.DeclareQueue(queue, true); err != nil {
			slog.Error("rabbitmq connection failed",
				slog.String("service", "mrp"),
				slog.String("component", "rabbitmq"),
				slog.String("url", url),
				slog.String("queue", queue),
				slog.String("error", err.Error()),
			)
			return
		}
	}
	slog.Info("rabbitmq connection successful", slog.String("service", "mrp"), slog.String("component", "rabbitmq"), slog.String("url", url))
}

func logStartupValkey(conn cache.ValkeyConn, addr string) {
	if addr == "" {
		slog.Info("valkey cache disabled", slog.String("service", "mrp"), slog.String("component", "valkey"), slog.String("reason", "no_address_configured"))
		return
	}
	if conn == nil {
		slog.Error("valkey connection failed", slog.String("service", "mrp"), slog.String("component", "valkey"), slog.String("addr", addr), slog.String("error", "client is nil"))
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := conn.Exists(ctx, "mrp:startup:probe"); err != nil {
		slog.Error("valkey connection failed", slog.String("service", "mrp"), slog.String("component", "valkey"), slog.String("addr", addr), slog.String("error", err.Error()))
		return
	}
	slog.Info("valkey connection successful", slog.String("service", "mrp"), slog.String("component", "valkey"), slog.String("addr", addr))
}
