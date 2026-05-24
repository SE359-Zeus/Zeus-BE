package sqlite

import (
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/file"
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

func RunMigrations(db *gorm.DB, migrationsPath string) error {
	if err := dropLegacySchema(db); err != nil {
		return err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	driver, err := sqlite3.WithInstance(sqlDB, &sqlite3.Config{})
	if err != nil {
		return err
	}
	src, err := (&file.File{}).Open("file://" + migrationsPath)
	if err != nil {
		return err
	}
	m, err := migrate.NewWithInstance("file", src, "sqlite3", driver)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}

func dropLegacySchema(db *gorm.DB) error {
	statements := []string{
		"DROP TABLE IF EXISTS parts_by_models",
		"DROP TABLE IF EXISTS endpoint_roles",
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}
