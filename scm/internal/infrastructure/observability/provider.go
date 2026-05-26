package observability

// provider.go — single bootstrap entry point for the observability stack.
//
// Usage in main():
//
//	cfg := config.Load()
//	obs, shutdown := observability.Setup(observability.Config{...})
//	defer shutdown()
//	slog.SetDefault(obs.Logger)

import (
	"log/slog"
	"os"
	"time"

	slogmulti "github.com/samber/slog-multi"
)

// Config holds all observability settings read from environment / config.
type Config struct {
	// ServiceName is used as the slog "service" label and the Alloy job label.
	ServiceName string

	// Env is the deployment environment label (e.g. "production", "staging").
	// Defaults to "production" when empty.
	Env string

	// Alloy log-push settings.
	// AlloyURL is the Alloy Loki-receiver endpoint, e.g.
	//   http://alloy:3100/loki/api/v1/push          (self-hosted)
	//   https://logs-prod-XXX.grafana.net/loki/api/v1/push  (Grafana Cloud direct)
	// Leave AlloyURL empty to disable remote push (stdout-only mode).
	AlloyURL      string
	AlloyUsername string // Grafana Cloud user ID — empty for unauthenticated Alloy
	AlloyPassword string // Grafana Cloud API token — empty for unauthenticated Alloy

	// LogLevel is the minimum level for both handlers (default: Info).
	LogLevel slog.Level
}

// Provider is the live observability state. Hold it for the process lifetime
// and call shutdown() on exit.
type Provider struct {
	// Logger is the configured slog.Logger (tee: stdout + Alloy when enabled).
	Logger *slog.Logger

	// Metrics is the service-wide metric registry.
	Metrics *Registry

	stopLog func() // non-nil only when Alloy log-push is active
}

// Setup initialises the observability stack and returns a Provider plus a
// shutdown function. Always call shutdown() before the process exits.
func Setup(cfg Config) (p *Provider, shutdown func()) {
	if cfg.Env == "" {
		cfg.Env = "production"
	}
	if cfg.ServiceName == "" {
		cfg.ServiceName = "zeus"
	}

	// ── stdout handler (always active) ───────────────────────────────────────
	level := cfg.LogLevel
	if level == 0 {
		level = slog.LevelInfo
	}
	stdoutH := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})

	// ── Alloy log-push handler (optional) ────────────────────────────────────
	var stopLog func()
	var activeHandler slog.Handler = stdoutH

	if cfg.AlloyURL != "" {
		logH, stop := NewLogHandler(LogHandlerOptions{
			PushURL:       cfg.AlloyURL,
			BasicAuthUser: cfg.AlloyUsername,
			BasicAuthPass: cfg.AlloyPassword,
			Labels: map[string]string{
				"job": cfg.ServiceName,
				"env": cfg.Env,
			},
			Level:         level,
			BatchSize:     200,
			FlushInterval: 5 * time.Second,
		})
		stopLog = stop
		activeHandler = slogmulti.Fanout(stdoutH, logH)
	}

	logger := slog.New(activeHandler)

	// ── Metrics registry ──────────────────────────────────────────────────────
	registry := NewRegistry()
	DefaultRegistry = registry // expose as package-level singleton

	p = &Provider{
		Logger:  logger,
		Metrics: registry,
		stopLog: stopLog,
	}

	shutdown = func() {
		if p.stopLog != nil {
			p.stopLog()
		}
	}

	// Emit a startup log so the first record in Grafana shows config.
	if cfg.AlloyURL != "" {
		logger.Info("observability ready",
			slog.String("service", cfg.ServiceName),
			slog.String("event", "observability_setup"),
			slog.String("alloy_push", "enabled"),
			slog.String("alloy_url", cfg.AlloyURL),
			slog.String("env", cfg.Env),
		)
	} else {
		logger.Info("observability ready",
			slog.String("service", cfg.ServiceName),
			slog.String("event", "observability_setup"),
			slog.String("alloy_push", "disabled — stdout only"),
			slog.String("env", cfg.Env),
		)
	}

	return p, shutdown
}
