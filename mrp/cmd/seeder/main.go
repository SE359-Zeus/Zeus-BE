package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"

	reposqlite "zeus-mrp-service/internal/repository/sqlite"
	"zeus-mrp-service/seeder"
)

func main() {
	defaultDBPath := getenv("MRP_DB_PATH", filepath.Join(".", "mrp.db"))
	dbPath := flag.String("db", defaultDBPath, "sqlite database path")
	defaultManifestPath := getenv("SCM_MANIFEST_PATH", filepath.Join(".", "contracts", "scm-manifest.json"))
	manifestPath := flag.String("manifest", defaultManifestPath, "path to the SCM seed manifest JSON")
	flag.Parse()

	if err := clearSeedArtifacts(*dbPath); err != nil {
		log.Fatalf("failed to clear previous seed artifacts: %v", err)
	}

	db, err := reposqlite.OpenDatabase(*dbPath)
	if err != nil {
		log.Fatal(err)
	}

	repo := reposqlite.NewSqliteMRPRepository(db)
	if err := seeder.SeedAll(context.Background(), repo, *manifestPath); err != nil {
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
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
