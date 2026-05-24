package models

import (
	"time"

	"github.com/google/uuid"
)

type SalesOrderStatusLUT struct {
	ID         uuid.UUID `json:"id"`
	Code       string    `json:"code"`
	Label      string    `json:"label"`
	SortOrder  int       `json:"sort_order"`
	IsTerminal bool      `json:"is_terminal"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

const (
	SalesOrderStatusPendingCode    = "PENDING"
	SalesOrderStatusProcessingCode = "PROCESSING"
	SalesOrderStatusDeliveringCode = "DELIVERING"
	SalesOrderStatusCompletedCode  = "COMPLETED"
	SalesOrderStatusCancelledCode  = "CANCELLED"
)

func (status *SalesOrderStatusLUT) CanTransitionTo(next *SalesOrderStatusLUT) bool {
	if status == nil || next == nil {
		return false
	}
	switch status.Code {
	case SalesOrderStatusPendingCode:
		return next.Code == SalesOrderStatusProcessingCode || next.Code == SalesOrderStatusCancelledCode
	case SalesOrderStatusProcessingCode:
		return next.Code == SalesOrderStatusDeliveringCode || next.Code == SalesOrderStatusCancelledCode
	case SalesOrderStatusDeliveringCode:
		return next.Code == SalesOrderStatusCompletedCode
	default:
		return false
	}
}

type SalesOrder struct {
	ID                 uuid.UUID            `json:"id"`
	ClientID           uuid.UUID            `json:"client_id"`
	ClientName         string               `json:"client_name"`
	DestinationAddress string               `json:"destination_address"`
	RequiredDate       time.Time            `json:"required_date"`
	StatusID           uuid.UUID            `json:"status_id"`
	Status             *SalesOrderStatusLUT `json:"status,omitempty"`
	TotalValue         float64              `json:"total_value"`
	Locked             bool                 `json:"locked"`
	CreatedAt          time.Time            `json:"created_at"`
	UpdatedAt          time.Time            `json:"updated_at"`
}
