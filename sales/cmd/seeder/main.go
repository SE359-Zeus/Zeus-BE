package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"

	"zeus-sales-service/config"
	"zeus-sales-service/internal/repository/sqlite"
	"zeus-sales-service/seeder"
)

func main() {
	defaultDBPath := config.Load().SQLiteDBPath
	if defaultDBPath == "" {
		defaultDBPath = filepath.Join("configs", "sales.db")
	}
	defaultManifestPath := getenv("SCM_MANIFEST_PATH", filepath.Join(".", "scm-manifest.json"))

	dbPath := flag.String("db", defaultDBPath, "sqlite database path")
	manifestPath := flag.String("manifest", defaultManifestPath, "path to the SCM seed manifest JSON")
	flag.Parse()

	sqliteRepo, err := sqlite.Open(*dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer sqliteRepo.Close()

	if err := seeder.SeedAll(context.Background(), sqliteRepo, *manifestPath); err != nil {
		log.Fatal(err)
	}
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
