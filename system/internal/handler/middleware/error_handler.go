package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"zeus-system-service/internal/handler"
	"zeus-system-service/internal/infrastructure/observability"

	"zeus-be/pkg/exception"

	"github.com/gin-gonic/gin"
)

type loggingResponseWriter struct {
	gin.ResponseWriter
	status int
}

func (w *loggingResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *loggingResponseWriter) WriteHeaderNow() {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	w.ResponseWriter.WriteHeaderNow()
}

func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		writer := &loggingResponseWriter{ResponseWriter: c.Writer, status: http.StatusOK}
		c.Writer = writer

		c.Next()

		status := writer.status
		if status == 0 {
			status = c.Writer.Status()
		}

		userID := "none"
		role := "unknown"
		authMethod := "none"
		if v, ok := c.Get(ContextKeyUserID); ok {
			if s, ok := v.(string); ok && s != "" {
				userID = s
			}
		}
		if v, ok := c.Get(ContextKeyRole); ok {
			if s, ok := v.(string); ok && s != "" {
				role = s
			}
		}
		if v, ok := c.Get(ContextKeyAuthMethod); ok {
			if s, ok := v.(string); ok && s != "" {
				authMethod = s
			}
		}
		if authMethod == "none" && c.GetHeader("X-API-KEY") != "" {
			userID = "none"
			role = "apikey"
			authMethod = "api_key"
		}

		slog.Info("request complete",
			slog.String("service", "system"),
			slog.String("event", "request_complete"),
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", status),
			slog.Int64("duration_ms", time.Since(start).Milliseconds()),
			slog.String("client_ip", c.ClientIP()),
			slog.String("user_id", userID),
			slog.String("role", role),
			slog.String("auth_method", authMethod),
		)
		observability.DefaultRegistry.Counter(observability.MetricHTTPRequestsTotal).Inc()
		if status >= 500 {
			observability.DefaultRegistry.Counter(observability.MetricHTTPRequestErrors).Inc()
		}
	}
}

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				observability.DefaultRegistry.Counter(observability.MetricHTTPPanicsTotal).Inc()
				slog.Error("panic recovered",
					slog.String("service", "system"),
					slog.String("event", "panic_recovered"),
					slog.String("method", c.Request.Method),
					slog.String("path", c.Request.URL.Path),
					slog.Any("panic", r),
				)
				handler.WriteAppError(c, exception.ErrPanic)
			}
		}()
		c.Next()
	}
}
