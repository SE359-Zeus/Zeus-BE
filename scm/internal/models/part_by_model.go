package models

import (
	"github.com/google/uuid"
)

type PartsByModel struct {
	PartCatalogID    uuid.UUID `gorm:"primaryKey;type:uuid" json:"part_catalog_id"`
	ProductModelCode string    `gorm:"primaryKey;type:varchar" json:"product_model_code"`
	Quantity         int32     `gorm:"not null" json:"quantity"`
	ImageName        *string   `gorm:"type:varchar" json:"image_name"`
}

func (PartsByModel) TableName() string {
	return "parts_by_model"
}
