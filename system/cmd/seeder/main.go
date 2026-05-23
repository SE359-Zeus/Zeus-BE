package main

import (
	"flag"
	"log"

	"zeus-system-service/internal/config"
	"zeus-system-service/internal/repository/sqlite"
	"zeus-system-service/seeder"
)

func main() {
	dbPath := flag.String("db", "", "path to the sqlite database file")
	migrationsDir := flag.String("dir", "./migrations", "path to migrations directory")
	flag.Parse()

	cfg := config.Load()
	if *dbPath != "" {
		cfg.DBPath = *dbPath
	}

	db, err := sqlite.NewDB(cfg.DBPath)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	log.Println("Running migrations...")
	if err := sqlite.ApplyMigrations(db, *migrationsDir, sqlite.DirectionUp); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	if err := seeder.SeedAll(db); err != nil {
		log.Fatalf("Seeding failed: %v", err)
	}

	log.Println("Seed complete.")
}
