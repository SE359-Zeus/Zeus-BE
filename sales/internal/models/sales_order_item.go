package models

import (
	"time"

	"github.com/google/uuid"
)

type SalesOrderItem struct {
	ID           uuid.UUID `json:"id"`
	OrderID      uuid.UUID `json:"-"`
	SKU          string    `json:"sku"`
	RequestedQty int       `json:"requested_qty"`
	AllocatedQty int       `json:"-"`
	UnitPrice    float64   `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
