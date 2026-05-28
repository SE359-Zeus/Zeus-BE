package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"zeus-system-service/internal/config"
	"zeus-system-service/internal/consumer"
	"zeus-system-service/internal/handler"
	"zeus-system-service/internal/handler/middleware"
	"zeus-system-service/internal/infrastructure/cache"
	"zeus-system-service/internal/infrastructure/cronjob"
	"zeus-system-service/internal/infrastructure/messaging"
	"zeus-system-service/internal/infrastructure/observability"
	"zeus-system-service/internal/repository/sqlite"
	valkeyRepo "zeus-system-service/internal/repository/valkey"
	"zeus-system-service/internal/service"

	openapiui "github.com/PeterTakahashi/gin-openapi/openapiui"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

func dialValkey(addr string) func() (cache.ValkeyConn, error) {
	return func() (cache.ValkeyConn, error) {
		return cache.DialValkey(addr)
	}
}

func checkValkeyConnection(addr string) {
	conn, err := cache.DialValkey(addr)
	if err != nil {
		slog.Warn("valkey connection failed",
			slog.String("service", "system"),
			slog.String("event", "dependency_unavailable"),
			slog.String("component", "valkey"),
			slog.String("addr", addr),
			slog.Any("error", err),
		)
		return
	}
	defer conn.Close()

	if _, err := conn.Exists(context.Background(), "system:healthcheck"); err != nil {
		slog.Warn("valkey health check failed",
			slog.String("service", "system"),
			slog.String("event", "dependency_unhealthy"),
			slog.String("component", "valkey"),
			slog.String("addr", addr),
			slog.Any("error", err),
		)
		return
	}

	slog.Info("valkey connection successful",
		slog.String("service", "system"),
		slog.String("event", "dependency_ready"),
		slog.String("component", "valkey"),
		slog.String("addr", addr),
	)
}

func checkRabbitMQConnection(url string) {
	conn, err := messaging.Dial(url)
	if err != nil {
		slog.Warn("rabbitmq connection failed",
			slog.String("service", "system"),
			slog.String("event", "dependency_unavailable"),
			slog.String("component", "rabbitmq"),
			slog.String("url", url),
			slog.Any("error", err),
		)
		return
	}
	conn.Close()

	slog.Info("rabbitmq connection successful",
		slog.String("service", "system"),
		slog.String("event", "dependency_ready"),
		slog.String("component", "rabbitmq"),
		slog.String("url", url),
	)
}

func loadPrivateKey(path string) *rsa.PrivateKey {
	if path == "" {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			slog.Error("failed to generate dev RSA key",
				slog.String("service", "system"),
				slog.String("event", "startup_failed"),
				slog.String("component", "jwt_key"),
				slog.Any("error", err),
			)
			os.Exit(1)
		}
		slog.Warn("using ephemeral rsa key (dev mode)",
			slog.String("service", "system"),
			slog.String("event", "dev_mode_key"),
		)
		return key
	}
	data, err := os.ReadFile(path)
	if err != nil {
		slog.Error("failed to read private key",
			slog.String("service", "system"),
			slog.String("event", "startup_failed"),
			slog.String("component", "jwt_key"),
			slog.String("path", path),
			slog.Any("error", err),
		)
		os.Exit(1)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		slog.Error("failed to decode PEM block",
			slog.String("service", "system"),
			slog.String("event", "startup_failed"),
			slog.String("component", "jwt_key"),
			slog.String("path", path),
		)
		os.Exit(1)
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			slog.Error("failed to parse private key",
				slog.String("service", "system"),
				slog.String("event", "startup_failed"),
				slog.String("component", "jwt_key"),
				slog.String("path", path),
				slog.Any("error", err),
			)
			os.Exit(1)
		}
	}
	return key.(*rsa.PrivateKey)
}

func buildOpenAPISpec(serverPort string) func() ([]byte, error) {
	return func() ([]byte, error) {
		data, err := os.ReadFile("docs/openapi.yaml")
		if err != nil {
			return nil, err
		}

		var parsed map[string]any
		if err := yaml.Unmarshal(data, &parsed); err != nil {
			return nil, err
		}

		parsed["servers"] = []map[string]string{{
			"url":         runtimeServerURL(serverPort),
			"description": "Local development",
		}}

		return json.Marshal(parsed)
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
	// PUBLIC_BASE_URL must be set to the host only (no path) in stack.env:
	//   PUBLIC_BASE_URL=https://zeus.ryanandexen.qzz.io
	// Swagger UI calls: PUBLIC_BASE_URL + /api/v1/system + <path-from-spec>
	// Locally (unset), falls back to http://localhost:<port>.
	base := os.Getenv("PUBLIC_BASE_URL")
	if base == "" {
		if port == "" {
			port = "8084"
		}
		base = "http://localhost:" + port
	}
	return base + "/api/v1/system"
}

func main() {
	cfg := config.Load()

	// ── Observability: logger + metrics (must be first) ───────────────────────
	env := cfg.Env
	if env == "" {
		env = os.Getenv("APP_ENV")
	}
	obs, shutdownObs := observability.Setup(observability.Config{
		ServiceName:   "system",
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

	slog.Info("starting system service",
		slog.String("service", "system"),
		slog.String("event", "startup"),
		slog.String("db_path", cfg.DBPath),
		slog.String("valkey_addr", cfg.ValkeyAddr),
		slog.String("rabbitmq_url", cfg.RabbitMQURL),
	)

	checkValkeyConnection(cfg.ValkeyAddr)
	checkRabbitMQConnection(cfg.RabbitMQURL)

	db, err := sqlite.NewDB(cfg.DBPath)
	if err != nil {
		slog.Error("failed to connect to database",
			slog.String("service", "system"),
			slog.String("event", "startup_failed"),
			slog.String("component", "database"),
			slog.Any("error", err),
		)
		os.Exit(1)
	}

	if err := sqlite.ApplyMigrations(db, "./migrations", sqlite.DirectionUp); err != nil {
		slog.Error("migration failed",
			slog.String("service", "system"),
			slog.String("event", "startup_failed"),
			slog.String("component", "migration"),
			slog.Any("error", err),
		)
		os.Exit(1)
	}

	userRepo := sqlite.NewUserRepository(db)
	auditRepo := sqlite.NewAuditRepository(db)
	roleRepo := sqlite.NewRoleRepository(db)
	actionTypeRepo := sqlite.NewActionTypeRepository(db)

	vkDialer := dialValkey(cfg.ValkeyAddr)

	refreshTokenRepo := valkeyRepo.NewRefreshTokenRepository(vkDialer)
	actionTypeCacheRepo := valkeyRepo.NewActionTypeCacheRepository(vkDialer)

	rbacSvc := service.NewEndpointRBACService(roleRepo)
	if err := rbacSvc.WarmCache(context.Background()); err != nil {
		slog.Warn("rbac cache warm failed",
			slog.String("service", "system"),
			slog.String("event", "cache_warm_failed"),
			slog.String("component", "rbac"),
			slog.Any("error", err),
		)
	}

	actionTypeSvc := service.NewActionTypeService(actionTypeRepo, actionTypeCacheRepo)
	if err := actionTypeSvc.WarmCache(context.Background()); err != nil {
		slog.Warn("action type cache warm failed",
			slog.String("service", "system"),
			slog.String("event", "cache_warm_failed"),
			slog.String("component", "action_type"),
			slog.Any("error", err),
		)
	}

	emailSvc, err := service.NewSMTPEmailService(
		cfg.SMTPHost,
		cfg.SMTPPort,
		cfg.SMTPUser,
		cfg.SMTPPass,
		cfg.EmailFromAddress,
		cfg.EmailTemplateDir,
	)
	if err != nil {
		slog.Warn("account email sender disabled",
			slog.String("service", "system"),
			slog.String("event", "dependency_unavailable"),
			slog.String("component", "smtp"),
			slog.Any("error", err),
		)
	}

	sessionRepo := sqlite.NewSessionRepository(db)

	userCacheRepo := valkeyRepo.NewUserCacheRepository(vkDialer)
	userSvc := service.NewUserService(userRepo, rbacSvc, emailSvc, userCacheRepo)
	privateKey := loadPrivateKey(cfg.JWTKeyPath)
	authSvc := service.NewAuthService(userSvc, refreshTokenRepo, sessionRepo, privateKey)
	auditSvc := service.NewAuditService(auditRepo, actionTypeSvc)

	auditConsumer := consumer.NewAuditConsumer(cfg.RabbitMQURL, auditSvc)
	if err := auditConsumer.Start(context.Background()); err != nil {
		slog.Warn("failed to start audit rabbitmq consumer",
			slog.String("service", "system"),
			slog.String("event", "consumer_start_failed"),
			slog.String("component", "audit_consumer"),
			slog.Any("error", err),
		)
	}

	sessions, err := sessionRepo.ListActive(context.Background())
	if err != nil {
		slog.Warn("failed to load sessions for valkey warm-up",
			slog.String("service", "system"),
			slog.String("event", "cache_warm_failed"),
			slog.String("component", "refresh_token"),
			slog.Any("error", err),
		)
	} else {
		for _, s := range sessions {
			if err := refreshTokenRepo.SaveRefreshToken(context.Background(), s.JTI, s.UserID.String()); err != nil {
				slog.Warn("failed to warm refresh token",
					slog.String("service", "system"),
					slog.String("event", "cache_warm_failed"),
					slog.String("component", "refresh_token"),
					slog.String("jti", s.JTI),
					slog.Any("error", err),
				)
			}
		}
		slog.Info("warmed sessions into valkey",
			slog.String("service", "system"),
			slog.String("event", "cache_warm_complete"),
			slog.Int("session_count", len(sessions)),
		)
	}

	authH := handler.NewAuthHandler(authSvc, auditSvc)
	userH := handler.NewUserHandler(userSvc, auditSvc)
	auditH := handler.NewAuditHandler(auditSvc)

	r := gin.New()
	// Disable Gin's trailing-slash redirect: prevents a 301 /docs → /docs/
	// from escaping the nginx /system/docs/ proxy prefix in production.
	r.RedirectTrailingSlash = false
	r.Use(
		middleware.CORS(),
		observability.Tracing("system"), // inject trace_id / span_id
		middleware.RequestLogger(),
		middleware.Recovery(),
	)

	// Internal metrics endpoint — Alloy scrapes this.
	r.GET("/metrics", gin.WrapF(observability.MetricsHTTPHandler(obs.Metrics)))

	specPath := findOpenAPISpec()
	specURL := runtimeServerURL(cfg.ServerPort)
	spec, err := loadOpenAPISpec(specPath, specURL)
	if err != nil {
		slog.Warn("could not load openapi spec",
			slog.String("service", "system"),
			slog.String("event", "openapi_load_failed"),
			slog.String("path", specPath),
			slog.Any("error", err),
		)
	}

	r.GET("/docs/*any", openapiui.WrapHandler(openapiui.Config{
		Title:   "Zeus System API",
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
		handler.WriteEnvelope(c, 200, "ok", gin.H{}, gin.H{"status": "ok"})
	})

	systemPublic := r.Group("/api/v1/system")
	{
		systemPublic.POST("/auth/login", authH.Login)
		systemPublic.POST("/auth/refresh", authH.Refresh)
		systemPublic.POST("/auth/logout", authH.Logout)
	}

	api := r.Group("/api/v1/system", middleware.JWTAuth(authSvc))
	{
		api.POST("/auth/change-password", authH.ChangePassword)
		api.POST("/users", middleware.RequireRoles("admin"), userH.Create)
		api.GET("/users", middleware.RequireRoles("admin"), userH.List)
		api.GET("/users/:id", middleware.RequireRoles("admin"), userH.GetByID)
		api.PUT("/users/:id", middleware.RequireRoles("admin"), userH.Update)
		api.PATCH("/users/:id/status", middleware.RequireRoles("admin"), userH.SetStatus)

		api.POST("/logs/ingest", middleware.RequireRoles("admin"), auditH.Ingest)
		api.GET("/logs", middleware.RequireRoles("admin"), auditH.Query)
		api.GET("/logs/metrics", middleware.RequireRoles("admin"), auditH.GetMetrics)
	}

	slog.Info("system service listening",
		slog.String("service", "system"),
		slog.String("event", "server_starting"),
		slog.String("port", cfg.ServerPort),
	)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		slog.Error("server failed",
			slog.String("service", "system"),
			slog.String("event", "server_failed"),
			slog.Any("error", err),
		)
		shutdownObs()
		os.Exit(1)
	}
}
