package main

import (
	"flag"
	"log/slog"
	"os"

	"zeus-scm-service/internal/repository/sqlite"
)

func main() {
	setupLogger()

	dbPath := flag.String("db", "scm.db", "path to the SQLite database file")
	migrationsPath := flag.String("migrations", "migrations", "path to SQL migrations")
	flag.Parse()

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
		slog.Error("migration failed",
			slog.String("service", "scm"),
			slog.String("event", "migration_failed"),
			slog.Any("error", err),
		)
		os.Exit(1)
	}

	slog.Info("migrations complete",
		slog.String("service", "scm"),
		slog.String("event", "migration_completed"),
	)
}

func setupLogger() {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(handler))
}
