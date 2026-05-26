package observability

// trace.go — W3C-compatible trace / span ID helpers.
//
// Philosophy:
//   - We do NOT import OpenTelemetry SDK at this stage. IDs are generated
//     locally using crypto/rand so the format is already OTel-compatible
//     (128-bit trace ID, 64-bit span ID, hex-encoded). Plugging in a real
//     OTel SDK later only requires swapping the context key extraction.
//   - Every HTTP request gets a fresh span. The trace_id is taken from the
//     incoming "traceparent" header (W3C Trace Context) when present, allowing
//     distributed tracing across services without an SDK.
//   - trace_id and span_id are stored on the request context so both slog
//     (via LogHandler.Handle) and any downstream code can read them.

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ── Context keys ──────────────────────────────────────────────────────────────

type contextKey string

const (
	ctxTraceID contextKey = "trace_id"
	ctxSpanID  contextKey = "span_id"
)

// TraceIDFromContext returns the trace ID stored in ctx, or "".
func TraceIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ctxTraceID).(string); ok {
		return v
	}
	return ""
}

// SpanIDFromContext returns the span ID stored in ctx, or "".
func SpanIDFromContext(ctx context.Context) string {
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

// ── ID generation ─────────────────────────────────────────────────────────────

// NewTraceID generates a random 128-bit (32 hex chars) W3C-compatible trace ID.
func NewTraceID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Extremely unlikely; fall back to timestamp-based ID.
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

// ── W3C traceparent parsing ───────────────────────────────────────────────────

// parseTraceparent tries to extract the trace ID from a W3C "traceparent" header.
// Format: 00-<traceID>-<parentSpanID>-<flags>
// Returns "" if the header is absent or malformed.
func parseTraceparent(header string) string {
	parts := strings.Split(header, "-")
	if len(parts) != 4 || len(parts[1]) != 32 {
		return ""
	}
	return parts[1]
}

// ── Gin middleware ────────────────────────────────────────────────────────────

// Tracing is a Gin middleware that:
//  1. Reads the incoming W3C "traceparent" header to continue an existing trace,
//     or generates a fresh 128-bit trace ID.
//  2. Always generates a new 64-bit span ID for this hop.
//  3. Stores both IDs on the request context (accessible via TraceIDFromContext /
//     SpanIDFromContext) and as Gin context values.
//  4. Emits the "traceparent" response header so callers can propagate the trace.
//  5. Logs the completed span with duration so every request has a root span log.
func Tracing(serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 1. Determine trace ID.
		traceID := parseTraceparent(c.GetHeader("traceparent"))
		if traceID == "" {
			traceID = NewTraceID()
		}
		spanID := NewSpanID()

		// 2. Inject into both Gin and stdlib contexts.
		c.Set(string(ctxTraceID), traceID)
		c.Set(string(ctxSpanID), spanID)

		ctx := WithTraceID(c.Request.Context(), traceID)
		ctx = WithSpanID(ctx, spanID)
		c.Request = c.Request.WithContext(ctx)

		// 3. Propagate downstream via response header.
		c.Header("traceparent", "00-"+traceID+"-"+spanID+"-01")

		c.Next()

		// 4. Emit root span log entry.
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
