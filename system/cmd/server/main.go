package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"log"
	"os"

	"zeus-system-service/internal/cache"
	"zeus-system-service/internal/config"
	"zeus-system-service/internal/handler"
	"zeus-system-service/internal/handler/middleware"
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

func loadPrivateKey(path string) *rsa.PrivateKey {
	if path == "" {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			log.Fatalf("failed to generate dev RSA key: %v", err)
		}
		log.Println("using ephemeral RSA key (dev mode)")
		return key
	}
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("failed to read private key: %v", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		log.Fatalf("failed to decode PEM block from %s", path)
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			log.Fatalf("failed to parse private key: %v", err)
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
			"url":         "http://localhost:" + serverPort + "/api/v1/system",
			"description": "Local development",
		}}

		return json.Marshal(parsed)
	}
}

func main() {
	cfg := config.Load()

	db, err := sqlite.NewDB(cfg.DBPath)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	if err := sqlite.ApplyMigrations(db, "./migrations", sqlite.DirectionUp); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	userRepo := sqlite.NewUserRepository(db)
	auditRepo := sqlite.NewAuditRepository(db)
	roleRepo := sqlite.NewRoleRepository(db)
	actionTypeRepo := sqlite.NewActionTypeRepository(db)
	endpointRoleRepo := sqlite.NewEndpointRoleRepository(db)

	vkDialer := dialValkey(cfg.ValkeyAddr)

	refreshTokenRepo := valkeyRepo.NewRefreshTokenRepository(vkDialer)
	actionTypeCacheRepo := valkeyRepo.NewActionTypeCacheRepository(vkDialer)
	endpointRBACCacheRepo := valkeyRepo.NewEndpointRBACCacheRepository(vkDialer)

	rbacSvc := service.NewEndpointRBACService(roleRepo, endpointRoleRepo, endpointRBACCacheRepo)
	if err := rbacSvc.WarmCache(context.Background()); err != nil {
		log.Printf("Warning: RBAC cache warm failed: %v", err)
	}

	actionTypeSvc := service.NewActionTypeService(actionTypeRepo, actionTypeCacheRepo)
	if err := actionTypeSvc.WarmCache(context.Background()); err != nil {
		log.Printf("Warning: action type cache warm failed: %v", err)
	}

	sessionRepo := sqlite.NewSessionRepository(db)

	userSvc := service.NewUserService(userRepo, rbacSvc)
	privateKey := loadPrivateKey(cfg.JWTKeyPath)
	authSvc := service.NewAuthService(userSvc, refreshTokenRepo, sessionRepo, privateKey)
	auditSvc := service.NewAuditService(auditRepo, actionTypeSvc)

	sessions, err := sessionRepo.ListActive(context.Background())
	if err != nil {
		log.Printf("Warning: failed to load sessions for Valkey warm-up: %v", err)
	} else {
		for _, s := range sessions {
			if err := refreshTokenRepo.SaveRefreshToken(context.Background(), s.JTI, s.UserID.String()); err != nil {
				log.Printf("Warning: failed to warm refresh token %s: %v", s.JTI, err)
			}
		}
		log.Printf("warmed %d sessions into Valkey", len(sessions))
	}

	authH := handler.NewAuthHandler(authSvc)
	userH := handler.NewUserHandler(userSvc)
	auditH := handler.NewAuditHandler(auditSvc)

	r := gin.New()
	r.Use(gin.Logger(), middleware.Recovery())

	r.GET("/docs/*any", openapiui.WrapHandler(openapiui.Config{
		Title:        "Zeus System API",
		SpecProvider: buildOpenAPISpec(cfg.ServerPort),
		Theme:        "dark",
	}))
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	systemPublic := r.Group("/api/v1/system")
	{
		systemPublic.POST("/auth/login", authH.Login)
		systemPublic.POST("/auth/refresh", authH.Refresh)
		systemPublic.POST("/auth/logout", authH.Logout)
	}

	api := r.Group("/api/v1/system")
	api.Use(middleware.JWTAuth(authSvc), middleware.RequireRoleLevel(rbacSvc))
	{
		api.POST("/users", userH.Create)
		api.GET("/users", userH.List)
		api.GET("/users/:id", userH.GetByID)
		api.PUT("/users/:id", userH.Update)
		api.PATCH("/users/:id/status", userH.SetStatus)

		api.POST("/logs/ingest", auditH.Ingest)
		api.GET("/logs", auditH.Query)
		api.GET("/logs/metrics", auditH.GetMetrics)
	}

	log.Printf("Zeus System service starting on :%s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
