package models

import (
	"time"

	"github.com/google/uuid"
)

type AllocationQueueEntry struct {
	OrderID      uuid.UUID  `json:"order_id"`
	ClientID     uuid.UUID  `json:"client_id"`
	ClientTier   ClientTier `json:"client_tier"`
	RequiredDate time.Time  `json:"required_date"`
	IngestedAt   time.Time  `json:"ingested_at"`
}
