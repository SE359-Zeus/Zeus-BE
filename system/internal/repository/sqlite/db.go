package sqlite

import (
	"fmt"
	"log"
	"os"
	"time"
	"zeus-system-service/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewDB(path string) (*gorm.DB, error) {
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		return nil, err
	}

	var journalMode string
	if err := db.Raw("PRAGMA journal_mode=WAL;").Scan(&journalMode).Error; err != nil {
		return nil, fmt.Errorf("failed to enable sqlite WAL: %w", err)
	}
	if journalMode != "wal" {
		return nil, fmt.Errorf("failed to enable sqlite WAL: unexpected journal mode %q", journalMode)
	}

	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
		&models.Session{},
		&models.AuditLog{},
		&models.Role{},
		&models.ActionTypeEntry{},
	)
}
