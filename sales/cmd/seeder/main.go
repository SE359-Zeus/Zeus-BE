package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"

	"zeus-sales-service/internal/repository/sqlite"
	"zeus-sales-service/seeder"
)

func main() {
	defaultDBPath := getenv("SALES_SQLITE_DB", filepath.Join("configs", "sales.db"))

	dbPath := flag.String("db", defaultDBPath, "sqlite database path")
	flag.Parse()

	sqliteRepo, err := sqlite.Open(*dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer sqliteRepo.Close()

	if err := seeder.SeedAll(context.Background(), sqliteRepo); err != nil {
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
