package main

import (
	"flag"
	"log"

	"zeus-system-service/internal/config"
	"zeus-system-service/internal/repository/sqlite"
)

func main() {
	direction := flag.String("direction", sqlite.DirectionUp, "migration direction: up or down")
	migrationsDir := flag.String("dir", "./migrations", "path to migrations directory")
	dbPath := flag.String("db", "", "path to the sqlite database file")
	flag.Parse()

	cfg := config.Load()
	if *dbPath != "" {
		cfg.DBPath = *dbPath
	}

	db, err := sqlite.NewDB(cfg.DBPath)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	if err := sqlite.ApplyMigrations(db, *migrationsDir, *direction); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	log.Println("Migration complete.")
}
