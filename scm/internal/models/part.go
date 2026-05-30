package models

import (
	"time"

	"github.com/google/uuid"
)

type Part struct {
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	PartCatalogID    uuid.UUID  `gorm:"type:uuid;not null" json:"part_catalog_id"`
	ProductID        *uuid.UUID `gorm:"type:uuid" json:"product_id,omitempty"`
	SerialNumber     string     `gorm:"type:varchar;not null" json:"serial_number"`
	PartConditionID  int32      `gorm:"not null" json:"part_condition_id"`
	ManufacturedDate time.Time  `gorm:"not null" json:"manufactured_date"`
	InstallationDate *time.Time `json:"installation_date,omitempty"`
	RemovalDate      *time.Time `json:"removal_date,omitempty"`
	ScrappedDate     *time.Time `json:"scrapped_date,omitempty"`
	CreatedAt        time.Time  `gorm:"not null" json:"created_at"`
	UpdatedAt        time.Time  `gorm:"not null" json:"updated_at"`
	DeletedAt        *time.Time `json:"deleted_at,omitempty"`
}
