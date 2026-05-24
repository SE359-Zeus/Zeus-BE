package middlewares

import (
	"encoding/json"
	"errors"
	"net/http"

	rootrepo "zeus-sales-service/internal/repository"
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
		defer func() {
			if recovered := recover(); recovered != nil {
				httpError := normalizeError(recovered)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(httpError.Status)
				_ = json.NewEncoder(w).Encode(ResponseEnvelope{
					Message:    httpError.Message,
					StatusCode: httpError.Status,
					Metadata:   map[string]any{"code": httpError.Code},
					Data:       nil,
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
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
		if errors.Is(err, rootrepo.ErrNotFound) {
			return NewHTTPError(http.StatusNotFound, "not_found", err.Error(), err)
		}
		if errors.Is(err, rootrepo.ErrInsufficientInventory) {
			return NewHTTPError(http.StatusConflict, "insufficient_inventory", err.Error(), err)
		}
		return NewHTTPError(http.StatusInternalServerError, "internal_error", err.Error(), err)
	default:
		return NewHTTPError(http.StatusInternalServerError, "internal_error", http.StatusText(http.StatusInternalServerError), ErrInternal)
	}
}
