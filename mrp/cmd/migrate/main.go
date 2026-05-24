package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/file"

	"zeus-mrp-service/internal/models"
	reposqlite "zeus-mrp-service/internal/repository/sqlite"
)

func main() {
	dbPath := flag.String("db", "mrp.db", "path to sqlite db file")
	migrations := flag.String("migrations", "migrations", "path to migrations directory")
	flag.Parse()

	// open gorm DB
	db, err := reposqlite.OpenDatabase(*dbPath)
	if err != nil {
		log.Fatalf("failed to open sqlite db: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("failed to get sql DB: %v", err)
	}

	driver, err := sqlite3.WithInstance(sqlDB, &sqlite3.Config{})
	if err != nil {
		log.Fatalf("failed to create sqlite driver: %v", err)
	}

	absPath, err := filepath.Abs(*migrations)
	if err != nil {
		log.Printf("failed to resolve migrations path: %v", err)
	}

	// Try file-based migrations first
	if absPath != "" {
		if fi, err := os.Stat(absPath); err == nil && fi.IsDir() {
			fileURL := "file:///" + strings.TrimRight(filepath.ToSlash(absPath), "/") + "/"
			log.Printf("migrate: attempting file migrations path=%s fileURL=%s", absPath, fileURL)
			src, err := (&file.File{}).Open(fileURL)
			if err == nil {
				m, err := migrate.NewWithInstance("file", src, "sqlite3", driver)
				if err == nil {
					if err := m.Up(); err == nil || err == migrate.ErrNoChange {
						log.Printf("migrations applied from %s to %s", absPath, *dbPath)
						return
					}
					log.Printf("migrate: file migrations Up() returned error: %v", err)
				} else {
					log.Printf("migrate: failed to create migrate instance: %v", err)
				}
			} else {
				log.Printf("migrate: failed to open migrations source: %v", err)
			}
		} else {
			log.Printf("migrate: migrations path missing or not dir: %s", absPath)
		}
	}

	// Fallback: use GORM AutoMigrate to ensure schema
	log.Printf("migrate: falling back to gorm AutoMigrate")
	if err := db.AutoMigrate(
		&models.ProductionOrder{},
		&models.BomEntry{},
		&models.ShortageLog{},
	); err != nil {
		log.Fatalf("auto-migrate failed: %v", err)
	}
	log.Printf("gorm AutoMigrate applied to %s", *dbPath)
}
