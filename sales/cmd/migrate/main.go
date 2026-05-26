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
	if err := clearArtifacts(dbPath); err != nil {
		return err
	}

	repo, err := sqlite.Open(dbPath)
	if err != nil {
		return err
	}
	defer repo.Close()
	return nil
}

func clearArtifacts(dbPath string) error {
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
