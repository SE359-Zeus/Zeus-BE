package exception

import (
	"errors"
	"fmt"
)

type AppError struct {
	Code       string `json:"error_code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"-"`
	Err        error  `json:"-"`
}

func (e *AppError) Error() string {
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func (e *AppError) Is(target error) bool {
	var other *AppError
	if !errors.As(target, &other) {
		return false
	}
	return e.Code == other.Code
}

func New(code string, message string, httpStatus int) *AppError {
	return &AppError{Code: code, Message: message, HTTPStatus: httpStatus}
}

func Wrap(code string, message string, httpStatus int, err error) *AppError {
	return &AppError{Code: code, Message: message, HTTPStatus: httpStatus, Err: err}
}

func (e *AppError) WithMessage(msg string) *AppError {
	clone := *e
	clone.Message = msg
	return &clone
}

func (e *AppError) WithError(err error) *AppError {
	clone := *e
	clone.Err = err
	return &clone
}

func (e *AppError) formatMessage(args ...interface{}) *AppError {
	clone := *e
	clone.Message = fmt.Sprintf(e.Message, args...)
	return &clone
}
