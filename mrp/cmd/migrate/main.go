package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/file"

	reposqlite "zeus-mrp-service/internal/repository/sqlite"
)

func main() {
	dbPath := flag.String("db", "mrp.db", "path to sqlite db file")
	migrations := flag.String("migrations", "migrations", "path to migrations directory")
	flag.Parse()

	if *migrations == "" {
		log.Fatalf("migrate: migrations directory path is required")
	}

	if err := clearArtifacts(*dbPath); err != nil {
		log.Fatalf("migrate: failed to clear previous db artifacts: %v", err)
	}

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
		log.Fatalf("failed to resolve migrations path: %v", err)
	}

	fi, err := os.Stat(absPath)
	if err != nil {
		log.Fatalf("migrate: migrations directory stat failed: %v", err)
	}
	if !fi.IsDir() {
		log.Fatalf("migrate: migrations path is not a directory: %s", absPath)
	}

	fileURL := "file://" + filepath.ToSlash(absPath)
	log.Printf("migrate: attempting file migrations path=%s fileURL=%s", absPath, fileURL)

	src, err := (&file.File{}).Open(fileURL)
	if err != nil {
		log.Fatalf("migrate: failed to open migrations source: %v", err)
	}

	m, err := migrate.NewWithInstance("file", src, "sqlite3", driver)
	if err != nil {
		log.Fatalf("migrate: failed to create migrate instance: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("migrate: file migrations Up() returned error: %v", err)
	}

	log.Printf("migrations applied successfully from %s to %s", absPath, *dbPath)
}

func clearArtifacts(dbPath string) error {
	paths := []string{dbPath}
	if dbPath != "" && dbPath != ":memory:" && dbPath != "file::memory:" {
		paths = append(paths, dbPath+"-wal", dbPath+"-shm")
	}

	for _, path := range paths {
		if path == "" {
			continue
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	return nil
}
