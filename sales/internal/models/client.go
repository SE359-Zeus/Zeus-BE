package models

import (
	"time"

	"github.com/google/uuid"
)

type ClientTier string

const (
	ClientTierB2B ClientTier = "B2B"
	ClientTierB2C ClientTier = "B2C"
)

type Client struct {
	ID                        uuid.UUID  `json:"id"`
	Name                      string     `json:"name"`
	Tier                      ClientTier `json:"tier"`
	DefaultDestinationAddress string     `json:"default_destination_address"`
	TotalLifetimeOrders       int        `json:"total_lifetime_orders"`
	CreatedAt                 time.Time  `json:"created_at"`
	UpdatedAt                 time.Time  `json:"updated_at"`
}
