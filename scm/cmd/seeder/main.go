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

	dbPath := flag.String("db", "scm.db", "Path to SQLite database")
	backfill := flag.Bool("backfill", false, "Backfill parts for products that have no parts")
	flag.Parse()

	if *backfill {
		slog.Info("running backfill", slog.String("service", "scm"), slog.String("db", *dbPath))
		db, err := sqlite.NewDB(*dbPath)
		if err != nil {
			slog.Error("failed to open database", slog.Any("error", err))
			os.Exit(1)
		}
		if err := seeder.BackfillPartsForEmptyProducts(db); err != nil {
			slog.Error("backfill failed", slog.Any("error", err))
			os.Exit(1)
		}
		return
	}

	slog.Info("seeder disabled — skipping",
		slog.String("service", "scm"),
		slog.String("event", "seed_skipped"),
	)
}

func setupLogger() {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(handler))
}
