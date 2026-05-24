package main

import (
	"log"
	"zeus-scm-service/internal/repository/sqlite"
	"zeus-scm-service/seeder"
)

func main() {
	// 1. Initialize DB
	db, err := sqlite.NewDB("scm.db")
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	// 2. Run SQL migrations
	log.Println("Running SQL Migrations...")
	if err := sqlite.RunMigrations(db, "internal/migration"); err != nil {
		log.Printf("Migration warning (might already be up to date): %v", err)
	}

	// 3. Seed Data
	if err := seeder.SeedAll(db); err != nil {
		log.Fatalf("Seeding failed: %v", err)
	}

	log.Println("Process complete.")
}
