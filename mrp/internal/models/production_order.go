package models

import (
	"time"

	"github.com/google/uuid"
)

type ProductionOrderStatus string

const (
	StatusPlanned      ProductionOrderStatus = "PLANNED"
	StatusClearToBuild ProductionOrderStatus = "CLEAR_TO_BUILD"
	StatusPartial      ProductionOrderStatus = "PARTIAL"
	StatusShortage     ProductionOrderStatus = "SHORTAGE"
)

type ProductionOrder struct {
	ID                uuid.UUID             `json:"id"`
	ProductModelCode  string                `json:"product_model_code"`
	TargetQuantity    int                   `json:"target_quantity"`
	Status            ProductionOrderStatus `json:"status"`
	ScheduledAt       time.Time             `json:"scheduled_at"`
	CreatedAt         time.Time             `json:"created_at"`
}
