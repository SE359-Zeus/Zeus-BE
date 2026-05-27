package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"zeus-scm-service/internal/exception"

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

		attrs := []any{
			slog.String("service", "scm"),
			slog.String("event", "request_complete"),
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", status),
			slog.Int64("duration_ms", time.Since(start).Milliseconds()),
			slog.String("client_ip", c.ClientIP()),
		}
		if size := c.Writer.Size(); size >= 0 {
			attrs = append(attrs, slog.Int("response_bytes", size))
		}
		if v, ok := c.Get(ContextKeyUserID); ok {
			attrs = append(attrs, slog.Any("user_id", v))
		}
		if v, ok := c.Get(ContextKeyRole); ok {
			attrs = append(attrs, slog.Any("role", v))
		}
		if v, ok := c.Get(ContextKeyAuthMethod); ok {
			attrs = append(attrs, slog.Any("auth_method", v))
		}
		logFunc := slog.InfoContext
		if status >= 500 {
			logFunc = slog.ErrorContext
		} else if status >= 400 {
			logFunc = slog.WarnContext
		}
		logFunc(c.Request.Context(), "request complete", attrs...)
	}
}

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				slog.ErrorContext(c.Request.Context(), "panic recovered",
					slog.String("service", "scm"),
					slog.String("event", "panic_recovered"),
					slog.String("method", c.Request.Method),
					slog.String("path", c.Request.URL.Path),
					slog.Any("panic", r),
				)
				exception.WriteError(c, exception.ErrPanic)
			}
		}()
		c.Next()
	}
}
