package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"zeus-sales-service/internal/repository/sqlite"
)

func main() {
	dbPath := flag.String("db", filepath.Join("configs", "sales.db"), "sqlite database path")
	flag.Parse()
	if err := run(*dbPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(dbPath string) error {
	repo, err := sqlite.Open(dbPath)
	if err != nil {
		return err
	}
	defer repo.Close()
	return nil
}
