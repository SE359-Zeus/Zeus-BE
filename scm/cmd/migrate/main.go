package main

import (
	"log/slog"
	"os"
)

func main() {
	setupLogger()
	slog.Info("migrations disabled — skipping",
		slog.String("service", "scm"),
		slog.String("event", "migration_skipped"),
	)
}

func setupLogger() {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(handler))
}
