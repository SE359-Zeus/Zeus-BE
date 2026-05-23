package models

import (
	"time"

	"github.com/google/uuid"
)

type ReservationItem struct {
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
}

type InventoryReservation struct {
	ID         uuid.UUID         `json:"id"`
	OrderID    uuid.UUID         `json:"order_id"`
	Items      []ReservationItem `json:"items"`
	ReservedAt time.Time         `json:"reserved_at"`
}

type FulfillmentManifest struct {
	OrderID            uuid.UUID         `json:"order_id"`
	ClientID           uuid.UUID         `json:"client_id"`
	DestinationAddress string            `json:"destination_address"`
	Items              []ReservationItem `json:"items"`
	GeneratedAt        time.Time         `json:"generated_at"`
}
