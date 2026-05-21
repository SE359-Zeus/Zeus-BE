package sqlite

import (
	"zeus-system-service/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func NewDB(path string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
		&models.AuditLog{},
		&models.Role{},
		&models.ActionTypeEntry{},
		&models.EndpointRole{},
	)
}
