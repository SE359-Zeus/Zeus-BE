package models

import (
	"time"
)

type ProductModel struct {
	ModelCode   string     `gorm:"type:varchar;primaryKey" json:"model_code"`
	ModelName   string     `gorm:"type:varchar;not null" json:"model_name"`
	Description *string    `gorm:"type:text" json:"description"`
	UnitPrice   float64    `gorm:"column:unit_price;type:real;not null;default:0.0" json:"unit_price"`
	ImageName   *string    `gorm:"type:varchar" json:"image_name"`
	CreatedAt   time.Time  `gorm:"not null" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"not null" json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}
