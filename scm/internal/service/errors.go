package service

import "zeus-be/pkg/exception"

var (
	ErrNotImplemented      = exception.ErrNotImplemented
	ErrLockExpired         = exception.ErrLockExpired
	ErrStateRegression     = exception.ErrStateRegression
	ErrAlreadyLocked       = exception.ErrAlreadyLocked
	ErrMonoVendorViolation = exception.ErrMonoVendorViolation
	ErrInsufficientDeficit = exception.ErrInsufficientDeficit
	ErrNoOptimalSupplier   = exception.ErrNoOptimalSupplier
	ErrAgingQuarantine     = exception.ErrAgingQuarantine
	ErrNotFound            = exception.ErrNotFound
	ErrInvalidTransition   = exception.ErrInvalidTransition
	ErrUnauthorized        = exception.ErrUnauthorized
)
