package models

import "github.com/google/uuid"

const (
	ResolutionStatusPlanned      = "planned"
	ResolutionStatusPartial      = "partial"
	ResolutionStatusShortage     = "shortage"
	ResolutionStatusReadyToBuild = "ready_to_build"
)

type ShortageLog struct {
	ID                 uuid.UUID `json:"id"`
	ProductionOrderID  uuid.UUID `json:"production_order_id"`
	PartID             uuid.UUID `json:"part_id"`
	ShortageQty        int       `json:"shortage_qty"`
	ResolutionStatusID int       `json:"resolution_status_id"`
	ResolutionStatus   string    `json:"resolution_status"`
}
