package main

import (
	"flag"
	"log"

	"zeus-scm-service/internal/repository/sqlite"
	"zeus-scm-service/seeder"
)

func main() {
	dbPath := flag.String("db", "scm.db", "path to the SQLite database file")
	migrationsPath := flag.String("migrations", "migrations", "path to SQL migrations")
	partsDataPath := flag.String("parts-data", "reference/seeder/parts.json", "path to the parts seed data JSON")
	manifestOutPath := flag.String("manifest-out", "seeder/resources/scm-manifest.json", "path to write the generated seed manifest JSON")
	flag.Parse()

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
