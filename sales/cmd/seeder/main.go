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

	if err := clearSeedArtifacts(*dbPath); err != nil {
		log.Fatalf("failed to clear previous seed artifacts: %v", err)
	}

	sqliteRepo, err := sqlite.Open(*dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer sqliteRepo.Close()

	if err := seeder.SeedAll(context.Background(), sqliteRepo, *manifestPath); err != nil {
		log.Fatal(err)
	}
}

func clearSeedArtifacts(dbPath string) error {
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

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
