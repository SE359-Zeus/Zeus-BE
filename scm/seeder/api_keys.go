package seeder

import (
	"errors"
	"log/slog"

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
	var existingCount int64
	if err := db.Model(&models.ApiKey{}).Count(&existingCount).Error; err == nil && existingCount > 0 {
		return defaultAPIKeyPlaintext
	}

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
			slog.Warn("failed to seed api key",
				slog.String("service", "scm"),
				slog.String("event", "seed_failed"),
				slog.String("component", "api_key"),
				slog.Any("error", err),
			)
		}
	case err != nil:
		slog.Warn("failed to load api key for seeding",
			slog.String("service", "scm"),
			slog.String("event", "seed_failed"),
			slog.String("component", "api_key"),
			slog.Any("error", err),
		)
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
			slog.Warn("failed to update api key seed",
				slog.String("service", "scm"),
				slog.String("event", "seed_failed"),
				slog.String("component", "api_key"),
				slog.Any("error", err),
			)
		}
	}
	return defaultAPIKeyPlaintext
}
