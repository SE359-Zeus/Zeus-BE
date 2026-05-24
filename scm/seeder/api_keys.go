package seeder

import (
	"errors"
	"log"

	"zeus-scm-service/internal/models"

	"gorm.io/gorm"
)

const (
	defaultAPIKeyName      = "Default SCM API Key"
	defaultAPIKeyPrefix    = "scmkey01"
	defaultAPIKeyPlaintext = "scmkey01-admin-20260524"
	defaultAPIKeyHash      = "$2a$10$w6KpSUPBMyQfgIMjcmKi5uS6sJisStIBfIGOHtJeQn0dnwLXfJhW2"
)

func seedAPIKeys(db *gorm.DB) string {
	apiKey := models.ApiKey{}
	err := db.Unscoped().Where("key_prefix = ?", defaultAPIKeyPrefix).First(&apiKey).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		apiKey = models.ApiKey{
			ID:        stableUUID("api-key:default-scm"),
			Name:      defaultAPIKeyName,
			KeyPrefix: defaultAPIKeyPrefix,
			KeyHash:   defaultAPIKeyHash,
			Active:    true,
		}
		if err := db.Create(&apiKey).Error; err != nil {
			log.Printf("warning: failed to seed api key: %v", err)
		}
	case err != nil:
		log.Printf("warning: failed to load api key for seeding: %v", err)
	default:
		updates := map[string]any{
			"name":         defaultAPIKeyName,
			"key_prefix":   defaultAPIKeyPrefix,
			"key_hash":     defaultAPIKeyHash,
			"active":       true,
			"expires_at":   nil,
			"last_used_at": nil,
			"deleted_at":   nil,
		}
		if err := db.Unscoped().Model(&apiKey).Updates(updates).Error; err != nil {
			log.Printf("warning: failed to update api key seed: %v", err)
		}
	}
	return defaultAPIKeyPlaintext
}
