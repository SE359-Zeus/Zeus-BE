package sqlite

import (
	"fmt"
	"path/filepath"

	gsqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func OpenDatabase(path string) (*gorm.DB, error) {
	dsn := fmt.Sprintf("file:%s?cache=shared", filepath.ToSlash(path))
	db, err := gorm.Open(gsqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	if _, err := sqlDB.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		return nil, err
	}

	return db, nil
}
