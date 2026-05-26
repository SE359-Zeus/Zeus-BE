package main

import (
	"flag"
	"log"
	"os"

	"zeus-scm-service/internal/repository/sqlite"
	"zeus-scm-service/seeder"
)

func main() {
	dbPath := flag.String("db", "scm.db", "path to the SQLite database file")
	migrationsPath := flag.String("migrations", "migrations", "path to SQL migrations")
	partsDataPath := flag.String("parts-data", "reference/seeder/parts.json", "path to the parts seed data JSON")
	manifestOutPath := flag.String("manifest-out", "seeder/resources/scm-manifest.json", "path to write the generated seed manifest JSON")
	flag.Parse()

	if err := clearSeederArtifacts(*dbPath, *manifestOutPath); err != nil {
		log.Fatalf("failed to clear previous seed artifacts: %v", err)
	}

	db, err := sqlite.NewDB(*dbPath)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	log.Println("Running SQL Migrations...")
	if err := sqlite.RunMigrations(db, *migrationsPath); err != nil {
		log.Printf("Migration warning (might already be up to date): %v", err)
	}

	if err := seeder.SeedAll(db, *partsDataPath, *manifestOutPath); err != nil {
		log.Fatalf("Seeding failed: %v", err)
	}

	log.Println("Process complete.")
}

func clearSeederArtifacts(dbPath string, manifestOutPath string) error {
	paths := []string{dbPath}
	if dbPath != "" && dbPath != ":memory:" && dbPath != "file::memory:" {
		paths = append(paths, dbPath+"-wal", dbPath+"-shm")
	}
	if manifestOutPath != "" {
		paths = append(paths, manifestOutPath)
	}

	for _, path := range paths {
		if err := removeIfExists(path); err != nil {
			return err
		}
	}

	return nil
}

func removeIfExists(path string) error {
	if path == "" {
		return nil
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}
