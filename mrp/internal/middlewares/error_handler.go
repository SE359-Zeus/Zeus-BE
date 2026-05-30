package middlewares

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"zeus-mrp-service/internal/infrastructure/observability"
)

var (
	ErrValidation = errors.New("validation error")
	ErrConflict   = errors.New("conflict")
	ErrInternal   = errors.New("internal error")
)

type ResponseEnvelope struct {
	Message    string `json:"message"`
	StatusCode int    `json:"statusCode"`
	Metadata   any    `json:"metadata"`
	Data       any    `json:"data"`
}

type HTTPError struct {
	Status  int    `json:"-"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

func (e *HTTPError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return http.StatusText(e.Status)
}

func (e *HTTPError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func NewHTTPError(status int, code, message string, err error) *HTTPError {
	if message == "" && err != nil {
		message = err.Error()
	}
	return &HTTPError{Status: status, Code: code, Message: message, Err: err}
}

func ErrorHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		logger := requestLoggerFromContext(r)
		slog.InfoContext(r.Context(), "request started",
			slog.String("service", "mrp"),
			slog.String("event", "http_request_started"),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("query", r.URL.RawQuery),
			slog.String("remote_addr", r.RemoteAddr),
			slog.String("user_id", logger.userID),
			slog.String("role", logger.role),
		)

		recorder := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		defer func() {
			if recovered := recover(); recovered != nil {
				observability.DefaultRegistry.Counter(observability.MetricHTTPPanicsTotal).Inc()
				httpError := normalizeError(recovered)
				recorder.Header().Set("Content-Type", "application/json")
				recorder.WriteHeader(httpError.Status)
				_ = json.NewEncoder(recorder).Encode(ResponseEnvelope{
					Message:    httpError.Message,
					StatusCode: httpError.Status,
					Metadata:   map[string]any{"code": httpError.Code},
					Data:       nil,
				})
				slog.ErrorContext(r.Context(), "request failed",
					slog.String("service", "mrp"),
					slog.String("event", "http_request_failed"),
					slog.String("outcome", "panic"),
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.Int("status", httpError.Status),
					slog.Int64("duration_ms", time.Since(start).Milliseconds()),
					slog.String("error", httpError.Message),
					slog.String("user_id", logger.userID),
					slog.String("role", logger.role),
				)
				return
			}
			status := recorder.statusCode
			logFunc := slog.InfoContext
			if status >= 500 {
				logFunc = slog.ErrorContext
				observability.DefaultRegistry.Counter(observability.MetricHTTPRequestErrors).Inc()
			} else if status >= 400 {
				logFunc = slog.WarnContext
			}
			observability.DefaultRegistry.Counter(observability.MetricHTTPRequestsTotal).Inc()
			logFunc(r.Context(), "request completed",
				slog.String("service", "mrp"),
				slog.String("event", "http_request_completed"),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", status),
				slog.Int64("duration_ms", time.Since(start).Milliseconds()),
				slog.String("user_id", logger.userID),
				slog.String("role", logger.role),
			)
		}()
		next.ServeHTTP(recorder, r)
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *loggingResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func requestLoggerFromContext(r *http.Request) struct{ userID, role string } {
	userID, _ := r.Context().Value(ContextKeyUserID).(string)
	role, _ := r.Context().Value(ContextKeyRole).(string)
	if strings.TrimSpace(userID) == "" {
		userID = "anonymous"
	}
	if strings.TrimSpace(role) == "" {
		role = "unknown"
	}
	return struct{ userID, role string }{userID: userID, role: role}
}

func normalizeError(value any) *HTTPError {
	switch err := value.(type) {
	case *HTTPError:
		return err
	case error:
		if errors.Is(err, ErrValidation) {
			return NewHTTPError(http.StatusBadRequest, "validation_error", err.Error(), err)
		}
		if errors.Is(err, ErrConflict) {
			return NewHTTPError(http.StatusConflict, "conflict", err.Error(), err)
		}
		// repository-specific sentinel errors (e.g., ErrNotFound) are not referenced here
		return NewHTTPError(http.StatusInternalServerError, "internal_error", err.Error(), err)
	default:
		return NewHTTPError(http.StatusInternalServerError, "internal_error", http.StatusText(http.StatusInternalServerError), ErrInternal)
	}
}
