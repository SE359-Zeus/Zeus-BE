package models

import (
	"time"

	"github.com/google/uuid"
)

type PartCatalog struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	PartNumber    string    `gorm:"type:varchar;not null" json:"part_number"`
	PartTypesID   int32     `gorm:"not null" json:"part_types_id"`
	MfgNumber     string    `gorm:"type:varchar;not null" json:"mfg_number"`
	Description   *string   `gorm:"type:text" json:"description,omitempty"`
	PartMfgStatus int32     `gorm:"not null" json:"part_mfg_status"`
	CreatedAt     time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt     time.Time `gorm:"not null" json:"updated_at"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty"`
}
