package main

import (
	"log/slog"
	"os"
)

func main() {
	setupLogger()
	slog.Info("seeder disabled — skipping",
		slog.String("service", "scm"),
		slog.String("event", "seed_skipped"),
	)
}

func setupLogger() {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(handler))
}
