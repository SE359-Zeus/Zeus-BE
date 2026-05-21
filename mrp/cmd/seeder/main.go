package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"

	reposqlite "zeus-mrp-service/internal/repository/sqlite"
	"zeus-mrp-service/seeder"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	defaultDBPath := getenv("MRP_DB_PATH", filepath.Join(".", "mrp.db"))
	dbPath := flag.String("db", defaultDBPath, "sqlite database path")
	flag.Parse()

	db, err := gorm.Open(sqlite.Open(*dbPath), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	repo := reposqlite.NewSqliteMRPRepository(db)
	if err := seeder.SeedAll(context.Background(), repo); err != nil {
		log.Fatal(err)
	}
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
