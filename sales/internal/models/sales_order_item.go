package models

import (
	"time"

	"github.com/google/uuid"
)

type SalesOrderItem struct {
	ID           uuid.UUID `json:"id"`
	OrderID      uuid.UUID `json:"order_id"`
	SKU          string    `json:"sku"`
	RequestedQty int       `json:"requested_qty"`
	AllocatedQty int       `json:"allocated_qty"`
	UnitPrice    float64   `json:"unit_price"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
