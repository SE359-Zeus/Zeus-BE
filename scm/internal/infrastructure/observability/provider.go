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
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	slogmulti "github.com/samber/slog-multi"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
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
	tp      *sdktrace.TracerProvider
}

type contextInjectHandler struct {
	slog.Handler
}

func (h contextInjectHandler) Handle(ctx context.Context, r slog.Record) error {
	if traceID := TraceIDFromContext(ctx); traceID != "" {
		r.AddAttrs(slog.String("trace_id", traceID))
	}
	if spanID := SpanIDFromContext(ctx); spanID != "" {
		r.AddAttrs(slog.String("span_id", spanID))
	}
	return h.Handler.Handle(ctx, r)
}

func (h contextInjectHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return contextInjectHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h contextInjectHandler) WithGroup(name string) slog.Handler {
	return contextInjectHandler{Handler: h.Handler.WithGroup(name)}
}// Setup initialises the observability stack and returns a Provider plus a
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
	var activeHandler slog.Handler = contextInjectHandler{Handler: stdoutH}

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
		activeHandler = slogmulti.Fanout(contextInjectHandler{Handler: stdoutH}, logH)
	}

	logger := slog.New(activeHandler)

	// ── Metrics registry ──────────────────────────────────────────────────────
	registry := NewRegistry()
	DefaultRegistry = registry // expose as package-level singleton

	// Pre-register all known counters so they appear in /metrics at zero
	registry.Counter(MetricHTTPRequestsTotal)
	registry.Counter(MetricHTTPRequestErrors)
	registry.Counter(MetricHTTPPanicsTotal)
	registry.Counter(MetricPOCreated)
	registry.Counter(MetricPOStateTransitions)
	registry.Counter(MetricGRCreated)
	registry.Counter(MetricGRProcessed)
	registry.Counter(MetricShipmentDispatched)
	registry.Counter(MetricShipmentDelivered)
	registry.Counter(MetricLockAcquisitions)
	registry.Counter(MetricLockContention)

	// Set up Tracer
	var otlpEP string
	if val := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); val != "" {
		otlpEP = val
	} else if cfg.AlloyURL != "" {
		clean := cfg.AlloyURL
		clean = strings.TrimPrefix(clean, "https://")
		clean = strings.TrimPrefix(clean, "http://")
		if idx := strings.Index(clean, ":"); idx != -1 {
			clean = clean[:idx]
		} else if idx := strings.Index(clean, "/"); idx != -1 {
			clean = clean[:idx]
		}
		if clean != "" {
			if strings.HasPrefix(cfg.AlloyURL, "https://") {
				otlpEP = "https://" + clean + ":4318"
			} else {
				otlpEP = "http://" + clean + ":4318"
			}
		}
	}

	tp, err := InitTracer(context.Background(), cfg.ServiceName, cfg.Env, otlpEP)
	if err != nil {
		slog.Error("failed to initialize tracer provider", "error", err)
	}

	p = &Provider{
		Logger:  logger,
		Metrics: registry,
		stopLog: stopLog,
		tp:      tp,
	}

	shutdown = func() {
		if p.stopLog != nil {
			p.stopLog()
		}
		if p.tp != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = p.tp.Shutdown(ctx)
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
