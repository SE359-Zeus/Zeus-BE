package middleware

import (
	"net/http"
	"time"

	"zeus-scm-service/internal/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func APIKeyAuth(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-KEY")
		if apiKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing X-API-KEY header"})
			return
		}

		if len(apiKey) < 8 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid api key"})
			return
		}
		prefix := apiKey[:8]

		var key models.ApiKey
		if err := db.Where("key_prefix = ? AND active = ? AND deleted_at IS NULL", prefix, true).First(&key).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid api key"})
			return
		}

		if key.ExpiresAt != nil && time.Now().After(*key.ExpiresAt) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "api key expired"})
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(key.KeyHash), []byte(apiKey)); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid api key"})
			return
		}

		now := time.Now()
		db.Model(&key).Update("last_used_at", &now)

		c.Set("api_key_id", key.ID.String())
		c.Set("api_key_name", key.Name)
		c.Next()
	}
}

func Public() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}
