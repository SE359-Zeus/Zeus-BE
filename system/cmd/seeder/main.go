package main

import (
	"log"

	"zeus-system-service/internal/repository/sqlite"
	"zeus-system-service/seeder"
)

func main() {
	db, err := sqlite.NewDB("system.db")
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	log.Println("Running migrations...")
	if err := sqlite.ApplyMigrations(db, "./migrations", sqlite.DirectionUp); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	if err := seeder.SeedAll(db); err != nil {
		log.Fatalf("Seeding failed: %v", err)
	}

	log.Println("Seed complete.")
}
