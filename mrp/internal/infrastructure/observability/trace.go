package observability

// trace.go — W3C-compatible trace / span ID helpers with OpenTelemetry and Tempo.

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// ── Context keys ──────────────────────────────────────────────────────────────

type contextKey string

const (
	ctxTraceID contextKey = "trace_id"
	ctxSpanID  contextKey = "span_id"
)

// TraceIDFromContext returns the trace ID stored in ctx (either from OTel span or custom key).
func TraceIDFromContext(ctx context.Context) string {
	spanContext := trace.SpanContextFromContext(ctx)
	if spanContext.IsValid() {
		return spanContext.TraceID().String()
	}
	if v, ok := ctx.Value(ctxTraceID).(string); ok {
		return v
	}
	return ""
}

// SpanIDFromContext returns the span ID stored in ctx (either from OTel span or custom key).
func SpanIDFromContext(ctx context.Context) string {
	spanContext := trace.SpanContextFromContext(ctx)
	if spanContext.IsValid() {
		return spanContext.SpanID().String()
	}
	if v, ok := ctx.Value(ctxSpanID).(string); ok {
		return v
	}
	return ""
}

// WithTraceID returns a new context carrying the given trace ID.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, ctxTraceID, traceID)
}

// WithSpanID returns a new context carrying the given span ID.
func WithSpanID(ctx context.Context, spanID string) context.Context {
	return context.WithValue(ctx, ctxSpanID, spanID)
}

// ── ID generation (for non-OTel fallback) ──────────────────────────────────────

// NewTraceID generates a random 128-bit (32 hex chars) W3C-compatible trace ID.
func NewTraceID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return hex.EncodeToString([]byte(strings.ReplaceAll(time.Now().Format("20060102150405.999999999"), ".", "")))
	}
	return hex.EncodeToString(b)
}

// NewSpanID generates a random 64-bit (16 hex chars) W3C-compatible span ID.
func NewSpanID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format("20060102150405")))
	}
	return hex.EncodeToString(b)
}

// ── OpenTelemetry Initialization ──────────────────────────────────────────────

// InitTracer initializes an OTLP trace exporter and registers it as the global tracer provider.
func InitTracer(ctx context.Context, serviceName, env, otlpEndpoint string) (*sdktrace.TracerProvider, error) {
	if otlpEndpoint == "" {
		otlpEndpoint = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
		if otlpEndpoint == "" {
			// Return dummy tracer provider with tracecontext propagator so local spans work without exporter
			tp := sdktrace.NewTracerProvider(
				sdktrace.WithSampler(sdktrace.AlwaysSample()),
			)
			otel.SetTracerProvider(tp)
			otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
				propagation.TraceContext{},
				propagation.Baggage{},
			))
			return tp, nil
		}
	}

	cleanEndpoint := otlpEndpoint
	cleanEndpoint = strings.TrimPrefix(cleanEndpoint, "https://")
	cleanEndpoint = strings.TrimPrefix(cleanEndpoint, "http://")
	if idx := strings.Index(cleanEndpoint, "/"); idx != -1 {
		cleanEndpoint = cleanEndpoint[:idx]
	}

	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(cleanEndpoint),
	}
	if !strings.HasPrefix(otlpEndpoint, "https://") {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	exporter, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			semconv.DeploymentEnvironmentKey.String(env),
		),
	)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp, nil
}

// ── Gin middleware ────────────────────────────────────────────────────────────

// Tracing is a Gin middleware that extracts context, starts OTel span, and propagates traceparent.
func Tracing(serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Extract parent context from incoming HTTP headers
		propagator := otel.GetTextMapPropagator()
		ctx := propagator.Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		// Start OTel span
		tracer := otel.Tracer(serviceName)
		ctx, span := tracer.Start(ctx, fmt.Sprintf("%s %s", c.Request.Method, c.Request.URL.Path))
		defer span.End()

		c.Request = c.Request.WithContext(ctx)

		sc := span.SpanContext()
		traceID := sc.TraceID().String()
		spanID := sc.SpanID().String()

		c.Set(string(ctxTraceID), traceID)
		c.Set(string(ctxSpanID), spanID)

		c.Header("traceparent", fmt.Sprintf("00-%s-%s-01", traceID, spanID))

		c.Next()

		duration := time.Since(start)
		slog.InfoContext(ctx, "http span",
			slog.String("service", serviceName),
			slog.String("event", "span_complete"),
			slog.String("trace_id", traceID),
			slog.String("span_id", spanID),
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", c.Writer.Status()),
			slog.Int64("duration_ms", duration.Milliseconds()),
		)
	}
}
