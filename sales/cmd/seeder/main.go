package main

import (
	"context"
	"flag"
	"log"
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

	dbPath := flag.String("db", defaultDBPath, "sqlite database path")
	flag.Parse()

	sqliteRepo, err := sqlite.Open(*dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer sqliteRepo.Close()

	if err := seeder.SeedAll(context.Background(), sqliteRepo); err != nil {
		log.Fatal(err)
	}
}
