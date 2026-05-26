package main

import (
	"flag"
	"log/slog"
	"os"

	"zeus-scm-service/internal/repository/sqlite"
	"zeus-scm-service/seeder"
)

func main() {
	setupLogger()

	dbPath := flag.String("db", "scm.db", "path to the SQLite database file")
	migrationsPath := flag.String("migrations", "migrations", "path to SQL migrations")
	partsDataPath := flag.String("parts-data", "reference/seeder/parts.json", "path to the parts seed data JSON")
	manifestOutPath := flag.String("manifest-out", "seeder/resources/scm-manifest.json", "path to write the generated seed manifest JSON")
	flag.Parse()

	if err := clearSeederArtifacts(*dbPath, *manifestOutPath); err != nil {
		slog.Error("failed to clear previous seed artifacts",
			slog.String("service", "scm"),
			slog.String("event", "startup_failed"),
			slog.String("component", "seeder_artifacts"),
			slog.Any("error", err),
		)
		os.Exit(1)
	}

	db, err := sqlite.NewDB(*dbPath)
	if err != nil {
		slog.Error("failed to connect to database",
			slog.String("service", "scm"),
			slog.String("event", "startup_failed"),
			slog.String("component", "database"),
			slog.Any("error", err),
		)
		os.Exit(1)
	}

	slog.Info("running SQL migrations",
		slog.String("service", "scm"),
		slog.String("event", "migration_started"),
		slog.String("path", *migrationsPath),
	)
	if err := sqlite.RunMigrations(db, *migrationsPath); err != nil {
		slog.Warn("migration warning (might already be up to date)",
			slog.String("service", "scm"),
			slog.String("event", "migration_warning"),
			slog.Any("error", err),
		)
	}

	if err := seeder.SeedAll(db, *partsDataPath, *manifestOutPath); err != nil {
		slog.Error("seeding failed",
			slog.String("service", "scm"),
			slog.String("event", "seed_failed"),
			slog.Any("error", err),
		)
		os.Exit(1)
	}

	slog.Info("seeder process complete",
		slog.String("service", "scm"),
		slog.String("event", "seed_completed"),
	)
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

func setupLogger() {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(handler))
}
