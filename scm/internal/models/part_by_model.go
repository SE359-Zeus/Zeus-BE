package models

import (
	"github.com/google/uuid"
)

type PartsByModel struct {
	PartCatalogID    uuid.UUID `gorm:"primaryKey;type:uuid"`
	ProductModelCode string    `gorm:"primaryKey;type:varchar"`
	Quantity         int32     `gorm:"not null"`
}

func (PartsByModel) TableName() string {
	return "parts_by_model"
}
