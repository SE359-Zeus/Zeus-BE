package main

import (
	"flag"
	"log"

	"zeus-scm-service/internal/repository/sqlite"
)

func main() {
	dbPath := flag.String("db", "scm.db", "path to the SQLite database file")
	migrationsPath := flag.String("migrations", "migrations", "path to SQL migrations")
	flag.Parse()

	db, err := sqlite.NewDB(*dbPath)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	log.Printf("running SQL migrations from %s", *migrationsPath)
	if err := sqlite.RunMigrations(db, *migrationsPath); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	log.Println("migrations complete")
}
