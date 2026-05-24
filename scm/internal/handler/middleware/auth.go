package middleware

import (
	"strings"
	"time"

	"zeus-scm-service/internal/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"zeus-be/pkg/exception"
	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/service"
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

		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				exception.WriteError(c, exception.ErrInvalidAuthHeader)
				return
			}

			claims, err := jwtSvc.VerifyAccessToken(parts[1])
			if err != nil {
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

		if apiKey != "" {
			if len(apiKey) < 8 {
				exception.WriteError(c, exception.ErrAPIKeyInvalid)
				return
			}
			prefix := apiKey[:8]

			var key models.ApiKey
			if err := db.Where("key_prefix = ? AND active = ? AND deleted_at IS NULL", prefix, true).First(&key).Error; err != nil {
				exception.WriteError(c, exception.ErrAPIKeyInvalid)
				return
			}

			if key.ExpiresAt != nil && time.Now().After(*key.ExpiresAt) {
				exception.WriteError(c, exception.ErrAPIKeyExpired)
				return
			}

			if err := bcrypt.CompareHashAndPassword([]byte(key.KeyHash), []byte(apiKey)); err != nil {
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

		exception.WriteError(c, exception.ErrMissingAuth)
	}
}

func Public() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}
