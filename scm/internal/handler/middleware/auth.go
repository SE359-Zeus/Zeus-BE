package middleware

import (
	"log/slog"
	"strings"
	"time"

	"zeus-scm-service/internal/exception"
	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/service"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	ContextKeyUserID     = "user_id"
	ContextKeyRole       = "role"
	ContextKeyEmail      = "email"
	ContextKeyFullName   = "full_name"
	ContextKeyStatus     = "status"
	ContextKeyAuthMethod = "auth_method"
)

func Authenticate(jwtSvc *service.JWTService, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		apiKey := c.GetHeader("X-API-KEY")

		if apiKey != "" {
			if len(apiKey) < 8 {
				slog.Warn("api key rejected",
					slog.String("service", "scm"),
					slog.String("event", "auth_rejected"),
					slog.String("auth_method", "api_key"),
					slog.String("reason", "invalid_length"),
					slog.String("method", c.Request.Method),
					slog.String("path", c.Request.URL.Path),
				)
				exception.WriteError(c, exception.ErrAPIKeyInvalid)
				return
			}
			prefix := apiKey[:8]

			var key models.ApiKey
			if err := db.Where("key_prefix = ? AND active = ? AND deleted_at IS NULL", prefix, true).First(&key).Error; err != nil {
				slog.Warn("api key rejected",
					slog.String("service", "scm"),
					slog.String("event", "auth_rejected"),
					slog.String("auth_method", "api_key"),
					slog.String("reason", "lookup_failed"),
					slog.String("method", c.Request.Method),
					slog.String("path", c.Request.URL.Path),
					slog.Any("error", err),
				)
				exception.WriteError(c, exception.ErrAPIKeyInvalid)
				return
			}

			if key.ExpiresAt != nil && time.Now().After(*key.ExpiresAt) {
				slog.Warn("api key rejected",
					slog.String("service", "scm"),
					slog.String("event", "auth_rejected"),
					slog.String("auth_method", "api_key"),
					slog.String("reason", "expired"),
					slog.String("method", c.Request.Method),
					slog.String("path", c.Request.URL.Path),
				)
				exception.WriteError(c, exception.ErrAPIKeyExpired)
				return
			}

			if err := bcrypt.CompareHashAndPassword([]byte(key.KeyHash), []byte(apiKey)); err != nil {
				slog.Warn("api key rejected",
					slog.String("service", "scm"),
					slog.String("event", "auth_rejected"),
					slog.String("auth_method", "api_key"),
					slog.String("reason", "hash_mismatch"),
					slog.String("method", c.Request.Method),
					slog.String("path", c.Request.URL.Path),
				)
				exception.WriteError(c, exception.ErrAPIKeyInvalid)
				return
			}

			now := time.Now()
			db.Model(&key).Update("last_used_at", &now)

			c.Set("api_key_id", key.ID.String())
			c.Set("api_key_name", key.Name)
			c.Set(ContextKeyAuthMethod, "api_key")
			c.Set(ContextKeyRole, "api_key")
			c.Set(ContextKeyUserID, key.ID)
			c.Next()
			return
		}

		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				slog.Warn("bearer token rejected",
					slog.String("service", "scm"),
					slog.String("event", "auth_rejected"),
					slog.String("auth_method", "jwt"),
					slog.String("reason", "invalid_header"),
					slog.String("method", c.Request.Method),
					slog.String("path", c.Request.URL.Path),
				)
				exception.WriteError(c, exception.ErrInvalidAuthHeader)
				return
			}

			claims, err := jwtSvc.VerifyAccessToken(parts[1])
			if err != nil {
				slog.Warn("bearer token rejected",
					slog.String("service", "scm"),
					slog.String("event", "auth_rejected"),
					slog.String("auth_method", "jwt"),
					slog.String("reason", "invalid_token"),
					slog.String("method", c.Request.Method),
					slog.String("path", c.Request.URL.Path),
					slog.Any("error", err),
				)
				exception.WriteError(c, exception.ErrInvalidToken)
				return
			}

			c.Set(ContextKeyUserID, claims.UserID)
			c.Set(ContextKeyRole, claims.Role)
			c.Set(ContextKeyEmail, claims.Email)
			c.Set(ContextKeyFullName, claims.FullName)
			c.Set(ContextKeyStatus, claims.Status)
			c.Set(ContextKeyAuthMethod, "jwt")
			c.Next()
			return
		}

		slog.Warn("missing authentication",
			slog.String("service", "scm"),
			slog.String("event", "auth_rejected"),
			slog.String("reason", "missing_credentials"),
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
		)
		exception.WriteError(c, exception.ErrMissingAuth)
	}
}

func Public() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}
