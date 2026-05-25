package service

import "zeus-be/pkg/exception"

var (
	ErrNotFound        = exception.ErrNotFound
	ErrDuplicateEmail  = exception.ErrDuplicateEmail
	ErrInactiveAccount = exception.ErrInactiveAccount
	ErrUnauthorized    = exception.ErrUnauthorized
	ErrForbidden       = exception.ErrForbidden
	ErrInvalidInput    = exception.ErrInvalidInput
	ErrShortPassword   = exception.ErrShortPassword
	ErrInvalidRole     = exception.ErrInvalidRole
	ErrInvalidEmail    = exception.ErrInvalidEmail
	ErrEmptyEmail      = exception.ErrEmptyEmail
	ErrEmptyPassword   = exception.ErrEmptyPassword
	ErrEmptyName       = exception.ErrEmptyName
	ErrNilID           = exception.ErrNilID
)
